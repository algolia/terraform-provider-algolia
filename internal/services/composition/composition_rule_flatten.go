package composition

import (
	compositionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/composition"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// flattenCompositionRule copies a CompositionRule (from GetRule, or read
// back after PutCompositionRule) into the Terraform model. conditions/
// consequence/validity are refreshed using the semantic-equality
// preserve-prior pattern in json.go: model's existing value is kept as-is
// when it is semantically equal to the API's encoding, and only replaced
// when it actually differs. For the data source, and on import, model has
// no prior configuration for those fields (they start out null), so the
// API's encoding is always adopted.
func flattenCompositionRule(compositionID string, rule *compositionapi.CompositionRule, model *CompositionRuleResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(compositionRuleResourceID(compositionID, rule.GetObjectID()))
	model.CompositionID = types.StringValue(compositionID)
	model.ObjectID = types.StringValue(rule.GetObjectID())

	conditionsValue, conditionsDiags := flattenSliceJSONField(rule.GetConditions(), model.Conditions, "conditions")
	diags.Append(conditionsDiags...)
	model.Conditions = conditionsValue

	consequenceValue, consequenceDiags := flattenRequiredJSONField(rule.Consequence, model.Consequence, "consequence")
	diags.Append(consequenceDiags...)
	model.Consequence = consequenceValue

	if rule.Description != nil {
		model.Description = types.StringValue(rule.GetDescription())
	} else {
		model.Description = types.StringNull()
	}

	if rule.Enabled != nil {
		model.Enabled = types.BoolValue(rule.GetEnabled())
	} else {
		model.Enabled = types.BoolValue(true)
	}

	validityValue, validityDiags := flattenSliceJSONField(rule.GetValidity(), model.Validity, "validity")
	diags.Append(validityDiags...)
	model.Validity = validityValue

	tagsValue, tagsDiags := nullableStringList(model.Tags, rule.GetTags())
	diags.Append(tagsDiags...)
	model.Tags = tagsValue

	return diags
}

// nullableStringList converts a []string into a Terraform list. `tags` is
// Optional and not Computed, so its planned value is the configuration
// verbatim: emitting a known empty list where the plan held null (or null where
// the plan held `[]`) makes Terraform reject the apply with "Provider produced
// inconsistent result after apply". When the API returns nothing, the prior
// value therefore decides: a null prior stays null, while a prior that was
// explicitly configured as `[]` stays a known empty list.
func nullableStringList(prior types.List, values []string) (types.List, diag.Diagnostics) {
	if len(values) == 0 {
		if prior.IsNull() || prior.IsUnknown() {
			return types.ListNull(types.StringType), nil
		}

		return types.ListValue(types.StringType, []attr.Value{})
	}

	attrValues := make([]attr.Value, 0, len(values))
	for _, value := range values {
		attrValues = append(attrValues, types.StringValue(value))
	}

	return types.ListValue(types.StringType, attrValues)
}
