package composition

import (
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// jsonSemanticallyEqual returns true if two strings are both valid JSON and
// represent the same data structure, ignoring object key order and whitespace.
// Array order is significant: sorting arrays would hide changes in API read-back.
func jsonSemanticallyEqual(a, b string) bool {
	var va, vb any
	if err := json.Unmarshal([]byte(a), &va); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &vb); err != nil {
		return false
	}

	return reflect.DeepEqual(va, vb)
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
