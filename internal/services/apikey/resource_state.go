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

	// Unlike the fields above, an empty string is sent rather than skipped: it is
	// a valid way to ask for the restriction to be cleared, and the API resets
	// the field either way.
	if !model.QueryParameters.IsNull() && !model.QueryParameters.IsUnknown() {
		apiKey.SetQueryParameters(model.QueryParameters.ValueString())
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

	if expected.GetQueryParameters() != response.GetQueryParameters() {
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
	return math.Abs(float64(*actualValidity-expectedValidity)) <= expiresAtToleranceSeconds
}

// expiresAtToleranceSeconds is how far a key's reported validity may sit from the
// configured expires_at before the difference counts as a real change rather than
// the clock and network latency between computing a validity and the API
// recording it. Shared by the apply-consistency check above and by
// flattenExpiresAt, which must agree with it: if one treated a gap as drift while
// the other treated it as equal, a key could refresh into a state the same code
// then judged inconsistent.
const expiresAtToleranceSeconds = 5

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
	validity, hasValidity := resp.GetValidityOk()
	preserved.ExpiresAt = flattenExpiresAt(preservedExpiry, validity, hasValidity)
	preserved.Indexes = nullableStringSet(preserved.Indexes, resp.GetIndexes())
	preserved.Referers = nullableStringSet(preserved.Referers, resp.GetReferers())
	preserved.MaxHitsPerQuery = nullableInt32(resp.GetMaxHitsPerQueryOk())
	preserved.MaxQueriesPerIPPerHour = nullableInt32(resp.GetMaxQueriesPerIPPerHourOk())
	preserved.QueryParameters = nullableStringPreservingEmpty(preserved.QueryParameters, nullableString(resp.GetQueryParametersOk()))
	preserved.CreatedAt = types.StringValue(createdAtTimestamp(resp.GetCreatedAt()))

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

// nullableStringPreservingEmpty restores an explicitly configured empty string
// that nullableString collapsed to null. It is the string counterpart of
// nullableStringSet and exists for exactly the same reason: `query_parameters`
// is Optional and not Computed, so its planned value is the configuration
// verbatim, and emitting null where the plan held a known "" makes Terraform
// reject the apply with "Provider produced inconsistent result after apply".
// The API drops the field entirely once it is cleared, so when it comes back
// absent the prior value decides: a known "" stays "", anything else stays null.
func nullableStringPreservingEmpty(prior, value types.String) types.String {
	if value.IsNull() && !prior.IsNull() && !prior.IsUnknown() && prior.ValueString() == "" {
		return types.StringValue("")
	}

	return value
}

func nullableInt32(value *int32, ok bool) types.Int64 {
	if !ok || value == nil {
		return types.Int64Null()
	}

	return types.Int64Value(int64(*value))
}

// createdAtTimestamp renders an API key's creation time as RFC3339.
//
// The value is in *seconds* since the Unix epoch, even though the generated
// client documents GetApiKeyResponse.CreatedAt as milliseconds: the API returned
// 1785269593 for a key created at 2026-07-28T20:13:13Z. Reading it as
// milliseconds put every key in January 1970.
func createdAtTimestamp(seconds int64) string {
	return time.Unix(seconds, 0).UTC().Format(time.RFC3339)
}

// flattenExpiresAt resolves expires_at from the key's validity.
//
// The API reports `validity` as the number of seconds the key still has to live,
// not the value originally requested, so the stable quantity is now+validity:
// the absolute instant the key expires, which does not move between reads except
// by clock and latency jitter. That instant is what expires_at holds.
//
// A configured timestamp is kept verbatim whenever it denotes the same instant
// within expiresAtToleranceSeconds, because expires_at is Optional and not
// Computed: rewriting it to a re-derived string differing by a second would make
// Terraform reject the apply as an inconsistent result, and would otherwise
// produce a diff on every refresh. Beyond the tolerance the recomputed value
// wins, so shortening or extending a key's life out of band surfaces as drift.
//
// Deriving rather than preserving matters most on import, where there is no
// prior value: leaving it null recorded no expiry at all, and because
// expandAPIKey only sends `validity` when expires_at is known, the next apply
// then reset the key to never expire with nothing in state or plan to show it.
func flattenExpiresAt(prior types.String, validity *int32, hasValidity bool) types.String {
	if !hasValidity || validity == nil || *validity == 0 {
		return types.StringNull()
	}

	expiry := timeNow().Add(time.Duration(*validity) * time.Second)

	if !prior.IsNull() && !prior.IsUnknown() {
		if parsed, err := time.Parse(time.RFC3339, prior.ValueString()); err == nil {
			drift := parsed.Sub(expiry).Seconds()
			if math.Abs(drift) <= expiresAtToleranceSeconds {
				return prior
			}
		}
	}

	return types.StringValue(expiry.UTC().Format(time.RFC3339))
}

// timeNow is indirected so tests can pin the clock that flattenExpiresAt reads.
var timeNow = time.Now
