package composition

import (
	"encoding/json"
	"reflect"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// jsonSemanticallyEqual returns true if two strings are both valid JSON and
// represent the same data structure, ignoring object key order, whitespace,
// and the order of arrays whose elements are all the same scalar kind (all
// strings or all numbers) - GetComposition/GetRule may echo behavior/
// sorting_strategy/conditions/consequence/validity back in a different
// encoding than what was submitted. Arrays containing objects, or with mixed
// element kinds, keep their order, since element order there is usually
// significant. Used by flatten helpers below to decide whether a
// JSON-encoded attribute actually changed, or the API just returned an
// equivalent encoding of the same configuration.
//
// This mirrors the identically named helper in internal/services/index,
// internal/services/ingestion, and internal/services/recommend - replicated
// here rather than imported, per those packages' own rationale: these are
// small, dependency-free functions, and importing across service packages
// isn't worth it for that.
func jsonSemanticallyEqual(a, b string) bool {
	var va, vb any
	if err := json.Unmarshal([]byte(a), &va); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &vb); err != nil {
		return false
	}

	return reflect.DeepEqual(normalizeJSON(va), normalizeJSON(vb))
}

// normalizeJSON recursively normalizes a decoded JSON value so that
// comparison is order-independent where order is insignificant: object keys
// (already order-independent as Go maps) are recursed into, and arrays whose
// elements are all strings or all numbers are sorted. Arrays containing
// objects or a mix of kinds are left in their original order.
func normalizeJSON(v any) any {
	switch val := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(val))
		for k, v := range val {
			m[k] = normalizeJSON(v)
		}
		return m
	case []any:
		normalized := make([]any, len(val))
		for i, item := range val {
			normalized[i] = normalizeJSON(item)
		}
		sortScalarSlice(normalized)
		return normalized
	default:
		return v
	}
}

// sortScalarSlice sorts s in place only when every element is a string, or
// every element is a float64 (JSON numbers decode to float64). Slices that
// contain objects or mix element kinds are left untouched, since reordering
// them could change meaning.
func sortScalarSlice(s []any) {
	if len(s) < 2 {
		return
	}

	allStrings, allNumbers := true, true
	for _, item := range s {
		if _, ok := item.(string); !ok {
			allStrings = false
		}
		if _, ok := item.(float64); !ok {
			allNumbers = false
		}
	}

	switch {
	case allStrings:
		sort.Slice(s, func(i, j int) bool { return s[i].(string) < s[j].(string) })
	case allNumbers:
		sort.Slice(s, func(i, j int) bool { return s[i].(float64) < s[j].(float64) })
	}
}

// flattenRequiredJSONField JSON-encodes an always-present (non-nullable) API
// field - such as Composition.Behavior or CompositionRule.Consequence - and
// decides whether to adopt it into state or keep the value already
// configured/in state (previous), using jsonSemanticallyEqual so a harmless
// re-encoding by the API does not cause a perpetual diff. label identifies
// the attribute in error messages.
func flattenRequiredJSONField(value any, previous types.String, label string) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	encoded, err := json.Marshal(value)
	if err != nil {
		diags.AddError("Error encoding "+label, "Could not JSON-encode the "+label+": "+err.Error())
		return previous, diags
	}
	apiValue := string(encoded)

	if !previous.IsNull() && !previous.IsUnknown() && jsonSemanticallyEqual(previous.ValueString(), apiValue) {
		return previous, diags
	}

	return types.StringValue(apiValue), diags
}

// flattenOptionalJSONField is flattenRequiredJSONField's counterpart for a
// nullable (pointer) API field, such as Composition.SortingStrategy: a nil
// value is treated the same as "the API returned no value", in which case
// the previously configured value (if any) is preserved rather than cleared.
func flattenOptionalJSONField[T any](value *T, previous types.String, label string) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if value == nil {
		if previous.IsNull() || previous.IsUnknown() {
			return types.StringNull(), diags
		}
		return previous, diags
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		diags.AddError("Error encoding "+label, "Could not JSON-encode the "+label+": "+err.Error())
		return previous, diags
	}
	apiValue := string(encoded)

	if !previous.IsNull() && !previous.IsUnknown() && jsonSemanticallyEqual(previous.ValueString(), apiValue) {
		return previous, diags
	}

	return types.StringValue(apiValue), diags
}

// flattenSliceJSONField is flattenRequiredJSONField's counterpart for a
// slice-typed API field, such as CompositionRule.Conditions or
// CompositionRule.Validity: an empty/nil slice is treated the same as "the
// API returned no value".
func flattenSliceJSONField[T any](value []T, previous types.String, label string) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(value) == 0 {
		if previous.IsNull() || previous.IsUnknown() {
			return types.StringNull(), diags
		}
		return previous, diags
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		diags.AddError("Error encoding "+label, "Could not JSON-encode the "+label+": "+err.Error())
		return previous, diags
	}
	apiValue := string(encoded)

	if !previous.IsNull() && !previous.IsUnknown() && jsonSemanticallyEqual(previous.ValueString(), apiValue) {
		return previous, diags
	}

	return types.StringValue(apiValue), diags
}
