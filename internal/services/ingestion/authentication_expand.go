package ingestion

import (
	"encoding/json"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/utils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// expandAuthenticationCreate converts the Terraform plan into an
// AuthenticationCreate request body for CreateAuthentication.
func expandAuthenticationCreate(model *AuthenticationResourceModel) (*ingestionapi.AuthenticationCreate, diag.Diagnostics) {
	var diags diag.Diagnostics

	input, inputDiags := expandAuthInput(model.Input)
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

	inputPartial, inputDiags := expandAuthInputPartial(model.Input)
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

// expandAuthInput JSON-decodes the `input` attribute into the AuthInput
// union type expected by AuthenticationCreate. The Algolia Go client's
// generated UnmarshalJSON inspects the object's keys to pick the right
// variant (AuthAlgolia, AuthAPIKey, AuthBasic, AuthOAuth,
// AuthGoogleServiceAccount, AuthAlgoliaInsights, or a raw
// map[string]string for "secrets").
func expandAuthInput(input types.String) (ingestionapi.AuthInput, diag.Diagnostics) {
	var diags diag.Diagnostics
	var authInput ingestionapi.AuthInput

	if err := json.Unmarshal([]byte(input.ValueString()), &authInput); err != nil {
		diags.AddError(
			"Invalid input JSON",
			"The `input` attribute must be JSON-encoded credentials matching the authentication `type` "+
				"(e.g. jsonencode({ appID = \"...\", apiKey = \"...\" }) for type \"algolia\"). Failed to parse: "+err.Error(),
		)
	}

	return authInput, diags
}

// expandAuthInputPartial is the AuthenticationUpdate counterpart of
// expandAuthInput: the update endpoint accepts AuthInputPartial rather than
// AuthInput.
func expandAuthInputPartial(input types.String) (ingestionapi.AuthInputPartial, diag.Diagnostics) {
	var diags diag.Diagnostics
	var authInput ingestionapi.AuthInputPartial

	if err := json.Unmarshal([]byte(input.ValueString()), &authInput); err != nil {
		diags.AddError(
			"Invalid input JSON",
			"The `input` attribute must be JSON-encoded credentials matching the authentication `type`. "+
				"Failed to parse: "+err.Error(),
		)
	}

	return authInput, diags
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
