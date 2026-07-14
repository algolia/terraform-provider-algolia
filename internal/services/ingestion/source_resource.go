package ingestion

import (
	"context"
	"errors"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &sourceResource{}
	_ resource.ResourceWithConfigure   = &sourceResource{}
	_ resource.ResourceWithImportState = &sourceResource{}
)

// sourceResource manages an algolia_ingestion_source resource. It embeds
// base (see client.go) for Configure-time wiring and an on-demand
// region-routed Ingestion client.
type sourceResource struct {
	base
}

// NewSourceResource returns the algolia_ingestion_source resource.
func NewSourceResource() resource.Resource {
	return &sourceResource{}
}

func (r *sourceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingestion_source"
}

func (r *sourceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = sourceResourceSchema()
}

func (r *sourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(r.configure(req.ProviderData)...)
}

func (r *sourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	create, expandDiags := expandSourceCreate(&plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Ingestion source", map[string]any{
		"name": plan.Name.ValueString(),
		"type": plan.Type.ValueString(),
	})

	createResp, err := client.CreateSource(client.NewApiCreateSourceRequest(create))
	if err != nil {
		resp.Diagnostics.AddError("Error creating Ingestion source", "Could not create source "+plan.Name.ValueString()+": "+err.Error())
		return
	}

	apiResp, err := client.GetSource(client.NewApiGetSourceRequest(createResp.SourceID))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ingestion source", "Could not read back source after creation: "+err.Error())
		return
	}

	// flattenSource compares the API's input against plan.Input (the value
	// the user just configured) and only adopts the API's encoding if it's
	// not semantically equivalent.
	resp.Diagnostics.Append(flattenSource(apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourceID := state.SourceID.ValueString()
	apiResp, err := client.GetSource(client.NewApiGetSourceRequest(sourceID))
	if err != nil {
		var apiErr *ingestionapi.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			tflog.Warn(ctx, "Ingestion source not found; removing from state", map[string]any{"source_id": sourceID})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading Ingestion source", "Could not read source "+sourceID+": "+err.Error())
		return
	}

	// flattenSource preserves state.Input as-is when it is semantically
	// equal to the API's current value, so out-of-band JSON formatting
	// differences don't create a perpetual diff. See the input attribute's
	// schema description.
	resp.Diagnostics.Append(flattenSource(apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *sourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	update, expandDiags := expandSourceUpdate(&plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourceID := plan.SourceID.ValueString()
	tflog.Debug(ctx, "Updating Ingestion source", map[string]any{"source_id": sourceID})

	if _, err := client.UpdateSource(client.NewApiUpdateSourceRequest(sourceID, update)); err != nil {
		resp.Diagnostics.AddError("Error updating Ingestion source", "Could not update source "+sourceID+": "+err.Error())
		return
	}

	apiResp, err := client.GetSource(client.NewApiGetSourceRequest(sourceID))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ingestion source", "Could not read back source after update: "+err.Error())
		return
	}

	resp.Diagnostics.Append(flattenSource(apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourceID := state.SourceID.ValueString()
	tflog.Debug(ctx, "Deleting Ingestion source", map[string]any{"source_id": sourceID})

	if _, err := client.DeleteSource(client.NewApiDeleteSourceRequest(sourceID)); err != nil {
		var apiErr *ingestionapi.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return
		}

		resp.Diagnostics.AddError("Error deleting Ingestion source", "Could not delete source "+sourceID+": "+err.Error())
	}
}

func (r *sourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourceID := req.ID
	apiResp, err := client.GetSource(client.NewApiGetSourceRequest(sourceID))
	if err != nil {
		resp.Diagnostics.AddError("Error importing Ingestion source", "Could not import source "+sourceID+": "+err.Error())
		return
	}

	// Unlike algolia_ingestion_authentication, GetSource returns input in
	// full (nothing is redacted), so - starting from a zero-value model
	// whose Input is null - flattenSource populates input straight from the
	// API response. There is no prior configured value to preserve on
	// import.
	var state SourceResourceModel
	resp.Diagnostics.Append(flattenSource(apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
