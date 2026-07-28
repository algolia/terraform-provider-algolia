package ingestion

import (
	"encoding/json"
	"testing"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The AuthInput and AuthInputPartial oneOf types in algoliasearch-client-go
// carry no discriminator field. Their generated UnmarshalJSON tries every
// variant in turn, never returns early, and no variant struct rejects unknown
// or missing keys - so decoding leaves several pointers non-nil at once.
// MarshalJSON then serializes whichever pointer comes first in its own fixed
// order, which is how a "basic" authentication used to reach the API as
// {"apiKey":"","appID":""}.
//
// These tests assert on the bytes the expand path actually puts on the wire,
// not on which pointer happens to be set: constructing the union through the
// client's ...AsAuthInput helpers sets exactly one pointer, a shape real
// decoding never produces, and therefore never catches the defect.

func TestExpandAuthenticationCreate_MarshalsDeclaredVariant(t *testing.T) {
	tests := []struct {
		name     string
		authType ingestionapi.AuthenticationType
		input    string
	}{
		{
			name:     "algolia",
			authType: ingestionapi.AUTHENTICATION_TYPE_ALGOLIA,
			input:    `{"appID":"APPID123","apiKey":"algolia-secret"}`,
		},
		{
			name:     "algoliaInsights",
			authType: ingestionapi.AUTHENTICATION_TYPE_ALGOLIA_INSIGHTS,
			input:    `{"appID":"APPID123","apiKey":"insights-secret"}`,
		},
		{
			name:     "apiKey",
			authType: ingestionapi.AUTHENTICATION_TYPE_API_KEY,
			input:    `{"key":"shpat_example_api_key"}`,
		},
		{
			name:     "basic",
			authType: ingestionapi.AUTHENTICATION_TYPE_BASIC,
			input:    `{"username":"admin","password":"s3cret"}`,
		},
		{
			name:     "googleServiceAccount",
			authType: ingestionapi.AUTHENTICATION_TYPE_GOOGLE_SERVICE_ACCOUNT,
			input:    `{"clientEmail":"svc@example.iam.gserviceaccount.com","privateKey":"-----BEGIN PRIVATE KEY-----"}`,
		},
		{
			name:     "oauth",
			authType: ingestionapi.AUTHENTICATION_TYPE_OAUTH,
			input:    `{"url":"https://example.com/oauth/token","client_id":"cid","client_secret":"csecret","code":"authcode","scope":"read"}`,
		},
		{
			name:     "secrets",
			authType: ingestionapi.AUTHENTICATION_TYPE_SECRETS,
			input:    `{"token":"abc123","region":"eu"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &AuthenticationResourceModel{
				Type:  types.StringValue(string(tt.authType)),
				Name:  types.StringValue("auth-" + tt.name),
				Input: types.StringValue(tt.input),
			}

			create, diags := expandAuthenticationCreate(model)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags.Errors())
			}

			encoded, err := json.Marshal(create.Input)
			if err != nil {
				t.Fatalf("marshaling AuthInput: %v", err)
			}

			if !jsonSemanticallyEqual(string(encoded), tt.input) {
				t.Fatalf("AuthInput marshaled to %s, want %s", encoded, tt.input)
			}
		})
	}
}

func TestExpandAuthenticationUpdate_MarshalsDeclaredVariant(t *testing.T) {
	tests := []struct {
		name     string
		authType ingestionapi.AuthenticationType
		input    string
	}{
		{
			name:     "algolia",
			authType: ingestionapi.AUTHENTICATION_TYPE_ALGOLIA,
			input:    `{"appID":"APPID123","apiKey":"rotated-secret"}`,
		},
		{
			name:     "algoliaInsights",
			authType: ingestionapi.AUTHENTICATION_TYPE_ALGOLIA_INSIGHTS,
			input:    `{"appID":"APPID123","apiKey":"rotated-insights"}`,
		},
		{
			name:     "apiKey",
			authType: ingestionapi.AUTHENTICATION_TYPE_API_KEY,
			input:    `{"key":"rotated-api-key"}`,
		},
		{
			name:     "basic",
			authType: ingestionapi.AUTHENTICATION_TYPE_BASIC,
			input:    `{"username":"admin","password":"rotated"}`,
		},
		{
			name:     "googleServiceAccount",
			authType: ingestionapi.AUTHENTICATION_TYPE_GOOGLE_SERVICE_ACCOUNT,
			input:    `{"clientEmail":"svc@example.iam.gserviceaccount.com","privateKey":"-----BEGIN PRIVATE KEY-----"}`,
		},
		{
			name:     "oauth",
			authType: ingestionapi.AUTHENTICATION_TYPE_OAUTH,
			input:    `{"url":"https://example.com/oauth/token","client_id":"cid","client_secret":"rotated","code":"authcode"}`,
		},
		{
			name:     "secrets",
			authType: ingestionapi.AUTHENTICATION_TYPE_SECRETS,
			input:    `{"token":"rotated","region":"eu"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &AuthenticationResourceModel{
				Type:  types.StringValue(string(tt.authType)),
				Name:  types.StringValue("auth-" + tt.name),
				Input: types.StringValue(tt.input),
			}

			update, diags := expandAuthenticationUpdate(model)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags.Errors())
			}

			encoded, err := json.Marshal(update.Input)
			if err != nil {
				t.Fatalf("marshaling AuthInputPartial: %v", err)
			}

			if !jsonSemanticallyEqual(string(encoded), tt.input) {
				t.Fatalf("AuthInputPartial marshaled to %s, want %s", encoded, tt.input)
			}
		})
	}
}

// TestExpandAuthInput_RejectsInputForAnotherType is the loud-failure half of
// the fix: an `input` that does not match the declared `type` must produce a
// diagnostic rather than silently reaching the API as a different variant.
func TestExpandAuthInput_RejectsInputForAnotherType(t *testing.T) {
	model := &AuthenticationResourceModel{
		Type:  types.StringValue(string(ingestionapi.AUTHENTICATION_TYPE_ALGOLIA)),
		Name:  types.StringValue("mismatched"),
		Input: types.StringValue(`{"username":"admin","password":"s3cret"}`),
	}

	if _, diags := expandAuthenticationCreate(model); !diags.HasError() {
		t.Fatal("expected a diagnostic for input that does not match the declared type")
	}

	if _, diags := expandAuthenticationUpdate(model); !diags.HasError() {
		t.Fatal("expected a diagnostic for input that does not match the declared type on update")
	}
}

func TestExpandAuthInput_RejectsUnknownType(t *testing.T) {
	model := &AuthenticationResourceModel{
		Type:  types.StringValue("someFutureType"),
		Name:  types.StringValue("unknown"),
		Input: types.StringValue(`{"appID":"APPID123","apiKey":"secret"}`),
	}

	if _, diags := expandAuthenticationCreate(model); !diags.HasError() {
		t.Fatal("expected a diagnostic for an authentication type the provider cannot map to a variant")
	}

	if _, diags := expandAuthenticationUpdate(model); !diags.HasError() {
		t.Fatal("expected a diagnostic for an authentication type the provider cannot map to a variant on update")
	}
}
