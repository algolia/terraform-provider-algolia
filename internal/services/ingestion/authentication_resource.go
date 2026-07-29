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
	_ resource.Resource                = &authenticationResource{}
	_ resource.ResourceWithConfigure   = &authenticationResource{}
	_ resource.ResourceWithImportState = &authenticationResource{}
)

// authenticationResource manages an algolia_ingestion_authentication
// resource. It embeds base (see client.go) for Configure-time wiring and an
// on-demand region-routed Ingestion client.
type authenticationResource struct {
	base
}

// NewAuthenticationResource returns the algolia_ingestion_authentication resource.
func NewAuthenticationResource() resource.Resource {
	return &authenticationResource{}
}

func (r *authenticationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingestion_authentication"
}

func (r *authenticationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = authenticationResourceSchema()
}

func (r *authenticationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(r.configure(req.ProviderData)...)
}

func (r *authenticationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AuthenticationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createAuthentication(ctx, client, &plan, resp)
}

// createAuthentication is the Create body, split out so that a unit test can
// drive it against an httptest-backed client (see identity_state_test.go) -
// the client base.client() builds always talks to the real, region-routed
// Ingestion API. Create itself only resolves the plan and the client. Same
// pattern as abtest.createABTest.
func createAuthentication(ctx context.Context, client *ingestionapi.APIClient, plan *AuthenticationResourceModel, resp *resource.CreateResponse) {
	create, expandDiags := expandAuthenticationCreate(plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Ingestion authentication", map[string]any{
		"name": plan.Name.ValueString(),
		"type": plan.Type.ValueString(),
	})

	createResp, err := client.CreateAuthentication(client.NewApiCreateAuthenticationRequest(create), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error creating Ingestion authentication", "Could not create authentication "+plan.Name.ValueString()+": "+err.Error())
		return
	}

	// The authentication resource now exists in Algolia under the
	// server-assigned UUID in createResp, which is the only handle Terraform
	// will ever have on it. Persist it before the read-back below, so that a
	// failure there leaves a resource Terraform can read, update and destroy
	// instead of orphaning credentials that exist remotely but not in state:
	// the next apply would create a second authentication resource and nothing
	// would ever adopt the first.
	resp.Diagnostics.Append(resp.State.Set(ctx, newAuthenticationIdentityState(*plan, createResp.AuthenticationID))...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := client.GetAuthentication(client.NewApiGetAuthenticationRequest(createResp.AuthenticationID), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ingestion authentication", "Could not read back authentication after creation: "+err.Error())
		return
	}

	// flattenAuthentication does not touch plan.Input; it keeps the value
	// already in the plan (the credentials the user just configured).
	resp.Diagnostics.Append(flattenAuthentication(apiResp, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// newAuthenticationIdentityState returns the state to persist immediately
// after a successful CreateAuthentication, before the read-back that can still
// fail. Terraform rejects an apply result containing unknown values, so every
// unknown has to be resolved here: id/authentication_id come from the create
// response, while created_at/updated_at are knowable only from the read-back
// and are therefore written as null - the next Read fills them in. Every other
// attribute is Required or Optional-only, so the plan holds the configuration
// verbatim and is already known; the model has no Object/List/Set/Map
// attribute, so no typed null has to be constructed.
func newAuthenticationIdentityState(plan AuthenticationResourceModel, authenticationID string) AuthenticationResourceModel {
	plan.ID = types.StringValue(authenticationID)
	plan.AuthenticationID = types.StringValue(authenticationID)
	plan.CreatedAt = types.StringNull()
	plan.UpdatedAt = types.StringNull()

	return plan
}

func (r *authenticationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AuthenticationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	authenticationID := state.AuthenticationID.ValueString()
	apiResp, err := client.GetAuthentication(client.NewApiGetAuthenticationRequest(authenticationID), ingestionapi.WithContext(ctx))
	if err != nil {
		if algoliaerr.IsNotFound(err) {
			tflog.Warn(ctx, "Ingestion authentication not found; removing from state", map[string]any{"authentication_id": authenticationID})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading Ingestion authentication", "Could not read authentication "+authenticationID+": "+err.Error())
		return
	}

	// flattenAuthentication does not touch state.Input: GetAuthentication
	// redacts secret values, so this Read only refreshes name/type/
	// platform/created_at/updated_at and preserves the previously
	// configured input as-is. See the input attribute's schema description.
	resp.Diagnostics.Append(flattenAuthentication(apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *authenticationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AuthenticationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	update, expandDiags := expandAuthenticationUpdate(&plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	authenticationID := plan.AuthenticationID.ValueString()
	tflog.Debug(ctx, "Updating Ingestion authentication", map[string]any{"authentication_id": authenticationID})

	if _, err := client.UpdateAuthentication(client.NewApiUpdateAuthenticationRequest(authenticationID, update), ingestionapi.WithContext(ctx)); err != nil {
		resp.Diagnostics.AddError("Error updating Ingestion authentication", "Could not update authentication "+authenticationID+": "+err.Error())
		return
	}

	apiResp, err := client.GetAuthentication(client.NewApiGetAuthenticationRequest(authenticationID), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ingestion authentication", "Could not read back authentication after update: "+err.Error())
		return
	}

	resp.Diagnostics.Append(flattenAuthentication(apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *authenticationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AuthenticationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	authenticationID := state.AuthenticationID.ValueString()
	tflog.Debug(ctx, "Deleting Ingestion authentication", map[string]any{"authentication_id": authenticationID})

	if _, err := client.DeleteAuthentication(client.NewApiDeleteAuthenticationRequest(authenticationID), ingestionapi.WithContext(ctx)); err != nil {
		if algoliaerr.IsNotFound(err) {
			return
		}

		resp.Diagnostics.AddError("Error deleting Ingestion authentication", "Could not delete authentication "+authenticationID+": "+err.Error())
	}
}

func (r *authenticationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	authenticationID := req.ID
	apiResp, err := client.GetAuthentication(client.NewApiGetAuthenticationRequest(authenticationID), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error importing Ingestion authentication", "Could not import authentication "+authenticationID+": "+err.Error())
		return
	}

	var state AuthenticationResourceModel
	// input cannot be recovered on import: GetAuthentication redacts secret
	// values. Leave it null; the next plan will show a diff until the user
	// sets input explicitly in configuration and applies it.
	state.Input = types.StringNull()
	resp.Diagnostics.Append(flattenAuthentication(apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
