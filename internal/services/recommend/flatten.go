package recommend

import (
	"encoding/json"

	recommendapi "github.com/algolia/algoliasearch-client-go/v4/algolia/recommend"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// flattenRecommendRule copies a RecommendRule (from GetRecommendRule, or
// read back after BatchRecommendRules) into the Terraform model.
// condition/consequence/validity are refreshed using the same
// semantic-equality preserve-prior pattern as the Ingestion package's
// JSON-encoded attributes (see json.go): model's existing value is kept
// as-is when it is semantically equal to the API's encoding, and only
// replaced when it actually differs. For the data source, and on import,
// model has no prior configuration for those fields (they start out null),
// so the API's encoding is always adopted.
func flattenRecommendRule(indexName, modelName string, rule *recommendapi.RecommendRule, model *RecommendRuleResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(recommendRuleResourceID(indexName, modelName, rule.GetObjectID()))
	model.IndexName = types.StringValue(indexName)
	model.Model = types.StringValue(modelName)
	model.ObjectID = types.StringValue(rule.GetObjectID())

	conditionValue, conditionDiags := flattenJSONField(rule.Condition, model.Condition, "condition")
	diags.Append(conditionDiags...)
	model.Condition = conditionValue

	consequenceValue, consequenceDiags := flattenJSONField(rule.Consequence, model.Consequence, "consequence")
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

	validityValue, validityDiags := flattenValidity(rule.GetValidity(), model.Validity)
	diags.Append(validityDiags...)
	model.Validity = validityValue

	return diags
}

// flattenJSONField JSON-encodes a nullable API field (*Condition or
// *Consequence) and decides whether to adopt it into state or keep the
// value already configured/in state (previous). label identifies the
// attribute in error messages.
func flattenJSONField[T any](value *T, previous types.String, label string) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if value == nil {
		if previous.IsNull() || previous.IsUnknown() {
			return types.StringNull(), diags
		}
		return previous, diags
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		diags.AddError("Error encoding "+label, "Could not JSON-encode the rule's "+label+": "+err.Error())
		return previous, diags
	}
	apiValue := string(encoded)

	if !previous.IsNull() && !previous.IsUnknown() && jsonSemanticallyEqual(previous.ValueString(), apiValue) {
		return previous, diags
	}

	return types.StringValue(apiValue), diags
}

// flattenValidity is flattenJSONField's counterpart for the `validity`
// attribute, which is a slice rather than a nullable pointer: an empty/nil
// slice is treated the same as "the API returned no value".
func flattenValidity(validity []recommendapi.TimeRange, previous types.String) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(validity) == 0 {
		if previous.IsNull() || previous.IsUnknown() {
			return types.StringNull(), diags
		}
		return previous, diags
	}

	encoded, err := json.Marshal(validity)
	if err != nil {
		diags.AddError("Error encoding validity", "Could not JSON-encode the rule's validity: "+err.Error())
		return previous, diags
	}
	apiValue := string(encoded)

	if !previous.IsNull() && !previous.IsUnknown() && jsonSemanticallyEqual(previous.ValueString(), apiValue) {
		return previous, diags
	}

	return types.StringValue(apiValue), diags
}
