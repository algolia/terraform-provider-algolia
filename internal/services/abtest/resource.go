package abtest

import (
	"context"
	"errors"
	"strconv"

	abtestingapi "github.com/algolia/algoliasearch-client-go/v4/algolia/abtesting-v3"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &abTestResource{}
	_ resource.ResourceWithConfigure   = &abTestResource{}
	_ resource.ResourceWithImportState = &abTestResource{}
)

// abTestResource manages an algolia_ab_test resource. It embeds base (see
// client.go) for Configure-time wiring and an on-demand region-routed A/B
// Testing client.
//
// The A/B Testing API has no update endpoint (AddABTests/GetABTest/
// StopABTest/DeleteABTest is the full CRUD-adjacent surface - see
// api_abtesting_v3.go in the abtesting-v3 client), so every attribute that
// shapes test creation (name, end_at, variants, metrics, configuration) is
// RequiresReplace in the schema. Terraform should therefore never actually
// call Update: any change to those attributes plans a replace instead.
// Update is still implemented, to satisfy resource.Resource, as a
// defensive read-back rather than a no-op.
type abTestResource struct {
	base
}

// NewResource returns the algolia_ab_test resource.
func NewResource() resource.Resource {
	return &abTestResource{}
}

func (r *abTestResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ab_test"
}

func (r *abTestResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = abTestResourceSchema()
}

func (r *abTestResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(r.configure(req.ProviderData)...)
}

func (r *abTestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ABTestResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	addRequest, expandDiags := expandAddABTestsRequest(&plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating A/B test", map[string]any{"name": plan.Name.ValueString()})

	createResp, err := client.AddABTests(client.NewApiAddABTestsRequest(addRequest))
	if err != nil {
		resp.Diagnostics.AddError("Error creating A/B test", "Could not create A/B test "+plan.Name.ValueString()+": "+err.Error())
		return
	}

	apiResp, err := client.GetABTest(client.NewApiGetABTestRequest(createResp.AbTestID))
	if err != nil {
		resp.Diagnostics.AddError("Error reading A/B test", "Could not read back A/B test after creation: "+err.Error())
		return
	}

	// flattenABTestComputed only refreshes id/ab_test_id/status; it leaves
	// name/end_at/variants/metrics/configuration as already set in plan.
	resp.Diagnostics.Append(flattenABTestComputed(apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *abTestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ABTestResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	abTestID := int32(state.ABTestID.ValueInt64())
	apiResp, err := client.GetABTest(client.NewApiGetABTestRequest(abTestID))
	if err != nil {
		var apiErr *abtestingapi.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			tflog.Warn(ctx, "A/B test not found; removing from state", map[string]any{"ab_test_id": abTestID})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading A/B test", "Could not read A/B test "+strconv.Itoa(int(abTestID))+": "+err.Error())
		return
	}

	// flattenABTestComputed only refreshes id/ab_test_id/status. It
	// deliberately preserves name/end_at/variants/metrics/configuration as
	// they are in state - see flattenABTestComputed and the resource
	// schema's description for why.
	resp.Diagnostics.Append(flattenABTestComputed(apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update should be unreachable in practice: every attribute Terraform could
// plan a change for is RequiresReplace, so Terraform plans a replace
// instead of calling Update. It is implemented defensively - re-reading
// current state - rather than left as a hard error, in case that
// invariant is ever violated by a future schema change.
func (r *abTestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ABTestResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	abTestID := int32(plan.ABTestID.ValueInt64())
	apiResp, err := client.GetABTest(client.NewApiGetABTestRequest(abTestID))
	if err != nil {
		resp.Diagnostics.AddError("Error reading A/B test", "Could not read A/B test "+strconv.Itoa(int(abTestID))+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(flattenABTestComputed(apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *abTestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ABTestResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	abTestID := int32(state.ABTestID.ValueInt64())
	tflog.Debug(ctx, "Deleting A/B test", map[string]any{"ab_test_id": abTestID})

	if _, err := client.DeleteABTest(client.NewApiDeleteABTestRequest(abTestID)); err != nil {
		var apiErr *abtestingapi.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return
		}

		resp.Diagnostics.AddError("Error deleting A/B test", "Could not delete A/B test "+strconv.Itoa(int(abTestID))+": "+err.Error())
	}
}

func (r *abTestResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	abTestID, err := strconv.ParseInt(req.ID, 10, 32)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected a numeric A/B test ID, got: "+req.ID+" ("+err.Error()+")")
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := client.GetABTest(client.NewApiGetABTestRequest(int32(abTestID)))
	if err != nil {
		resp.Diagnostics.AddError("Error importing A/B test", "Could not import A/B test "+req.ID+": "+err.Error())
		return
	}

	// flattenABTestImport seeds name/end_at/variants/configuration from the
	// enriched GetABTest response on a best-effort basis, and leaves
	// metrics null (unrecoverable). See its doc comment and the resource
	// schema's description for the full rationale; the next `terraform
	// plan` will likely show a diff (forcing a replace) until the user
	// reconciles these RequiresReplace attributes with configuration.
	var state ABTestResourceModel
	resp.Diagnostics.Append(flattenABTestImport(apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
