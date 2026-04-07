package apikey

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &apiKeyResource{}
	_ resource.ResourceWithConfigure   = &apiKeyResource{}
	_ resource.ResourceWithImportState = &apiKeyResource{}
)

type apiKeyResource struct {
	client *search.APIClient
	now    func() time.Time
}

func NewResource() resource.Resource {
	return &apiKeyResource{
		now: time.Now,
	}
}

func (r *apiKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *apiKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = apiKeyResourceSchema()
}

func (r *apiKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *apiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan APIKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey, diags := buildAPIKeyRequest(&plan, r.now().UTC())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createResp, err := r.client.AddApiKey(r.client.NewApiAddApiKeyRequest(apiKey))
	if err != nil {
		resp.Diagnostics.AddError("Error creating API key", "Could not create API key: "+err.Error())
		return
	}

	if _, err = r.client.WaitForApiKey(createResp.GetKey(), search.API_KEY_OPERATION_ADD); err != nil {
		resp.Diagnostics.AddError("Error waiting for API key creation", "Could not confirm API key creation: "+err.Error())
		return
	}

	readResp, err := r.client.GetApiKey(r.client.NewApiGetApiKeyRequest(createResp.GetKey()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading API key", "Could not read API key "+createResp.GetKey()+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(hydrateAPIKeyModel(readResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state APIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := state.ID.ValueString()
	tflog.Debug(ctx, "Reading API key", map[string]any{"id": key})

	apiResp, err := r.client.GetApiKey(r.client.NewApiGetApiKeyRequest(key))
	if err != nil {
		var apiErr *search.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			tflog.Warn(ctx, "API key not found; removing from state", map[string]any{"id": key})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading API key", "Could not read API key "+key+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(hydrateAPIKeyModel(apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *apiKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan APIKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := plan.ID.ValueString()
	apiKey, diags := buildAPIKeyRequest(&plan, r.now().UTC())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.UpdateApiKey(r.client.NewApiUpdateApiKeyRequest(key, apiKey)); err != nil {
		resp.Diagnostics.AddError("Error updating API key", "Could not update API key "+key+": "+err.Error())
		return
	}

	if _, err := r.client.WaitForApiKey(key, search.API_KEY_OPERATION_UPDATE, search.WithApiKey(apiKey)); err != nil {
		resp.Diagnostics.AddError("Error waiting for API key update", "Could not confirm API key update: "+err.Error())
		return
	}

	readResp, err := r.client.GetApiKey(r.client.NewApiGetApiKeyRequest(key))
	if err != nil {
		resp.Diagnostics.AddError("Error reading API key", "Could not read API key "+key+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(hydrateAPIKeyModel(readResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state APIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := state.ID.ValueString()

	if _, err := r.client.DeleteApiKey(r.client.NewApiDeleteApiKeyRequest(key)); err != nil {
		var apiErr *search.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return
		}

		resp.Diagnostics.AddError("Error deleting API key", "Could not delete API key "+key+": "+err.Error())
		return
	}

	if _, err := r.client.WaitForApiKey(key, search.API_KEY_OPERATION_DELETE); err != nil {
		var apiErr *search.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return
		}

		resp.Diagnostics.AddError("Error waiting for API key deletion", "Could not confirm API key deletion: "+err.Error())
		return
	}
}

func (r *apiKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	apiResp, err := r.client.GetApiKey(r.client.NewApiGetApiKeyRequest(req.ID))
	if err != nil {
		resp.Diagnostics.AddError("Error importing API key", "Could not import API key "+req.ID+": "+err.Error())
		return
	}

	state := APIKeyResourceModel{
		ExpiresAt: types.StringNull(),
	}
	resp.Diagnostics.Append(hydrateAPIKeyModel(apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
