package apikey

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

	key := createResp.GetKey()
	ctx = maskKeyValue(ctx, key)

	if _, err = r.client.WaitForApiKey(key, search.API_KEY_OPERATION_ADD); err != nil {
		resp.Diagnostics.AddError("Error waiting for API key creation", "Could not confirm API key creation: "+redactKey(err, key))
		return
	}

	readResp, err := r.client.GetApiKey(r.client.NewApiGetApiKeyRequest(key))
	if err != nil {
		resp.Diagnostics.AddError("Error reading API key", "Could not read "+keyLabel(plan.Description)+" after creation: "+redactKey(err, key))
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
	ctx = maskKeyValue(ctx, key)
	// The id is the key value itself, so it is never logged (see keyLabel).
	tflog.Debug(ctx, "Reading API key")

	apiResp, err := r.client.GetApiKey(r.client.NewApiGetApiKeyRequest(key))
	if err != nil {
		var apiErr *search.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			tflog.Warn(ctx, "API key not found; removing from state")
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading API key", "Could not read "+keyLabel(state.Description)+": "+redactKey(err, key))
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
	ctx = maskKeyValue(ctx, key)

	apiKey, diags := buildAPIKeyRequest(&plan, r.now().UTC())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	expiresAt, expiryDiags := parseExpiresAt(&plan)
	resp.Diagnostics.Append(expiryDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.UpdateApiKey(r.client.NewApiUpdateApiKeyRequest(key, apiKey)); err != nil {
		resp.Diagnostics.AddError("Error updating API key", "Could not update "+keyLabel(plan.Description)+": "+redactKey(err, key))
		return
	}

	if _, err := search.CreateIterable(
		func(*search.GetApiKeyResponse, error) (*search.GetApiKeyResponse, error) {
			return r.client.GetApiKey(r.client.NewApiGetApiKeyRequest(key))
		},
		func(apiResp *search.GetApiKeyResponse, err error) (bool, error) {
			if err != nil {
				return false, err
			}

			return apiKeyResponseMatches(apiResp, apiKey, expiresAt, r.now().UTC()), nil
		},
		search.WithTimeout(func(count int) time.Duration {
			return time.Duration(min(200*count, 5000)) * time.Millisecond
		}),
		search.WithMaxRetries(50),
	); err != nil {
		resp.Diagnostics.AddError("Error waiting for API key update", "Could not confirm API key update: "+redactKey(err, key))
		return
	}

	readResp, err := r.client.GetApiKey(r.client.NewApiGetApiKeyRequest(key))
	if err != nil {
		resp.Diagnostics.AddError("Error reading API key", "Could not read "+keyLabel(plan.Description)+": "+redactKey(err, key))
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
	ctx = maskKeyValue(ctx, key)
	// The id is the key value itself, so it is never logged (see keyLabel).
	tflog.Debug(ctx, "Deleting API key")

	if _, err := r.client.DeleteApiKey(r.client.NewApiDeleteApiKeyRequest(key)); err != nil {
		var apiErr *search.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return
		}

		resp.Diagnostics.AddError("Error deleting API key", "Could not delete "+keyLabel(state.Description)+": "+redactKey(err, key))
		return
	}

	if _, err := r.client.WaitForApiKey(key, search.API_KEY_OPERATION_DELETE); err != nil {
		var apiErr *search.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return
		}

		resp.Diagnostics.AddError("Error waiting for API key deletion", "Could not confirm API key deletion: "+redactKey(err, key))
		return
	}
}

func (r *apiKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is the key value itself, so it is masked before any logging
	// and never interpolated into a diagnostic.
	ctx = maskKeyValue(ctx, req.ID)

	apiResp, err := r.client.GetApiKey(r.client.NewApiGetApiKeyRequest(req.ID))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing API key",
			"Could not import the API key from the supplied import ID: "+redactKey(err, req.ID),
		)
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

// keyLabel returns a non-secret way to refer to an API key in a diagnostic.
//
// This resource's id *is* the key value, i.e. a live credential, so it must never
// be interpolated into a diagnostic or a log field: Sensitive only governs how
// Terraform renders state, not diagnostics or TF_LOG output. The description is
// operator-supplied metadata and is safe to echo; without one there is nothing
// safe left to identify the key by. The algolia_api_key data source already
// follows this rule (see data_source.go).
func keyLabel(description types.String) string {
	if description.IsNull() || description.IsUnknown() || description.ValueString() == "" {
		return "the API key"
	}

	return fmt.Sprintf("the API key described as %q", description.ValueString())
}

// redactKey removes the key value from an error message. Transport-level errors
// wrap the request URL and the key endpoints are /1/keys/{key}, so a raw error
// string can carry the credential into a diagnostic even when the caller never
// interpolates it itself.
func redactKey(err error, key string) string {
	if err == nil {
		return ""
	}
	if key == "" {
		return err.Error()
	}

	return strings.ReplaceAll(err.Error(), key, "***")
}

// maskKeyValue registers the key value with tflog so that any log line emitted on
// the returned context has it replaced with asterisks, whichever message or field
// it reaches. This has to be applied per-RPC rather than once in
// provider.Configure: masking options live on the context, and the framework
// discards the context it hands to Configure (see
// fwserver.Server.ConfigureProvider), so options registered there never reach
// resource operations.
func maskKeyValue(ctx context.Context, key string) context.Context {
	if key == "" {
		return ctx
	}

	return tflog.MaskLogStrings(ctx, key)
}
