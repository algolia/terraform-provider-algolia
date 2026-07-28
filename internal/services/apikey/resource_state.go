package apikey

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func buildAPIKeyRequest(model *APIKeyResourceModel, now time.Time) (*search.ApiKey, diag.Diagnostics) {
	var diags diag.Diagnostics

	acls, aclDiags := setToSortedStrings(context.Background(), model.ACL)
	diags.Append(aclDiags...)
	if diags.HasError() {
		return nil, diags
	}

	aclValues := make([]search.Acl, 0, len(acls))
	for _, acl := range acls {
		aclValues = append(aclValues, search.Acl(acl))
	}

	apiKey := search.NewApiKey(aclValues)

	if !model.Description.IsNull() && !model.Description.IsUnknown() {
		apiKey.SetDescription(model.Description.ValueString())
	}

	indexes, indexDiags := setToSortedStrings(context.Background(), model.Indexes)
	diags.Append(indexDiags...)
	if diags.HasError() {
		return nil, diags
	}
	if len(indexes) > 0 {
		apiKey.SetIndexes(indexes)
	}

	referers, refererDiags := setToSortedStrings(context.Background(), model.Referers)
	diags.Append(refererDiags...)
	if diags.HasError() {
		return nil, diags
	}
	if len(referers) > 0 {
		apiKey.SetReferers(referers)
	}

	if !model.MaxHitsPerQuery.IsNull() && !model.MaxHitsPerQuery.IsUnknown() && model.MaxHitsPerQuery.ValueInt64() > 0 {
		value := int32(model.MaxHitsPerQuery.ValueInt64())
		apiKey.SetMaxHitsPerQuery(value)
	}

	if !model.MaxQueriesPerIPPerHour.IsNull() && !model.MaxQueriesPerIPPerHour.IsUnknown() && model.MaxQueriesPerIPPerHour.ValueInt64() > 0 {
		value := int32(model.MaxQueriesPerIPPerHour.ValueInt64())
		apiKey.SetMaxQueriesPerIPPerHour(value)
	}

	if !model.ExpiresAt.IsNull() && !model.ExpiresAt.IsUnknown() {
		expiresAt, err := time.Parse(time.RFC3339, model.ExpiresAt.ValueString())
		if err != nil {
			diags.AddError("Invalid expires_at", "Could not parse expires_at as RFC3339: "+err.Error())
			return nil, diags
		}

		validity := int32(expiresAt.Sub(now).Seconds())
		if validity < 0 {
			diags.AddError("Invalid expires_at", "expires_at must be in the future.")
			return nil, diags
		}

		apiKey.SetValidity(validity)
	}

	return apiKey, diags
}

func parseExpiresAt(model *APIKeyResourceModel) (*time.Time, diag.Diagnostics) {
	var diags diag.Diagnostics

	if model == nil || model.ExpiresAt.IsNull() || model.ExpiresAt.IsUnknown() {
		return nil, diags
	}

	expiresAt, err := time.Parse(time.RFC3339, model.ExpiresAt.ValueString())
	if err != nil {
		diags.AddError("Invalid expires_at", "Could not parse expires_at as RFC3339: "+err.Error())
		return nil, diags
	}

	return &expiresAt, diags
}

func apiKeyResponseMatches(response *search.GetApiKeyResponse, expected *search.ApiKey, expiresAt *time.Time, now time.Time) bool {
	if response == nil || expected == nil {
		return false
	}

	if expected.GetDescription() != response.GetDescription() {
		return false
	}

	if expected.GetMaxHitsPerQuery() != response.GetMaxHitsPerQuery() {
		return false
	}

	if expected.GetMaxQueriesPerIPPerHour() != response.GetMaxQueriesPerIPPerHour() {
		return false
	}

	if !slicesEqualUnordered(expected.GetAcl(), response.GetAcl()) {
		return false
	}

	if !slicesEqualUnordered(expected.GetIndexes(), response.GetIndexes()) {
		return false
	}

	if !slicesEqualUnordered(expected.GetReferers(), response.GetReferers()) {
		return false
	}

	actualValidity, hasActualValidity := response.GetValidityOk()
	if expiresAt == nil {
		return !hasActualValidity || actualValidity == nil || *actualValidity == 0
	}

	if !hasActualValidity || actualValidity == nil {
		return false
	}

	expectedValidity := int32(expiresAt.Sub(now).Seconds())
	return int32(math.Abs(float64(*actualValidity-expectedValidity))) <= 5
}

func slicesEqualUnordered[T ~string](left, right []T) bool {
	if len(left) != len(right) {
		return false
	}

	leftValues := append([]T(nil), left...)
	rightValues := append([]T(nil), right...)

	sort.Slice(leftValues, func(i, j int) bool { return leftValues[i] < leftValues[j] })
	sort.Slice(rightValues, func(i, j int) bool { return rightValues[i] < rightValues[j] })

	for i := range leftValues {
		if leftValues[i] != rightValues[i] {
			return false
		}
	}

	return true
}

func hydrateAPIKeyModel(resp *search.GetApiKeyResponse, preserved *APIKeyResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	preservedExpiry := types.StringNull()
	if preserved != nil {
		preservedExpiry = preserved.ExpiresAt
	}

	aclValues := make([]attr.Value, 0, len(resp.GetAcl()))
	for _, acl := range resp.GetAcl() {
		aclValues = append(aclValues, types.StringValue(string(acl)))
	}

	preserved.ID = types.StringValue(resp.GetValue())
	preserved.ACL = types.SetValueMust(types.StringType, aclValues)
	preserved.Description = nullableString(resp.GetDescriptionOk())
	preserved.ExpiresAt = preservedExpiry
	preserved.Indexes = nullableStringSet(preserved.Indexes, resp.GetIndexes())
	preserved.Referers = nullableStringSet(preserved.Referers, resp.GetReferers())
	preserved.MaxHitsPerQuery = nullableInt32(resp.GetMaxHitsPerQueryOk())
	preserved.MaxQueriesPerIPPerHour = nullableInt32(resp.GetMaxQueriesPerIPPerHourOk())
	preserved.CreatedAt = types.StringValue(time.UnixMilli(resp.GetCreatedAt()).UTC().Format(time.RFC3339))

	return diags
}

func setToSortedStrings(ctx context.Context, value types.Set) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if value.IsNull() || value.IsUnknown() {
		return nil, diags
	}

	var values []string
	diags.Append(value.ElementsAs(ctx, &values, false)...)
	if diags.HasError() {
		return nil, diags
	}

	sort.Strings(values)
	return values, diags
}

func nullableString(value *string, ok bool) types.String {
	if !ok || value == nil || *value == "" {
		return types.StringNull()
	}

	return types.StringValue(*value)
}

// nullableStringSet converts an API string slice into a sorted Terraform set.
// `indexes` and `referers` are Optional and not Computed, so their planned
// value is the configuration verbatim: emitting a known empty set where the
// plan held null makes Terraform reject the apply with "Provider produced
// inconsistent result after apply". When the API returns nothing, the prior
// value therefore decides: a null prior stays null, while a prior that was
// explicitly configured as `[]` stays a known empty set.
func nullableStringSet(prior types.Set, values []string) types.Set {
	if len(values) == 0 {
		if prior.IsNull() || prior.IsUnknown() {
			return types.SetNull(types.StringType)
		}

		return types.SetValueMust(types.StringType, []attr.Value{})
	}

	sorted := append([]string(nil), values...)
	sort.Strings(sorted)

	attrValues := make([]attr.Value, 0, len(sorted))
	for _, value := range sorted {
		attrValues = append(attrValues, types.StringValue(value))
	}

	return types.SetValueMust(types.StringType, attrValues)
}

func nullableInt32(value *int32, ok bool) types.Int64 {
	if !ok || value == nil {
		return types.Int64Null()
	}

	return types.Int64Value(int64(*value))
}
