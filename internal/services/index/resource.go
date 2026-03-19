package index

import (
	"context"
	"fmt"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &indexResource{}
	_ resource.ResourceWithImportState = &indexResource{}
)

type indexResource struct {
	client *search.APIClient
}

func NewResource() resource.Resource {
	return &indexResource{}
}

func (r *indexResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_index"
}

func (r *indexResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = indexResourceSchema()
}

func (r *indexResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *indexResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IndexResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := plan.Name.ValueString()
	tflog.Debug(ctx, "Creating index", map[string]interface{}{"name": indexName})

	settings, diags := expandIndexSettings(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	setResp, err := r.client.SetSettings(r.client.NewApiSetSettingsRequest(indexName, settings))
	if err != nil {
		resp.Diagnostics.AddError("Error creating index", "Could not create index "+indexName+": "+err.Error())
		return
	}

	_, err = r.client.WaitForTask(indexName, setResp.TaskID)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for index creation", "Could not wait for task: "+err.Error())
		return
	}

	diags = r.readIndex(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *indexResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state IndexResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags := r.readIndex(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *indexResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan IndexResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := plan.Name.ValueString()
	tflog.Debug(ctx, "Updating index", map[string]interface{}{"name": indexName})

	settings, diags := expandIndexSettings(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	setResp, err := r.client.SetSettings(r.client.NewApiSetSettingsRequest(indexName, settings))
	if err != nil {
		resp.Diagnostics.AddError("Error updating index", "Could not update index "+indexName+": "+err.Error())
		return
	}

	_, err = r.client.WaitForTask(indexName, setResp.TaskID)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for index update", "Could not wait for task: "+err.Error())
		return
	}

	diags = r.readIndex(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *indexResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state IndexResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := state.Name.ValueString()

	if !state.DeletionProtection.IsNull() && state.DeletionProtection.ValueBool() {
		resp.Diagnostics.AddError(
			"Deletion Protection Enabled",
			fmt.Sprintf("Cannot delete index %q because deletion_protection is enabled. "+
				"Set deletion_protection = false and apply before destroying.", indexName),
		)
		return
	}

	tflog.Debug(ctx, "Deleting index", map[string]interface{}{"name": indexName})

	delResp, err := r.client.DeleteIndex(r.client.NewApiDeleteIndexRequest(indexName))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting index", "Could not delete index "+indexName+": "+err.Error())
		return
	}

	_, err = r.client.WaitForTask(indexName, delResp.TaskID)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for index deletion", "Could not wait for task: "+err.Error())
		return
	}
}

func (r *indexResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func (r *indexResource) readIndex(ctx context.Context, model *IndexResourceModel) diag.Diagnostics {
	indexName := model.Name.ValueString()

	settingsResp, err := r.client.GetSettings(r.client.NewApiGetSettingsRequest(indexName))
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Error reading index", "Could not read index "+indexName+": "+err.Error())
		return diags
	}

	return flattenSettingsResponse(ctx, settingsResp, model)
}
