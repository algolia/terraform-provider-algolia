package ingestion

import (
	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/utils"
	"github.com/algolia/terraform-provider-algolia/internal/deletionprotection"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// flattenAuthentication copies the non-secret fields of a GetAuthentication
// response into the Terraform resource model.
//
// It deliberately never touches model.Input: GetAuthentication redacts
// secret values (returning AuthInputPartial, not the AuthInput that was
// sent), so overwriting Input here would permanently diff against the real
// credentials configured by the user. Callers (Create/Read/Update) must
// leave the model's existing Input value as-is.
func flattenAuthentication(auth *ingestionapi.Authentication, model *AuthenticationResourceModel) diag.Diagnostics {
	// Algolia does not store this flag, so it survives only by being carried through
	// every rebuild of the model. Resolving it here also seeds an import, which
	// arrives with no value at all.
	model.DeletionProtection = deletionprotection.Value(model.DeletionProtection)

	var diags diag.Diagnostics

	model.ID = types.StringValue(auth.AuthenticationID)
	model.AuthenticationID = types.StringValue(auth.AuthenticationID)
	model.Type = types.StringValue(string(auth.Type))
	model.Name = types.StringValue(auth.Name)
	model.Platform = flattenPlatform(auth.Platform)
	model.CreatedAt = types.StringValue(auth.CreatedAt)
	model.UpdatedAt = types.StringValue(auth.UpdatedAt)

	return diags
}

// flattenAuthenticationDataSource is the data source counterpart of
// flattenAuthentication. AuthenticationDataSourceModel has no Input field at
// all, so there is nothing to preserve or omit here.
func flattenAuthenticationDataSource(auth *ingestionapi.Authentication, model *AuthenticationDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(auth.AuthenticationID)
	model.AuthenticationID = types.StringValue(auth.AuthenticationID)
	model.Type = types.StringValue(string(auth.Type))
	model.Name = types.StringValue(auth.Name)
	model.Platform = flattenPlatform(auth.Platform)
	model.CreatedAt = types.StringValue(auth.CreatedAt)
	model.UpdatedAt = types.StringValue(auth.UpdatedAt)

	return diags
}

// flattenPlatform converts the API's utils.Nullable[Platform] into a
// Terraform types.String, mapping both "unset" and "explicit null" to a
// null Terraform value.
func flattenPlatform(platform utils.Nullable[ingestionapi.Platform]) types.String {
	if !platform.IsSet() {
		return types.StringNull()
	}

	value := platform.Get()
	if value == nil {
		return types.StringNull()
	}

	return types.StringValue(string(*value))
}
