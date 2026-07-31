package ingestion

import (
	"context"
	"fmt"

	"github.com/algolia/terraform-provider-algolia/internal/deletionprotection"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

	createDestination(ctx, client, &plan, resp)
}

// createDestination is the Create body, split out so that a unit test can drive
// it against an httptest-backed client (see identity_state_test.go) - the client
// base.client() builds always talks to the real, region-routed Ingestion API.
// Create itself only resolves the plan and the client. Same pattern as
// abtest.createABTest.
func createDestination(ctx context.Context, client *ingestionapi.APIClient, plan *DestinationResourceModel, resp *resource.CreateResponse) {
	create, expandDiags := expandDestinationCreate(plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Ingestion destination", map[string]any{
		"name": plan.Name.ValueString(),
		"type": plan.Type.ValueString(),
	})

	createResp, err := client.CreateDestination(client.NewApiCreateDestinationRequest(create), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error creating Ingestion destination", "Could not create destination "+plan.Name.ValueString()+": "+algoliaerr.Explain(err))
		return
	}

	// The destination now exists in Algolia under the server-assigned UUID in
	// createResp, which is the only handle Terraform will ever have on it.
	// Persist it before the read-back below, so that a failure there leaves a
	// resource Terraform can read, update and destroy instead of orphaning a
	// destination that exists remotely but not in state: the next apply would
	// create a second destination and nothing would ever adopt the first.
	resp.Diagnostics.Append(resp.State.Set(ctx, newDestinationIdentityState(*plan, createResp.DestinationID))...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := client.GetDestination(client.NewApiGetDestinationRequest(createResp.DestinationID), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ingestion destination", "Could not read back destination after creation: "+algoliaerr.Explain(err))
		return
	}

	// flattenDestination compares the API's input against plan.Input (the
	// value the user just configured) and only adopts the API's encoding if
	// it's not semantically equivalent.
	resp.Diagnostics.Append(flattenDestination(apiResp, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// newDestinationIdentityState returns the state to persist immediately after a
// successful CreateDestination, before the read-back that can still fail.
// Terraform rejects an apply result containing unknown values, so every unknown
// has to be resolved here: id/destination_id come from the create response,
// while created_at/updated_at are knowable only from the read-back and are
// therefore written as null - the next Read fills them in. Every other
// attribute is Required or Optional-only, so the plan holds the configuration
// verbatim and is already known. That includes transformation_ids, the model's
// only non-scalar attribute: because it is Optional without Computed, the plan
// carries the configured list or a *typed* null list, which is what the
// framework requires (a zero-value types.List carries no element type and fails
// conversion at runtime), so it is passed through untouched rather than rebuilt.
func newDestinationIdentityState(plan DestinationResourceModel, destinationID string) DestinationResourceModel {
	plan.ID = types.StringValue(destinationID)
	plan.DestinationID = types.StringValue(destinationID)
	plan.CreatedAt = types.StringNull()
	plan.UpdatedAt = types.StringNull()

	return plan
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
	apiResp, err := client.GetDestination(client.NewApiGetDestinationRequest(destinationID), ingestionapi.WithContext(ctx))
	if err != nil {
		if algoliaerr.IsNotFound(err) {
			tflog.Warn(ctx, "Ingestion destination not found; removing from state", map[string]any{"destination_id": destinationID})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading Ingestion destination", "Could not read destination "+destinationID+": "+algoliaerr.Explain(err))
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

	if _, err := client.UpdateDestination(client.NewApiUpdateDestinationRequest(destinationID, update), ingestionapi.WithContext(ctx)); err != nil {
		resp.Diagnostics.AddError("Error updating Ingestion destination", "Could not update destination "+destinationID+": "+algoliaerr.Explain(err))
		return
	}

	apiResp, err := client.GetDestination(client.NewApiGetDestinationRequest(destinationID), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ingestion destination", "Could not read back destination after update: "+algoliaerr.Explain(err))
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

	if deletionprotection.Enabled(state.DeletionProtection) {
		resp.Diagnostics.Append(deletionprotection.Refuse(fmt.Sprintf("ingestion destination %q", state.DestinationID.ValueString())))
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	destinationID := state.DestinationID.ValueString()
	tflog.Debug(ctx, "Deleting Ingestion destination", map[string]any{"destination_id": destinationID})

	if _, err := client.DeleteDestination(client.NewApiDeleteDestinationRequest(destinationID), ingestionapi.WithContext(ctx)); err != nil {
		if algoliaerr.IsNotFound(err) {
			return
		}

		resp.Diagnostics.AddError("Error deleting Ingestion destination", "Could not delete destination "+destinationID+": "+algoliaerr.Explain(err))
	}
}

func (r *destinationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	destinationID := req.ID
	apiResp, err := client.GetDestination(client.NewApiGetDestinationRequest(destinationID), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error importing Ingestion destination", "Could not import destination "+destinationID+": "+algoliaerr.Explain(err))
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
