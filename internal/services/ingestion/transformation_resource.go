package ingestion

import (
	"context"
	"errors"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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

	create, expandDiags := expandTransformationCreate(&plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Ingestion transformation", map[string]any{"name": plan.Name.ValueString()})

	createResp, err := client.CreateTransformation(client.NewApiCreateTransformationRequest(create))
	if err != nil {
		resp.Diagnostics.AddError("Error creating Ingestion transformation", "Could not create transformation "+plan.Name.ValueString()+": "+err.Error())
		return
	}

	apiResp, err := client.GetTransformation(client.NewApiGetTransformationRequest(createResp.TransformationID))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ingestion transformation", "Could not read back transformation after creation: "+err.Error())
		return
	}

	// flattenTransformation compares the API's input against plan.Input
	// (the value the user just configured) and only adopts the API's
	// encoding if it's not semantically equivalent.
	resp.Diagnostics.Append(flattenTransformation(apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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
	apiResp, err := client.GetTransformation(client.NewApiGetTransformationRequest(transformationID))
	if err != nil {
		var apiErr *ingestionapi.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			tflog.Warn(ctx, "Ingestion transformation not found; removing from state", map[string]any{"transformation_id": transformationID})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading Ingestion transformation", "Could not read transformation "+transformationID+": "+err.Error())
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

	if _, err := client.UpdateTransformation(client.NewApiUpdateTransformationRequest(transformationID, update)); err != nil {
		resp.Diagnostics.AddError("Error updating Ingestion transformation", "Could not update transformation "+transformationID+": "+err.Error())
		return
	}

	apiResp, err := client.GetTransformation(client.NewApiGetTransformationRequest(transformationID))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ingestion transformation", "Could not read back transformation after update: "+err.Error())
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

	if _, err := client.DeleteTransformation(client.NewApiDeleteTransformationRequest(transformationID)); err != nil {
		var apiErr *ingestionapi.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return
		}

		resp.Diagnostics.AddError("Error deleting Ingestion transformation", "Could not delete transformation "+transformationID+": "+err.Error())
	}
}

func (r *transformationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	transformationID := req.ID
	apiResp, err := client.GetTransformation(client.NewApiGetTransformationRequest(transformationID))
	if err != nil {
		resp.Diagnostics.AddError("Error importing Ingestion transformation", "Could not import transformation "+transformationID+": "+err.Error())
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
