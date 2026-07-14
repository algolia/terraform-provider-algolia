package ingestion

import (
	"testing"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/utils"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenAuthentication_PopulatesNonSecretFields(t *testing.T) {
	platform := ingestionapi.PLATFORM_SHOPIFY
	auth := &ingestionapi.Authentication{
		AuthenticationID: "auth-1",
		Type:             ingestionapi.AUTHENTICATION_TYPE_BASIC,
		Name:             "my-auth",
		Platform:         *utils.NewNullable(&platform),
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-02T00:00:00Z",
	}

	var model AuthenticationResourceModel
	diags := flattenAuthentication(auth, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.ID.ValueString() != "auth-1" {
		t.Fatalf("id = %v, want auth-1", model.ID.ValueString())
	}
	if model.AuthenticationID.ValueString() != "auth-1" {
		t.Fatalf("authentication_id = %v, want auth-1", model.AuthenticationID.ValueString())
	}
	if model.Type.ValueString() != "basic" {
		t.Fatalf("type = %v, want basic", model.Type.ValueString())
	}
	if model.Name.ValueString() != "my-auth" {
		t.Fatalf("name = %v, want my-auth", model.Name.ValueString())
	}
	if model.Platform.ValueString() != "shopify" {
		t.Fatalf("platform = %v, want shopify", model.Platform.ValueString())
	}
	if model.CreatedAt.ValueString() != "2024-01-01T00:00:00Z" {
		t.Fatalf("created_at = %v, want 2024-01-01T00:00:00Z", model.CreatedAt.ValueString())
	}
	if model.UpdatedAt.ValueString() != "2024-01-02T00:00:00Z" {
		t.Fatalf("updated_at = %v, want 2024-01-02T00:00:00Z", model.UpdatedAt.ValueString())
	}
}

func TestFlattenAuthentication_NoPlatformIsNull(t *testing.T) {
	auth := &ingestionapi.Authentication{
		AuthenticationID: "auth-2",
		Type:             ingestionapi.AUTHENTICATION_TYPE_ALGOLIA,
		Name:             "my-auth",
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-01T00:00:00Z",
	}

	var model AuthenticationResourceModel
	diags := flattenAuthentication(auth, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.Platform.IsNull() {
		t.Fatalf("platform = %#v, want null", model.Platform)
	}
}

// TestFlattenAuthentication_PreservesInputAcrossRead is the critical
// redacted-secret regression test: GetAuthentication redacts secret values
// in its response (returning AuthInputPartial rather than the AuthInput
// that was configured), so flattening that response must never touch
// model.Input. A Read that clobbered Input with the redacted response would
// create a permanent diff against the real, user-configured credentials.
func TestFlattenAuthentication_PreservesInputAcrossRead(t *testing.T) {
	const realSecretInput = `{"appID": "APPID123", "apiKey": "the-real-secret"}`

	// Simulate state as it exists after a prior Create/Update: Input holds
	// the real credentials the user configured.
	model := AuthenticationResourceModel{
		Input: types.StringValue(realSecretInput),
	}

	// Simulate what GetAuthentication actually returns for this resource: a
	// redacted Input (Authentication.Input is AuthInputPartial, with secret
	// fields nulled out by the API), but real, current values everywhere
	// else.
	auth := &ingestionapi.Authentication{
		AuthenticationID: "auth-3",
		Type:             ingestionapi.AUTHENTICATION_TYPE_ALGOLIA,
		Name:             "my-auth-renamed-out-of-band",
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-03T00:00:00Z",
		// Note: Authentication.Input is intentionally left as its zero
		// value here (redacted), mirroring what the real API returns.
	}

	diags := flattenAuthentication(auth, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	// The non-secret fields must be refreshed from the API response.
	if model.Name.ValueString() != "my-auth-renamed-out-of-band" {
		t.Fatalf("name = %v, want my-auth-renamed-out-of-band", model.Name.ValueString())
	}
	if model.UpdatedAt.ValueString() != "2024-01-03T00:00:00Z" {
		t.Fatalf("updated_at = %v, want 2024-01-03T00:00:00Z", model.UpdatedAt.ValueString())
	}

	// Input must be untouched: still the real secret from state, not
	// whatever the (redacted) API response contained.
	if model.Input.ValueString() != realSecretInput {
		t.Fatalf("input = %v, want unchanged real secret %v", model.Input.ValueString(), realSecretInput)
	}
}

func TestFlattenAuthenticationDataSource_HasNoInputField(t *testing.T) {
	auth := &ingestionapi.Authentication{
		AuthenticationID: "auth-4",
		Type:             ingestionapi.AUTHENTICATION_TYPE_API_KEY,
		Name:             "my-auth",
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-01T00:00:00Z",
	}

	var model AuthenticationDataSourceModel
	diags := flattenAuthenticationDataSource(auth, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.AuthenticationID.ValueString() != "auth-4" {
		t.Fatalf("authentication_id = %v, want auth-4", model.AuthenticationID.ValueString())
	}
	if model.Type.ValueString() != "apiKey" {
		t.Fatalf("type = %v, want apiKey", model.Type.ValueString())
	}
}

func TestFlattenPlatform(t *testing.T) {
	t.Run("unset returns null", func(t *testing.T) {
		got := flattenPlatform(utils.Nullable[ingestionapi.Platform]{})
		if !got.IsNull() {
			t.Fatalf("got %#v, want null", got)
		}
	})

	t.Run("explicit null returns null", func(t *testing.T) {
		var nullable utils.Nullable[ingestionapi.Platform]
		nullable.Set(nil)

		got := flattenPlatform(nullable)
		if !got.IsNull() {
			t.Fatalf("got %#v, want null", got)
		}
	})

	t.Run("set value is returned", func(t *testing.T) {
		platform := ingestionapi.PLATFORM_COMMERCETOOLS
		got := flattenPlatform(*utils.NewNullable(&platform))
		if got.ValueString() != "commercetools" {
			t.Fatalf("got %v, want commercetools", got.ValueString())
		}
	})
}

// TestExpandFlattenAuthentication_RoundTrip exercises the JSON round trip
// end-to-end: JSON string -> AuthInput (Create) -> ... -> Authentication
// response -> flatten. This mirrors how the real resource lifecycle uses
// these two halves of the JSON-encoded-field pattern together.
func TestExpandFlattenAuthentication_RoundTrip(t *testing.T) {
	model := &AuthenticationResourceModel{
		Type:  types.StringValue(string(ingestionapi.AUTHENTICATION_TYPE_ALGOLIA)),
		Name:  types.StringValue("round-trip-auth"),
		Input: types.StringValue(`{"appID": "APPID123", "apiKey": "secret-key"}`),
	}

	create, diags := expandAuthenticationCreate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics expanding: %v", diags)
	}

	// The API would echo back the authenticationID/createdAt/updatedAt and
	// redact input; simulate that response.
	auth := &ingestionapi.Authentication{
		AuthenticationID: "auth-5",
		Type:             create.Type,
		Name:             create.Name,
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-01T00:00:00Z",
	}

	diags = flattenAuthentication(auth, model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics flattening: %v", diags)
	}

	if model.AuthenticationID.ValueString() != "auth-5" {
		t.Fatalf("authentication_id = %v, want auth-5", model.AuthenticationID.ValueString())
	}
	// Input survives the round trip unchanged, exactly as configured.
	if model.Input.ValueString() != `{"appID": "APPID123", "apiKey": "secret-key"}` {
		t.Fatalf("input = %v, want unchanged", model.Input.ValueString())
	}
}
