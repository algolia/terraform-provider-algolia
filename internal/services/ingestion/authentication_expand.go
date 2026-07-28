package ingestion

import (
	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/utils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// expandAuthenticationCreate converts the Terraform plan into an
// AuthenticationCreate request body for CreateAuthentication.
func expandAuthenticationCreate(model *AuthenticationResourceModel) (*ingestionapi.AuthenticationCreate, diag.Diagnostics) {
	var diags diag.Diagnostics

	input, inputDiags := expandAuthInput(model.Type.ValueString(), model.Input)
	diags.Append(inputDiags...)
	if diags.HasError() {
		return nil, diags
	}

	create := ingestionapi.NewAuthenticationCreate(
		ingestionapi.AuthenticationType(model.Type.ValueString()),
		model.Name.ValueString(),
		input,
	)
	create.Platform = expandPlatform(model.Platform)

	return create, diags
}

// expandAuthenticationUpdate converts the Terraform plan into an
// AuthenticationUpdate request body for UpdateAuthentication.
//
// AuthenticationUpdate only accepts name and input - there is no way to
// change type or platform after creation, which is why both are marked
// RequiresReplace in the resource schema.
func expandAuthenticationUpdate(model *AuthenticationResourceModel) (*ingestionapi.AuthenticationUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics

	inputPartial, inputDiags := expandAuthInputPartial(model.Type.ValueString(), model.Input)
	diags.Append(inputDiags...)
	if diags.HasError() {
		return nil, diags
	}

	update := ingestionapi.NewAuthenticationUpdate(
		ingestionapi.WithAuthenticationUpdateName(model.Name.ValueString()),
		ingestionapi.WithAuthenticationUpdateInput(inputPartial),
	)

	return update, diags
}

// expandAuthInput decodes the `input` attribute into the AuthInput union
// expected by AuthenticationCreate, selecting the variant from the declared
// authentication `type`.
//
// The variant cannot be inferred from the JSON alone. AuthInput is a generated
// oneOf with no discriminator field: its UnmarshalJSON tries every variant,
// never returns early, and no variant struct rejects unknown or missing keys,
// so decoding leaves several pointers non-nil at once. MarshalJSON then
// serializes whichever pointer comes first in its own fixed order - AuthAlgolia
// for any payload without a "key" field - which is how "basic", "oauth",
// "googleServiceAccount" and "secrets" credentials reached the API as
// {"apiKey":"","appID":""}. `type` is the discriminator the API itself uses, so
// it is what selects the variant here.
func expandAuthInput(authType string, input types.String) (ingestionapi.AuthInput, diag.Diagnostics) {
	raw := []byte(input.ValueString())

	switch ingestionapi.AuthenticationType(authType) {
	case ingestionapi.AUTHENTICATION_TYPE_ALGOLIA:
		var variant ingestionapi.AuthAlgolia
		if diags := decodeAuthInput(raw, &variant, authType); diags.HasError() {
			return ingestionapi.AuthInput{}, diags
		}

		return *ingestionapi.AuthAlgoliaAsAuthInput(&variant), nil
	case ingestionapi.AUTHENTICATION_TYPE_ALGOLIA_INSIGHTS:
		var variant ingestionapi.AuthAlgoliaInsights
		if diags := decodeAuthInput(raw, &variant, authType); diags.HasError() {
			return ingestionapi.AuthInput{}, diags
		}

		return *ingestionapi.AuthAlgoliaInsightsAsAuthInput(&variant), nil
	case ingestionapi.AUTHENTICATION_TYPE_API_KEY:
		var variant ingestionapi.AuthAPIKey
		if diags := decodeAuthInput(raw, &variant, authType); diags.HasError() {
			return ingestionapi.AuthInput{}, diags
		}

		return *ingestionapi.AuthAPIKeyAsAuthInput(&variant), nil
	case ingestionapi.AUTHENTICATION_TYPE_BASIC:
		var variant ingestionapi.AuthBasic
		if diags := decodeAuthInput(raw, &variant, authType); diags.HasError() {
			return ingestionapi.AuthInput{}, diags
		}

		return *ingestionapi.AuthBasicAsAuthInput(&variant), nil
	case ingestionapi.AUTHENTICATION_TYPE_GOOGLE_SERVICE_ACCOUNT:
		var variant ingestionapi.AuthGoogleServiceAccount
		if diags := decodeAuthInput(raw, &variant, authType); diags.HasError() {
			return ingestionapi.AuthInput{}, diags
		}

		return *ingestionapi.AuthGoogleServiceAccountAsAuthInput(&variant), nil
	case ingestionapi.AUTHENTICATION_TYPE_OAUTH:
		var variant ingestionapi.AuthOAuth
		if diags := decodeAuthInput(raw, &variant, authType); diags.HasError() {
			return ingestionapi.AuthInput{}, diags
		}

		return *ingestionapi.AuthOAuthAsAuthInput(&variant), nil
	case ingestionapi.AUTHENTICATION_TYPE_SECRETS:
		var variant map[string]string
		if diags := decodeAuthInput(raw, &variant, authType); diags.HasError() {
			return ingestionapi.AuthInput{}, diags
		}

		return *ingestionapi.MapmapOfStringstringAsAuthInput(variant), nil
	default:
		return ingestionapi.AuthInput{}, unsupportedAuthTypeDiags(authType)
	}
}

// expandAuthInputPartial is the AuthenticationUpdate counterpart of
// expandAuthInput: the update endpoint accepts AuthInputPartial, whose variants
// mirror AuthInput's with every field optional. It has the same
// missing-discriminator defect - worse, in fact, since AuthAlgoliaInsightsPartial
// has no required field and therefore marshals to {} - so the variant is again
// selected from `type`, which the schema marks RequiresReplace and so always
// reflects the authentication's real type.
func expandAuthInputPartial(authType string, input types.String) (ingestionapi.AuthInputPartial, diag.Diagnostics) {
	raw := []byte(input.ValueString())

	switch ingestionapi.AuthenticationType(authType) {
	case ingestionapi.AUTHENTICATION_TYPE_ALGOLIA:
		var variant ingestionapi.AuthAlgoliaPartial
		if diags := decodeAuthInput(raw, &variant, authType); diags.HasError() {
			return ingestionapi.AuthInputPartial{}, diags
		}

		return *ingestionapi.AuthAlgoliaPartialAsAuthInputPartial(&variant), nil
	case ingestionapi.AUTHENTICATION_TYPE_ALGOLIA_INSIGHTS:
		var variant ingestionapi.AuthAlgoliaInsightsPartial
		if diags := decodeAuthInput(raw, &variant, authType); diags.HasError() {
			return ingestionapi.AuthInputPartial{}, diags
		}

		return *ingestionapi.AuthAlgoliaInsightsPartialAsAuthInputPartial(&variant), nil
	case ingestionapi.AUTHENTICATION_TYPE_API_KEY:
		var variant ingestionapi.AuthAPIKeyPartial
		if diags := decodeAuthInput(raw, &variant, authType); diags.HasError() {
			return ingestionapi.AuthInputPartial{}, diags
		}

		return *ingestionapi.AuthAPIKeyPartialAsAuthInputPartial(&variant), nil
	case ingestionapi.AUTHENTICATION_TYPE_BASIC:
		var variant ingestionapi.AuthBasicPartial
		if diags := decodeAuthInput(raw, &variant, authType); diags.HasError() {
			return ingestionapi.AuthInputPartial{}, diags
		}

		return *ingestionapi.AuthBasicPartialAsAuthInputPartial(&variant), nil
	case ingestionapi.AUTHENTICATION_TYPE_GOOGLE_SERVICE_ACCOUNT:
		var variant ingestionapi.AuthGoogleServiceAccountPartial
		if diags := decodeAuthInput(raw, &variant, authType); diags.HasError() {
			return ingestionapi.AuthInputPartial{}, diags
		}

		return *ingestionapi.AuthGoogleServiceAccountPartialAsAuthInputPartial(&variant), nil
	case ingestionapi.AUTHENTICATION_TYPE_OAUTH:
		var variant ingestionapi.AuthOAuthPartial
		if diags := decodeAuthInput(raw, &variant, authType); diags.HasError() {
			return ingestionapi.AuthInputPartial{}, diags
		}

		return *ingestionapi.AuthOAuthPartialAsAuthInputPartial(&variant), nil
	case ingestionapi.AUTHENTICATION_TYPE_SECRETS:
		var variant map[string]string
		if diags := decodeAuthInput(raw, &variant, authType); diags.HasError() {
			return ingestionapi.AuthInputPartial{}, diags
		}

		return *ingestionapi.MapmapOfStringstringAsAuthInputPartial(variant), nil
	default:
		return ingestionapi.AuthInputPartial{}, unsupportedAuthTypeDiags(authType)
	}
}

// decodeAuthInput strictly decodes the `input` attribute into the credential
// struct for authType. Keys belonging to a different authentication type are
// rejected rather than dropped: dropping them is what let a mismatched `input`
// reach the API as empty credentials.
func decodeAuthInput(raw []byte, target any, authType string) diag.Diagnostics {
	var diags diag.Diagnostics

	if err := decodeJSONStrict(raw, target); err != nil {
		diags.AddError(
			"Invalid input JSON",
			"The `input` attribute must be JSON-encoded credentials matching the authentication type \""+authType+"\" "+
				"(e.g. jsonencode({ appID = \"...\", apiKey = \"...\" }) for type \"algolia\"). Failed to decode: "+err.Error(),
		)
	}

	return diags
}

// unsupportedAuthTypeDiags reports an authentication type the provider cannot
// map to a credential shape. This is reachable when the Algolia client gains a
// new AuthenticationType - the schema's allowed values are derived from that
// enum - before this file learns its credential struct. Erroring is the point:
// falling back to a guessed variant would send empty credentials.
func unsupportedAuthTypeDiags(authType string) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.AddError(
		"Unsupported authentication type",
		"The provider does not know the credential shape for authentication type \""+authType+"\" and will not "+
			"guess it, since guessing would send empty credentials to Algolia. Please report this as a provider bug.",
	)

	return diags
}

// expandPlatform converts the Terraform platform attribute into the
// utils.Nullable[Platform] expected by AuthenticationCreate. An unset/null
// platform maps to an unset Nullable rather than an explicit null, matching
// how the API treats "no platform" for non-ecommerce authentication types.
func expandPlatform(platform types.String) utils.Nullable[ingestionapi.Platform] {
	if platform.IsNull() || platform.IsUnknown() {
		return utils.Nullable[ingestionapi.Platform]{}
	}

	value := ingestionapi.Platform(platform.ValueString())

	return *utils.NewNullable(&value)
}
