package ingestion

import (
	"context"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &transformationResource{}
	_ resource.ResourceWithConfigure   = &transformationResource{}
	_ resource.ResourceWithImportState = &transformationResource{}
)

// transformationResource manages an algolia_ingestion_transformation
// resource. It embeds base (see client.go) for Configure-time wiring and an
// on-demand region-routed Ingestion client.
type transformationResource struct {
	base
}

// NewTransformationResource returns the algolia_ingestion_transformation
// resource.
func NewTransformationResource() resource.Resource {
	return &transformationResource{}
}

func (r *transformationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingestion_transformation"
}

func (r *transformationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = transformationResourceSchema()
}

func (r *transformationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(r.configure(req.ProviderData)...)
}

func (r *transformationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TransformationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTransformation(ctx, client, &plan, resp)
}

// createTransformation is the Create body, split out so that a unit test can
// drive it against an httptest-backed client (see identity_state_test.go) - the
// client base.client() builds always talks to the real, region-routed Ingestion
// API. Create itself only resolves the plan and the client. Same pattern as
// abtest.createABTest.
func createTransformation(ctx context.Context, client *ingestionapi.APIClient, plan *TransformationResourceModel, resp *resource.CreateResponse) {
	create, expandDiags := expandTransformationCreate(plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Ingestion transformation", map[string]any{"name": plan.Name.ValueString()})

	createResp, err := client.CreateTransformation(client.NewApiCreateTransformationRequest(create), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error creating Ingestion transformation", "Could not create transformation "+plan.Name.ValueString()+": "+algoliaerr.Explain(err))
		return
	}

	// The transformation now exists in Algolia under the server-assigned UUID in
	// createResp, which is the only handle Terraform will ever have on it.
	// Persist it before the read-back below, so that a failure there leaves a
	// resource Terraform can read, update and destroy instead of orphaning a
	// transformation that exists remotely but not in state: the next apply would
	// create a second one (and fail, since transformation names must be unique)
	// and nothing would ever adopt the first.
	resp.Diagnostics.Append(resp.State.Set(ctx, newTransformationIdentityState(*plan, createResp.TransformationID))...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := client.GetTransformation(client.NewApiGetTransformationRequest(createResp.TransformationID), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ingestion transformation", "Could not read back transformation after creation: "+algoliaerr.Explain(err))
		return
	}

	// flattenTransformation compares the API's input against plan.Input
	// (the value the user just configured) and only adopts the API's
	// encoding if it's not semantically equivalent.
	resp.Diagnostics.Append(flattenTransformation(apiResp, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// newTransformationIdentityState returns the state to persist immediately after
// a successful CreateTransformation, before the read-back that can still fail.
// Terraform rejects an apply result containing unknown values, so every unknown
// has to be resolved here: id/transformation_id come from the create response,
// while created_at/updated_at are knowable only from the read-back and are
// therefore written as null - the next Read fills them in.
//
// code needs the same treatment for a different reason: it is Optional+Computed
// with no UseStateForUnknown (the API re-derives it from input.code), so it is
// unknown in the plan whenever the configuration omits it, and only the
// read-back can tell what the API derived. authentication_ids, the only
// non-scalar attribute, is Optional without Computed, so the plan carries the
// configured list or a *typed* null list - which is what the framework requires
// (a zero-value types.List carries no element type and fails conversion at
// runtime) - and it is passed through untouched rather than rebuilt.
func newTransformationIdentityState(plan TransformationResourceModel, transformationID string) TransformationResourceModel {
	plan.ID = types.StringValue(transformationID)
	plan.TransformationID = types.StringValue(transformationID)
	plan.CreatedAt = types.StringNull()
	plan.UpdatedAt = types.StringNull()
	if plan.Code.IsUnknown() {
		plan.Code = types.StringNull()
	}

	return plan
}

func (r *transformationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TransformationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	transformationID := state.TransformationID.ValueString()
	apiResp, err := client.GetTransformation(client.NewApiGetTransformationRequest(transformationID), ingestionapi.WithContext(ctx))
	if err != nil {
		if algoliaerr.IsNotFound(err) {
			tflog.Warn(ctx, "Ingestion transformation not found; removing from state", map[string]any{"transformation_id": transformationID})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading Ingestion transformation", "Could not read transformation "+transformationID+": "+algoliaerr.Explain(err))
		return
	}

	// flattenTransformation preserves state.Input as-is when it is
	// semantically equal to the API's current value, so out-of-band JSON
	// formatting differences don't create a perpetual diff. See the input
	// attribute's schema description.
	resp.Diagnostics.Append(flattenTransformation(apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *transformationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TransformationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// UpdateTransformation takes the same *TransformationCreate body as
	// CreateTransformation (see expandTransformationCreate), so it is
	// reused here rather than a separate expandTransformationUpdate.
	update, expandDiags := expandTransformationCreate(&plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	transformationID := plan.TransformationID.ValueString()
	tflog.Debug(ctx, "Updating Ingestion transformation", map[string]any{"transformation_id": transformationID})

	if _, err := client.UpdateTransformation(client.NewApiUpdateTransformationRequest(transformationID, update), ingestionapi.WithContext(ctx)); err != nil {
		resp.Diagnostics.AddError("Error updating Ingestion transformation", "Could not update transformation "+transformationID+": "+algoliaerr.Explain(err))
		return
	}

	apiResp, err := client.GetTransformation(client.NewApiGetTransformationRequest(transformationID), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ingestion transformation", "Could not read back transformation after update: "+algoliaerr.Explain(err))
		return
	}

	resp.Diagnostics.Append(flattenTransformation(apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *transformationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TransformationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	transformationID := state.TransformationID.ValueString()
	tflog.Debug(ctx, "Deleting Ingestion transformation", map[string]any{"transformation_id": transformationID})

	if _, err := client.DeleteTransformation(client.NewApiDeleteTransformationRequest(transformationID), ingestionapi.WithContext(ctx)); err != nil {
		if algoliaerr.IsNotFound(err) {
			return
		}

		resp.Diagnostics.AddError("Error deleting Ingestion transformation", "Could not delete transformation "+transformationID+": "+algoliaerr.Explain(err))
	}
}

func (r *transformationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	transformationID := req.ID
	apiResp, err := client.GetTransformation(client.NewApiGetTransformationRequest(transformationID), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error importing Ingestion transformation", "Could not import transformation "+transformationID+": "+algoliaerr.Explain(err))
		return
	}

	// Like algolia_ingestion_source/algolia_ingestion_destination,
	// GetTransformation returns code/input in full (nothing is redacted),
	// so - starting from a zero-value model whose Input is null -
	// flattenTransformation populates both straight from the API response.
	// There is no prior configured value to preserve on import.
	var state TransformationResourceModel
	resp.Diagnostics.Append(flattenTransformation(apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
