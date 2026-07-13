package collection

import (
	"context"
	"errors"
	"fmt"

	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                     = &collectionResource{}
	_ resource.ResourceWithImportState      = &collectionResource{}
	_ resource.ResourceWithConfigValidators = &collectionResource{}
)

type collectionResource struct {
	client *Client
}

func NewResource() resource.Resource {
	return &collectionResource{}
}

func (r *collectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_collection"
}

func (r *collectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = collectionResourceSchema()
}

func (r *collectionResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{maxFiltersValidator{}}
}

func (r *collectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	client, ok := data.CollectionsClient.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Collections Client Type",
			fmt.Sprintf("Expected *collection.Client, got: %T", data.CollectionsClient),
		)
		return
	}

	r.client = client
}

func (r *collectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CollectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating collection", map[string]interface{}{"name": plan.Name.ValueString(), "index_name": plan.IndexName.ValueString()})

	apiReq, diags := expandCreate(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.UpsertCollection(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating collection", "Could not create collection: "+err.Error())
		return
	}

	resp.Diagnostics.Append(hydrateCollectionResourceState(ctx, apiResp, plan.Commit, plan.DeletionProtection, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *collectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CollectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	tflog.Debug(ctx, "Reading collection", map[string]interface{}{"id": id})

	apiResp, err := r.client.GetCollection(ctx, id)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			tflog.Warn(ctx, "Collection not found; removing from state", map[string]interface{}{"id": id})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading collection", "Could not read collection "+id+": "+err.Error())
		return
	}

	// GET /collections/{id} omits indexName; preserve the value from prior
	// state so we don't trigger RequiresReplace on every refresh.
	if apiResp.IndexName == "" {
		apiResp.IndexName = state.IndexName.ValueString()
	}

	resp.Diagnostics.Append(hydrateCollectionResourceState(ctx, apiResp, state.Commit, state.DeletionProtection, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *collectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CollectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state CollectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := plan.ID.ValueString()
	tflog.Debug(ctx, "Updating collection", map[string]interface{}{"id": id})

	apiReq, diags := expandUpdate(ctx, &state, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.UpsertCollection(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating collection", "Could not update collection "+id+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(hydrateCollectionResourceState(ctx, apiResp, plan.Commit, plan.DeletionProtection, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *collectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CollectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	if deletionProtectionValue(state.DeletionProtection).ValueBool() {
		resp.Diagnostics.AddError(
			"Deletion Protection Enabled",
			fmt.Sprintf("Cannot delete collection %q because deletion_protection is enabled. "+
				"Set deletion_protection = false and apply before destroying.", id),
		)
		return
	}

	tflog.Debug(ctx, "Deleting collection", map[string]interface{}{"id": id})

	commit := commitValue(state.Commit).ValueBool()
	if err := r.client.DeleteCollection(ctx, id, &DeleteRequest{Commit: &commit}); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Error deleting collection", "Could not delete collection "+id+": "+err.Error())
		return
	}
}

func (r *collectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	indexName, id, err := parseCollectionImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	apiResp, err := r.client.GetCollection(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error importing collection", "Could not import collection "+req.ID+": "+err.Error())
		return
	}

	// GET /collections/{id} omits indexName from the response; inject it from
	// the import identifier so state matches the HCL without forcing a
	// destroy/recreate on the next plan (index_name is RequiresReplace).
	if apiResp.IndexName == "" {
		apiResp.IndexName = indexName
	}

	var state CollectionResourceModel
	resp.Diagnostics.Append(hydrateImportedCollectionResourceState(ctx, apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
