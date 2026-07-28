package ingestion

import (
	"testing"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExpandAuthenticationCreate_Algolia(t *testing.T) {
	model := &AuthenticationResourceModel{
		Type:     types.StringValue(string(ingestionapi.AUTHENTICATION_TYPE_ALGOLIA)),
		Name:     types.StringValue("my-algolia-auth"),
		Platform: types.StringNull(),
		Input:    types.StringValue(`{"appID": "APPID123", "apiKey": "secret-key"}`),
	}

	create, diags := expandAuthenticationCreate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if create.Type != ingestionapi.AUTHENTICATION_TYPE_ALGOLIA {
		t.Fatalf("type = %v, want algolia", create.Type)
	}
	if create.Name != "my-algolia-auth" {
		t.Fatalf("name = %v, want my-algolia-auth", create.Name)
	}
	if create.Platform.IsSet() {
		t.Fatalf("expected platform to be unset, got %#v", create.Platform)
	}

	authAlgolia := create.Input.AuthAlgolia
	if authAlgolia == nil {
		t.Fatalf("expected input to decode into AuthAlgolia, got %#v", create.Input)
	}
	if authAlgolia.AppID != "APPID123" || authAlgolia.ApiKey != "secret-key" {
		t.Fatalf("authAlgolia = %#v, want AppID=APPID123 ApiKey=secret-key", authAlgolia)
	}
}

func TestExpandAuthenticationCreate_APIKey(t *testing.T) {
	model := &AuthenticationResourceModel{
		Type:  types.StringValue(string(ingestionapi.AUTHENTICATION_TYPE_API_KEY)),
		Name:  types.StringValue("my-api-key-auth"),
		Input: types.StringValue(`{"key": "some-secret-api-key"}`),
	}

	create, diags := expandAuthenticationCreate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	authAPIKey := create.Input.AuthAPIKey
	if authAPIKey == nil {
		t.Fatalf("expected input to decode into AuthAPIKey, got %#v", create.Input)
	}
	if authAPIKey.Key != "some-secret-api-key" {
		t.Fatalf("authAPIKey.Key = %v, want some-secret-api-key", authAPIKey.Key)
	}
}

func TestExpandAuthenticationCreate_WithPlatform(t *testing.T) {
	model := &AuthenticationResourceModel{
		Type:     types.StringValue(string(ingestionapi.AUTHENTICATION_TYPE_BASIC)),
		Name:     types.StringValue("my-basic-auth"),
		Platform: types.StringValue(string(ingestionapi.PLATFORM_SHOPIFY)),
		Input:    types.StringValue(`{"username": "user", "password": "pass"}`),
	}

	create, diags := expandAuthenticationCreate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !create.Platform.IsSet() || create.Platform.Get() == nil || *create.Platform.Get() != ingestionapi.PLATFORM_SHOPIFY {
		t.Fatalf("platform = %#v, want set to shopify", create.Platform)
	}

	authBasic := create.Input.AuthBasic
	if authBasic == nil || authBasic.Username != "user" || authBasic.Password != "pass" {
		t.Fatalf("authBasic = %#v, want Username=user Password=pass", authBasic)
	}
}

func TestExpandAuthenticationCreate_InvalidInputJSON(t *testing.T) {
	model := &AuthenticationResourceModel{
		Type:  types.StringValue(string(ingestionapi.AUTHENTICATION_TYPE_ALGOLIA)),
		Name:  types.StringValue("broken"),
		Input: types.StringValue(`{not valid json`),
	}

	_, diags := expandAuthenticationCreate(model)
	if !diags.HasError() {
		t.Fatal("expected a diagnostic error for invalid input JSON")
	}
}

func TestExpandAuthenticationUpdate(t *testing.T) {
	model := &AuthenticationResourceModel{
		// `type` is RequiresReplace, so the plan always carries the
		// authentication's real type - which is what selects the
		// AuthInputPartial variant.
		Type:  types.StringValue(string(ingestionapi.AUTHENTICATION_TYPE_ALGOLIA)),
		Name:  types.StringValue("renamed"),
		Input: types.StringValue(`{"appID": "APPID123", "apiKey": "new-secret"}`),
	}

	update, diags := expandAuthenticationUpdate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if update.Name == nil || *update.Name != "renamed" {
		t.Fatalf("name = %#v, want renamed", update.Name)
	}
	if update.Input == nil || update.Input.AuthAlgoliaPartial == nil {
		t.Fatalf("expected input to decode into AuthAlgoliaPartial, got %#v", update.Input)
	}
	apiKey := update.Input.AuthAlgoliaPartial.ApiKey
	if apiKey == nil || *apiKey != "new-secret" {
		t.Fatalf("apiKey = %#v, want new-secret", apiKey)
	}
}

func TestExpandAuthenticationUpdate_InvalidInputJSON(t *testing.T) {
	model := &AuthenticationResourceModel{
		Type:  types.StringValue(string(ingestionapi.AUTHENTICATION_TYPE_ALGOLIA)),
		Name:  types.StringValue("renamed"),
		Input: types.StringValue(`not json at all`),
	}

	_, diags := expandAuthenticationUpdate(model)
	if !diags.HasError() {
		t.Fatal("expected a diagnostic error for invalid input JSON")
	}
}

func TestExpandPlatform(t *testing.T) {
	t.Run("null returns unset", func(t *testing.T) {
		platform := expandPlatform(types.StringNull())
		if platform.IsSet() {
			t.Fatalf("expected unset platform, got %#v", platform)
		}
	})

	t.Run("unknown returns unset", func(t *testing.T) {
		platform := expandPlatform(types.StringUnknown())
		if platform.IsSet() {
			t.Fatalf("expected unset platform, got %#v", platform)
		}
	})

	t.Run("value is set", func(t *testing.T) {
		platform := expandPlatform(types.StringValue("bigcommerce"))
		if !platform.IsSet() || platform.Get() == nil || *platform.Get() != ingestionapi.PLATFORM_BIGCOMMERCE {
			t.Fatalf("platform = %#v, want set to bigcommerce", platform)
		}
	})
}
