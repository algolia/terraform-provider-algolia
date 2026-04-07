package synonym

import (
	"context"
	"errors"
	"fmt"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &synonymResource{}
	_ resource.ResourceWithConfigure   = &synonymResource{}
	_ resource.ResourceWithImportState = &synonymResource{}
)

type synonymResource struct {
	client *search.APIClient
}

func NewResource() resource.Resource {
	return &synonymResource{}
}

func (r *synonymResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_synonym"
}

func (r *synonymResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = synonymResourceSchema()
}

func (r *synonymResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *synonymResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SynonymResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := plan.IndexName.ValueString()
	objectID := plan.ObjectID.ValueString()
	tflog.Debug(ctx, "Creating synonym", map[string]any{"index_name": indexName, "object_id": objectID})

	hit, diags := buildSynonymHit(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	saveResp, err := r.client.SaveSynonym(r.client.NewApiSaveSynonymRequest(indexName, objectID, hit))
	if err != nil {
		resp.Diagnostics.AddError("Error creating synonym", "Could not create synonym "+objectID+" on index "+indexName+": "+err.Error())
		return
	}

	if err := waitForSynonymTask(r.client, indexName, saveResp.TaskID); err != nil {
		resp.Diagnostics.AddError("Error waiting for synonym creation", "Could not confirm synonym creation: "+err.Error())
		return
	}

	apiResp, err := r.client.GetSynonym(r.client.NewApiGetSynonymRequest(indexName, objectID))
	if err != nil {
		resp.Diagnostics.AddError("Error reading synonym", "Could not read synonym "+objectID+" on index "+indexName+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(hydrateSynonymModel(indexName, apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *synonymResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SynonymResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := state.IndexName.ValueString()
	objectID := state.ObjectID.ValueString()

	apiResp, err := r.client.GetSynonym(r.client.NewApiGetSynonymRequest(indexName, objectID))
	if err != nil {
		var apiErr *search.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			tflog.Warn(ctx, "Synonym not found; removing from state", map[string]any{"index_name": indexName, "object_id": objectID})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading synonym", "Could not read synonym "+objectID+" on index "+indexName+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(hydrateSynonymModel(indexName, apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *synonymResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SynonymResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := plan.IndexName.ValueString()
	objectID := plan.ObjectID.ValueString()

	hit, diags := buildSynonymHit(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	saveResp, err := r.client.SaveSynonym(r.client.NewApiSaveSynonymRequest(indexName, objectID, hit))
	if err != nil {
		resp.Diagnostics.AddError("Error updating synonym", "Could not update synonym "+objectID+" on index "+indexName+": "+err.Error())
		return
	}

	if err := waitForSynonymTask(r.client, indexName, saveResp.TaskID); err != nil {
		resp.Diagnostics.AddError("Error waiting for synonym update", "Could not confirm synonym update: "+err.Error())
		return
	}

	apiResp, err := r.client.GetSynonym(r.client.NewApiGetSynonymRequest(indexName, objectID))
	if err != nil {
		resp.Diagnostics.AddError("Error reading synonym", "Could not read synonym "+objectID+" on index "+indexName+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(hydrateSynonymModel(indexName, apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *synonymResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SynonymResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := state.IndexName.ValueString()
	objectID := state.ObjectID.ValueString()

	deleteResp, err := r.client.DeleteSynonym(r.client.NewApiDeleteSynonymRequest(indexName, objectID))
	if err != nil {
		var apiErr *search.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return
		}

		resp.Diagnostics.AddError("Error deleting synonym", "Could not delete synonym "+objectID+" on index "+indexName+": "+err.Error())
		return
	}

	if err := waitForSynonymTask(r.client, indexName, deleteResp.TaskID); err != nil {
		var apiErr *search.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return
		}

		resp.Diagnostics.AddError("Error waiting for synonym deletion", "Could not confirm synonym deletion: "+err.Error())
	}
}

func (r *synonymResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	indexName, objectID, err := parseSynonymImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	apiResp, err := r.client.GetSynonym(r.client.NewApiGetSynonymRequest(indexName, objectID))
	if err != nil {
		resp.Diagnostics.AddError("Error importing synonym", "Could not import synonym "+req.ID+": "+err.Error())
		return
	}

	var state SynonymResourceModel
	resp.Diagnostics.Append(hydrateSynonymModel(indexName, apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

