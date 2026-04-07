package index

import (
	"context"
	"errors"
	"fmt"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &virtualIndexResource{}
	_ resource.ResourceWithConfigure   = &virtualIndexResource{}
	_ resource.ResourceWithImportState = &virtualIndexResource{}
)

type virtualIndexResource struct {
	client *search.APIClient
}

func NewVirtualResource() resource.Resource {
	return &virtualIndexResource{}
}

func (r *virtualIndexResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_index"
}

func (r *virtualIndexResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = virtualIndexResourceSchema()
}

func (r *virtualIndexResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*providertypes.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *providertypes.ProviderData, got: %T", req.ProviderData),
		)
		return
	}

	r.client = data.Client
}

func (r *virtualIndexResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan VirtualIndexResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := plan.Name.ValueString()
	primaryIndexName := plan.PrimaryIndexName.ValueString()
	tflog.Debug(ctx, "Creating virtual index", map[string]any{"name": indexName, "primary_index_name": primaryIndexName})

	if err := ensureVirtualReplicaLinked(r.client, primaryIndexName, indexName); err != nil {
		resp.Diagnostics.AddError("Error linking virtual replica", "Could not link virtual replica "+indexName+" to primary index "+primaryIndexName+": "+err.Error())
		return
	}

	indexPlan := virtualToIndexModel(plan)
	nullBlocks := captureNullBlocks(&indexPlan)
	settings, diags := expandIndexSettings(ctx, &indexPlan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	setResp, err := r.client.SetSettings(r.client.NewApiSetSettingsRequest(indexName, settings))
	if err != nil {
		resp.Diagnostics.AddError("Error creating virtual index", "Could not configure virtual index "+indexName+": "+err.Error())
		return
	}

	if err = waitForIndexTask(r.client, indexName, setResp.TaskID); err != nil {
		resp.Diagnostics.AddError("Error waiting for virtual index creation", "Could not wait for task: "+err.Error())
		return
	}

	planSnapshot := indexPlan
	diags = r.readVirtualIndex(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexState := virtualToIndexModel(plan)
	resp.Diagnostics.Append(preservePlannedValues(ctx, &planSnapshot, &indexState)...)
	virtualFromIndexModel(indexState, &plan)
	restoreVirtualNullBlocks(&plan, nullBlocks)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *virtualIndexResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state VirtualIndexResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexState := virtualToIndexModel(state)
	nullBlocks := captureNullBlocks(&indexState)
	stateSnapshot := indexState

	diags := r.readVirtualIndex(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexRead := virtualToIndexModel(state)
	resp.Diagnostics.Append(preservePlannedValues(ctx, &stateSnapshot, &indexRead)...)
	virtualFromIndexModel(indexRead, &state)
	restoreVirtualNullBlocks(&state, nullBlocks)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *virtualIndexResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan VirtualIndexResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := plan.Name.ValueString()
	primaryIndexName := plan.PrimaryIndexName.ValueString()
	if err := ensureVirtualReplicaLinked(r.client, primaryIndexName, indexName); err != nil {
		resp.Diagnostics.AddError("Error linking virtual replica", "Could not confirm virtual replica linkage: "+err.Error())
		return
	}

	indexPlan := virtualToIndexModel(plan)
	nullBlocks := captureNullBlocks(&indexPlan)
	settings, diags := expandIndexSettings(ctx, &indexPlan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	setResp, err := r.client.SetSettings(r.client.NewApiSetSettingsRequest(indexName, settings))
	if err != nil {
		resp.Diagnostics.AddError("Error updating virtual index", "Could not update virtual index "+indexName+": "+err.Error())
		return
	}

	if err = waitForIndexTask(r.client, indexName, setResp.TaskID); err != nil {
		resp.Diagnostics.AddError("Error waiting for virtual index update", "Could not wait for task: "+err.Error())
		return
	}

	planSnapshot := indexPlan
	diags = r.readVirtualIndex(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexState := virtualToIndexModel(plan)
	resp.Diagnostics.Append(preservePlannedValues(ctx, &planSnapshot, &indexState)...)
	virtualFromIndexModel(indexState, &plan)
	restoreVirtualNullBlocks(&plan, nullBlocks)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *virtualIndexResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state VirtualIndexResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := state.Name.ValueString()
	if !state.DeletionProtection.IsNull() && state.DeletionProtection.ValueBool() {
		resp.Diagnostics.AddError(
			"Deletion Protection Enabled",
			fmt.Sprintf("Cannot delete virtual index %q because deletion_protection is enabled. Set deletion_protection = false and apply before destroying.", indexName),
		)
		return
	}

	primaryIndexName := state.PrimaryIndexName.ValueString()
	if primaryIndexName != "" {
		if err := removeVirtualReplicaLink(r.client, primaryIndexName, indexName); err != nil {
			resp.Diagnostics.AddError("Error unlinking virtual replica", "Could not unlink virtual replica "+indexName+" from primary index "+primaryIndexName+": "+err.Error())
			return
		}
	}

	delResp, err := r.client.DeleteIndex(r.client.NewApiDeleteIndexRequest(indexName))
	if err != nil {
		var apiErr *search.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return
		}
		resp.Diagnostics.AddError("Error deleting virtual index", "Could not delete virtual index "+indexName+": "+err.Error())
		return
	}

	if err = waitForIndexTask(r.client, indexName, delResp.TaskID); err != nil {
		resp.Diagnostics.AddError("Error waiting for virtual index deletion", "Could not wait for task: "+err.Error())
	}
}

func (r *virtualIndexResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func (r *virtualIndexResource) readVirtualIndex(ctx context.Context, model *VirtualIndexResourceModel) diag.Diagnostics {
	indexModel := virtualToIndexModel(*model)
	diags := r.readIndexModel(ctx, &indexModel)
	virtualFromIndexModel(indexModel, model)
	if model.PrimaryIndexName.IsNull() || model.PrimaryIndexName.ValueString() == "" {
		diags.AddError("Index is not a virtual replica", "The requested index does not report a primary index and cannot be managed as algolia_virtual_index.")
	}
	return diags
}

func (r *virtualIndexResource) readIndexModel(ctx context.Context, model *IndexResourceModel) diag.Diagnostics {
	indexName := model.Name.ValueString()

	settingsResp, err := r.client.GetSettings(r.client.NewApiGetSettingsRequest(indexName))
	if err != nil {
		var diags diag.Diagnostics
		var apiErr *search.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			diags.AddError("Error reading index", "Could not read index "+indexName+": "+err.Error())
			return diags
		}
		diags.AddError("Error reading index", "Could not read index "+indexName+": "+err.Error())
		return diags
	}

	diags := flattenSettingsResponse(ctx, settingsResp, model)
	if diags.HasError() {
		return diags
	}

	listResp, err := r.client.ListIndices(r.client.NewApiListIndicesRequest())
	if err != nil {
		tflog.Warn(ctx, "Could not list indices for metadata", map[string]any{"error": err.Error()})
		model.Entries = types.Int64Value(0)
		model.DataSize = types.Int64Value(0)
		model.CreatedAt = types.StringValue("")
		model.UpdatedAt = types.StringValue("")
		return diags
	}

	for _, idx := range listResp.Items {
		if idx.Name == indexName {
			model.Entries = types.Int64Value(int64(idx.Entries))
			model.DataSize = types.Int64Value(idx.DataSize)
			model.CreatedAt = types.StringValue(idx.CreatedAt)
			model.UpdatedAt = types.StringValue(idx.UpdatedAt)
			return diags
		}
	}

	model.Entries = types.Int64Value(0)
	model.DataSize = types.Int64Value(0)
	model.CreatedAt = types.StringValue("")
	model.UpdatedAt = types.StringValue("")
	return diags
}

func ensureVirtualReplicaLinked(client *search.APIClient, primaryIndexName, replicaName string) error {
	settings, err := client.GetSettings(client.NewApiGetSettingsRequest(primaryIndexName))
	if err != nil {
		return err
	}

	virtualName := "virtual(" + replicaName + ")"
	replicas := append([]string(nil), settings.Replicas...)
	for _, existing := range replicas {
		if existing == virtualName {
			return nil
		}
	}

	replicas = append(replicas, virtualName)
	setResp, err := client.SetSettings(client.NewApiSetSettingsRequest(primaryIndexName, search.NewIndexSettings(
		search.WithIndexSettingsReplicas(replicas),
	)))
	if err != nil {
		return err
	}

	return waitForIndexTask(client, primaryIndexName, setResp.TaskID)
}

func removeVirtualReplicaLink(client *search.APIClient, primaryIndexName, replicaName string) error {
	settings, err := client.GetSettings(client.NewApiGetSettingsRequest(primaryIndexName))
	if err != nil {
		var apiErr *search.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return nil
		}
		return err
	}

	virtualName := "virtual(" + replicaName + ")"
	replicas := append([]string(nil), settings.Replicas...)
	filtered := replicas[:0]
	changed := false
	for _, existing := range replicas {
		if existing == virtualName {
			changed = true
			continue
		}
		filtered = append(filtered, existing)
	}
	if !changed {
		return nil
	}

	setResp, err := client.SetSettings(client.NewApiSetSettingsRequest(primaryIndexName, search.NewIndexSettings(
		search.WithIndexSettingsReplicas(filtered),
	)))
	if err != nil {
		return err
	}

	return waitForIndexTask(client, primaryIndexName, setResp.TaskID)
}

func restoreVirtualNullBlocks(model *VirtualIndexResourceModel, blocks nullBlocks) {
	indexModel := virtualToIndexModel(*model)
	restoreNullBlocks(&indexModel, blocks)
	virtualFromIndexModel(indexModel, model)
}
