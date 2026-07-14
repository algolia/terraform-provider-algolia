package ingestion

import (
	"context"
	"errors"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &destinationResource{}
	_ resource.ResourceWithConfigure   = &destinationResource{}
	_ resource.ResourceWithImportState = &destinationResource{}
)

// destinationResource manages an algolia_ingestion_destination resource.
// It embeds base (see client.go) for Configure-time wiring and an
// on-demand region-routed Ingestion client.
type destinationResource struct {
	base
}

// NewDestinationResource returns the algolia_ingestion_destination
// resource.
func NewDestinationResource() resource.Resource {
	return &destinationResource{}
}

func (r *destinationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingestion_destination"
}

func (r *destinationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = destinationResourceSchema()
}

func (r *destinationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(r.configure(req.ProviderData)...)
}

func (r *destinationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DestinationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	create, expandDiags := expandDestinationCreate(&plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Ingestion destination", map[string]any{
		"name": plan.Name.ValueString(),
		"type": plan.Type.ValueString(),
	})

	createResp, err := client.CreateDestination(client.NewApiCreateDestinationRequest(create))
	if err != nil {
		resp.Diagnostics.AddError("Error creating Ingestion destination", "Could not create destination "+plan.Name.ValueString()+": "+err.Error())
		return
	}

	apiResp, err := client.GetDestination(client.NewApiGetDestinationRequest(createResp.DestinationID))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ingestion destination", "Could not read back destination after creation: "+err.Error())
		return
	}

	// flattenDestination compares the API's input against plan.Input (the
	// value the user just configured) and only adopts the API's encoding if
	// it's not semantically equivalent.
	resp.Diagnostics.Append(flattenDestination(apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *destinationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DestinationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	destinationID := state.DestinationID.ValueString()
	apiResp, err := client.GetDestination(client.NewApiGetDestinationRequest(destinationID))
	if err != nil {
		var apiErr *ingestionapi.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			tflog.Warn(ctx, "Ingestion destination not found; removing from state", map[string]any{"destination_id": destinationID})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading Ingestion destination", "Could not read destination "+destinationID+": "+err.Error())
		return
	}

	// flattenDestination preserves state.Input as-is when it is
	// semantically equal to the API's current value, so out-of-band JSON
	// formatting differences don't create a perpetual diff. See the input
	// attribute's schema description.
	resp.Diagnostics.Append(flattenDestination(apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *destinationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DestinationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	update, expandDiags := expandDestinationUpdate(&plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	destinationID := plan.DestinationID.ValueString()
	tflog.Debug(ctx, "Updating Ingestion destination", map[string]any{"destination_id": destinationID})

	if _, err := client.UpdateDestination(client.NewApiUpdateDestinationRequest(destinationID, update)); err != nil {
		resp.Diagnostics.AddError("Error updating Ingestion destination", "Could not update destination "+destinationID+": "+err.Error())
		return
	}

	apiResp, err := client.GetDestination(client.NewApiGetDestinationRequest(destinationID))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ingestion destination", "Could not read back destination after update: "+err.Error())
		return
	}

	resp.Diagnostics.Append(flattenDestination(apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *destinationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DestinationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	destinationID := state.DestinationID.ValueString()
	tflog.Debug(ctx, "Deleting Ingestion destination", map[string]any{"destination_id": destinationID})

	if _, err := client.DeleteDestination(client.NewApiDeleteDestinationRequest(destinationID)); err != nil {
		var apiErr *ingestionapi.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return
		}

		resp.Diagnostics.AddError("Error deleting Ingestion destination", "Could not delete destination "+destinationID+": "+err.Error())
	}
}

func (r *destinationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	destinationID := req.ID
	apiResp, err := client.GetDestination(client.NewApiGetDestinationRequest(destinationID))
	if err != nil {
		resp.Diagnostics.AddError("Error importing Ingestion destination", "Could not import destination "+destinationID+": "+err.Error())
		return
	}

	// Like algolia_ingestion_source, GetDestination returns input in full
	// (nothing is redacted), so - starting from a zero-value model whose
	// Input is null - flattenDestination populates input straight from the
	// API response. There is no prior configured value to preserve on
	// import.
	var state DestinationResourceModel
	resp.Diagnostics.Append(flattenDestination(apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
