package abtest

import (
	"context"
	"strconv"

	abtestingapi "github.com/algolia/algoliasearch-client-go/v4/algolia/abtesting-v3"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

	createABTest(ctx, client, r.searchClient, &plan, resp)
}

// createABTest is the Create body, split out so that a unit test can drive it
// against an httptest-backed client (see resource_create_test.go) - the client
// base.client() builds always talks to the real, region-routed A/B Testing
// API. Create itself only resolves the plan and the client.
func createABTest(ctx context.Context, client *abtestingapi.APIClient, searchClient *search.APIClient, plan *ABTestResourceModel, resp *resource.CreateResponse) {
	addRequest, expandDiags := expandAddABTestsRequest(plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating A/B test", map[string]any{"name": plan.Name.ValueString()})

	createResp, err := client.AddABTests(client.NewApiAddABTestsRequest(addRequest), abtestingapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error creating A/B test", "Could not create A/B test "+plan.Name.ValueString()+": "+err.Error())
		return
	}

	// AddABTests has succeeded: a *paid* A/B test now exists in Algolia under
	// the server-assigned ID in createResp, and that ID is the only handle
	// Terraform will ever have on it. Persist it before anything that can
	// still fail, so that a failing read-back leaves a resource Terraform can
	// read, update and destroy instead of an unrecoverable orphan only the
	// dashboard can clean up. Same pattern as recommend.Create (see
	// internal/services/recommend/resource.go).
	//
	// status is the only other computed attribute and is still unknown here.
	// Terraform rejects an apply result containing unknown values, so it is
	// written as null; the read-back below fills it in, and if that fails the
	// next Read does. Every remaining attribute is Required or Optional, hence
	// already resolved in the plan.
	plan.ID = types.StringValue(strconv.FormatInt(int64(createResp.AbTestID), 10))
	plan.ABTestID = types.Int64Value(int64(createResp.AbTestID))
	plan.Status = types.StringNull()
	plan.CreatedAt = types.StringNull()
	plan.UpdatedAt = types.StringNull()
	plan.StoppedAt = types.StringNull()
	// configuration is Computed, so it is unknown here whenever the configuration
	// omitted it. Terraform rejects an applied state containing unknowns, and this
	// state has to survive a failure of the wait or read-back below, so resolve it
	// to null now; flattenABTestComputed fills in whatever Algolia chose.
	if plan.Configuration.IsUnknown() {
		plan.Configuration = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// AddABTests only queued the work. Wait for it before reading back, so the
	// response describes a test that actually exists, and so a caller that goes on
	// to touch the indexes involved is not blocked by a lock that has not lifted.
	if err := waitForABTestTask(ctx, searchClient, createResp); err != nil {
		resp.Diagnostics.AddError("Error creating A/B test", "Could not wait for A/B test creation to complete: "+err.Error())
		return
	}

	apiResp, err := client.GetABTest(client.NewApiGetABTestRequest(createResp.AbTestID), abtestingapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error reading A/B test", "Could not read back A/B test after creation: "+err.Error())
		return
	}

	// flattenABTestComputed only refreshes id/ab_test_id/status; it leaves
	// name/end_at/variants/metrics/configuration as already set in plan,
	// which is what Terraform requires of an apply (the applied value of a
	// Required attribute must equal the planned one). Refreshing name/end_at
	// is Read's job - see flattenABTestRead.
	resp.Diagnostics.Append(flattenABTestComputed(apiResp, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
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
	apiResp, err := client.GetABTest(client.NewApiGetABTestRequest(abTestID), abtestingapi.WithContext(ctx))
	if err != nil {
		if algoliaerr.IsNotFound(err) {
			tflog.Warn(ctx, "A/B test not found; removing from state", map[string]any{"ab_test_id": abTestID})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading A/B test", "Could not read A/B test "+strconv.Itoa(int(abTestID))+": "+err.Error())
		return
	}

	// flattenABTestRead refreshes id/ab_test_id/status plus name/end_at, so
	// a test renamed or rescheduled outside Terraform surfaces as drift. It
	// deliberately preserves variants/metrics/configuration as they are in
	// state, because GetABTest's shape for those diverges from the create
	// shape - see flattenABTestRead and the resource schema's description.
	resp.Diagnostics.Append(flattenABTestRead(apiResp, &state)...)
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
	apiResp, err := client.GetABTest(client.NewApiGetABTestRequest(abTestID), abtestingapi.WithContext(ctx))
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

	deleteResp, err := client.DeleteABTest(client.NewApiDeleteABTestRequest(abTestID), abtestingapi.WithContext(ctx))
	if err != nil {
		if algoliaerr.IsNotFound(err) {
			return
		}

		resp.Diagnostics.AddError("Error deleting A/B test", "Could not delete A/B test "+strconv.Itoa(int(abTestID))+": "+err.Error())
		return
	}

	// Wait for the queued deletion before returning. Without this, Terraform moves
	// straight on to destroying the indexes this test referenced and Algolia
	// rejects them with "cannot delete with an index under AB testing index as
	// destination", failing the destroy and leaving the indexes behind.
	if err := waitForABTestTask(ctx, r.searchClient, deleteResp); err != nil {
		resp.Diagnostics.AddError("Error deleting A/B test", "Could not wait for A/B test deletion to complete: "+err.Error())
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

	apiResp, err := client.GetABTest(client.NewApiGetABTestRequest(int32(abTestID)), abtestingapi.WithContext(ctx))
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
