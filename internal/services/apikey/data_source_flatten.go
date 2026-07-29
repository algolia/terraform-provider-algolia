package apikey

import (
	"context"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// flattenAPIKeyDataSource converts a GetApiKeyResponse into the
// algolia_api_key data source model.
func flattenAPIKeyDataSource(ctx context.Context, resp *search.GetApiKeyResponse, model *APIKeyDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	aclValues, d := types.ListValueFrom(ctx, types.StringType, aclStrings(resp.GetAcl()))
	diags.Append(d...)
	indexesValues, d := types.ListValueFrom(ctx, types.StringType, resp.GetIndexes())
	diags.Append(d...)
	referersValues, d := types.ListValueFrom(ctx, types.StringType, resp.GetReferers())
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	model.ID = types.StringValue(resp.GetValue())
	model.Key = types.StringValue(resp.GetValue())
	model.ACL = aclValues
	model.Description = nullableString(resp.GetDescriptionOk())
	model.Indexes = indexesValues
	model.MaxHitsPerQuery = nullableInt32(resp.GetMaxHitsPerQueryOk())
	model.MaxQueriesPerIPPerHour = nullableInt32(resp.GetMaxQueriesPerIPPerHourOk())
	model.QueryParameters = nullableString(resp.GetQueryParametersOk())
	model.Referers = referersValues
	model.Validity = nullableInt32(resp.GetValidityOk())
	model.CreatedAt = types.StringValue(createdAtTimestamp(resp.GetCreatedAt()))

	return diags
}

// flattenAPIKeysDataSource converts a ListApiKeysResponse into the
// algolia_api_keys data source model.
func flattenAPIKeysDataSource(ctx context.Context, resp *search.ListApiKeysResponse, appID string, model *APIKeysDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	keys := resp.GetKeys()
	items := make([]APIKeyListItemModel, 0, len(keys))
	for i := range keys {
		key := keys[i]

		aclValues, d := types.ListValueFrom(ctx, types.StringType, aclStrings(key.GetAcl()))
		diags.Append(d...)
		indexesValues, d := types.ListValueFrom(ctx, types.StringType, key.GetIndexes())
		diags.Append(d...)
		referersValues, d := types.ListValueFrom(ctx, types.StringType, key.GetReferers())
		diags.Append(d...)

		items = append(items, APIKeyListItemModel{
			Value:                  types.StringValue(key.GetValue()),
			ACL:                    aclValues,
			Description:            nullableString(key.GetDescriptionOk()),
			Indexes:                indexesValues,
			MaxHitsPerQuery:        nullableInt32(key.GetMaxHitsPerQueryOk()),
			MaxQueriesPerIPPerHour: nullableInt32(key.GetMaxQueriesPerIPPerHourOk()),
			QueryParameters:        nullableString(key.GetQueryParametersOk()),
			Referers:               referersValues,
			Validity:               nullableInt32(key.GetValidityOk()),
			CreatedAt:              types.StringValue(createdAtTimestamp(key.GetCreatedAt())),
		})
	}
	if diags.HasError() {
		return diags
	}

	keysValue, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: apiKeyListItemAttrTypes}, items)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	model.ID = types.StringValue(appID)
	model.Keys = keysValue

	return diags
}

// aclStrings converts a slice of the Acl enum type to plain strings.
func aclStrings(values []search.Acl) []string {
	result := make([]string, 0, len(values))
	for _, v := range values {
		result = append(result, string(v))
	}

	return result
}
