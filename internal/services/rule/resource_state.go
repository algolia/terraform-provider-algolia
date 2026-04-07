package rule

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func buildRuleRequest(model *RuleResourceModel) (*search.Rule, diag.Diagnostics) {
	var diags diag.Diagnostics

	consequence, consequenceDiags := expandConsequence(model.Consequence)
	diags.Append(consequenceDiags...)
	if diags.HasError() {
		return nil, diags
	}

	rule := search.NewRule(model.ObjectID.ValueString(), *consequence)

	if !model.Description.IsNull() && !model.Description.IsUnknown() {
		description := model.Description.ValueString()
		rule.Description = &description
	}
	if !model.Enabled.IsNull() && !model.Enabled.IsUnknown() {
		enabled := model.Enabled.ValueBool()
		rule.Enabled = &enabled
	}

	conditions, conditionDiags := expandConditions(model.Conditions)
	diags.Append(conditionDiags...)
	if diags.HasError() {
		return nil, diags
	}
	if len(conditions) > 0 {
		rule.Conditions = conditions
	}

	validity, validityDiags := expandValidity(model.Validity)
	diags.Append(validityDiags...)
	if diags.HasError() {
		return nil, diags
	}
	if len(validity) > 0 {
		rule.Validity = validity
	}

	return rule, diags
}

func hydrateRuleModel(indexName string, ruleResp *search.Rule, model *RuleResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(ruleResourceID(indexName, ruleResp.GetObjectID()))
	model.IndexName = types.StringValue(indexName)
	model.ObjectID = types.StringValue(ruleResp.GetObjectID())

	if ruleResp.Description != nil {
		model.Description = types.StringValue(ruleResp.GetDescription())
	} else {
		model.Description = types.StringNull()
	}

	if ruleResp.Enabled != nil {
		model.Enabled = types.BoolValue(ruleResp.GetEnabled())
	} else {
		model.Enabled = types.BoolValue(true)
	}

	conditions, conditionDiags := flattenConditions(ruleResp.GetConditions())
	diags.Append(conditionDiags...)
	if diags.HasError() {
		return diags
	}
	model.Conditions = conditions

	consequence, consequenceDiags := flattenConsequence(ruleResp.GetConsequence())
	diags.Append(consequenceDiags...)
	if diags.HasError() {
		return diags
	}
	model.Consequence = consequence

	validity, validityDiags := flattenValidity(ruleResp.GetValidity())
	diags.Append(validityDiags...)
	if diags.HasError() {
		return diags
	}
	model.Validity = validity

	return diags
}

func parseRuleImportID(id string) (string, string, error) {
	index := strings.Index(id, "/")
	if index <= 0 || index == len(id)-1 {
		return "", "", fmt.Errorf("expected import ID in the form <index_name>/<object_id>")
	}

	return id[:index], id[index+1:], nil
}

func ruleResourceID(indexName, objectID string) string {
	return indexName + "/" + objectID
}

func expandConditions(list types.List) ([]search.Condition, diag.Diagnostics) {
	var diags diag.Diagnostics

	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}

	conditions := make([]search.Condition, 0, len(list.Elements()))
	for i, value := range list.Elements() {
		objValue, ok := value.(types.Object)
		if !ok {
			diags.AddError("Invalid condition value", fmt.Sprintf("Condition %d is not an object.", i))
			return nil, diags
		}

		attrs := objValue.Attributes()
		condition := search.NewEmptyCondition()

		if v, ok := attrs["pattern"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
			pattern := v.ValueString()
			condition.Pattern = &pattern
		}
		if v, ok := attrs["anchoring"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
			anchoring := search.Anchoring(v.ValueString())
			condition.Anchoring = &anchoring
		}
		if v, ok := attrs["alternatives"].(types.Bool); ok && !v.IsNull() && !v.IsUnknown() {
			alternatives := v.ValueBool()
			condition.Alternatives = &alternatives
		}
		if v, ok := attrs["context"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
			context := v.ValueString()
			condition.Context = &context
		}
		if v, ok := attrs["filters"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
			filters := v.ValueString()
			condition.Filters = &filters
		}

		conditions = append(conditions, *condition)
	}

	return conditions, diags
}

func flattenConditions(conditions []search.Condition) (types.List, diag.Diagnostics) {
	values := make([]attr.Value, 0, len(conditions))
	for _, condition := range conditions {
		value, diags := types.ObjectValue(conditionModelAttrTypes, map[string]attr.Value{
			"pattern":      nullableString(condition.Pattern),
			"anchoring":    nullableEnum(condition.Anchoring),
			"alternatives": nullableBool(condition.Alternatives),
			"context":      nullableString(condition.Context),
			"filters":      nullableString(condition.Filters),
		})
		if diags.HasError() {
			return types.ListNull(conditionModelType), diags
		}
		values = append(values, value)
	}

	return types.ListValue(conditionModelType, values)
}

func expandConsequence(list types.List) (*search.Consequence, diag.Diagnostics) {
	var diags diag.Diagnostics

	if list.IsNull() || list.IsUnknown() || len(list.Elements()) == 0 {
		diags.AddError("Missing consequence", "A consequence block is required.")
		return nil, diags
	}

	objValue, ok := list.Elements()[0].(types.Object)
	if !ok {
		diags.AddError("Invalid consequence value", "Consequence must be an object.")
		return nil, diags
	}

	attrs := objValue.Attributes()
	consequence := search.NewConsequence()

	if v, ok := attrs["params_json"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
		var params search.ConsequenceParams
		if err := json.Unmarshal([]byte(v.ValueString()), &params); err != nil {
			diags.AddError("Invalid params_json", "Could not decode consequence params_json: "+err.Error())
			return nil, diags
		}
		consequence.Params = &params
	}

	if v, ok := attrs["promote"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
		promote := make([]search.Promote, 0, len(v.Elements()))
		for i, entry := range v.Elements() {
			promoteObj, ok := entry.(types.Object)
			if !ok {
				diags.AddError("Invalid promote value", fmt.Sprintf("Promote entry %d is not an object.", i))
				return nil, diags
			}
			promoteAttrs := promoteObj.Attributes()
			objectIDsValue, ok := promoteAttrs["object_ids"].(types.Set)
			if !ok || objectIDsValue.IsNull() || objectIDsValue.IsUnknown() {
				diags.AddError("Missing object_ids", "Each promote block requires object_ids.")
				return nil, diags
			}
			positionValue, ok := promoteAttrs["position"].(types.Int64)
			if !ok || positionValue.IsNull() || positionValue.IsUnknown() {
				diags.AddError("Missing position", "Each promote block requires position.")
				return nil, diags
			}

			objectIDs := setStrings(objectIDsValue)
			sort.Strings(objectIDs)
			promote = append(promote, *search.PromoteObjectIDsAsPromote(search.NewPromoteObjectIDs(objectIDs, int32(positionValue.ValueInt64()))))
		}
		consequence.Promote = promote
	}

	if v, ok := attrs["hide"].(types.Set); ok && !v.IsNull() && !v.IsUnknown() {
		objectIDs := setStrings(v)
		sort.Strings(objectIDs)
		hide := make([]search.ConsequenceHide, 0, len(objectIDs))
		for _, objectID := range objectIDs {
			hide = append(hide, *search.NewConsequenceHide(objectID))
		}
		consequence.Hide = hide
	}

	if v, ok := attrs["user_data"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
		var userData map[string]any
		if err := json.Unmarshal([]byte(v.ValueString()), &userData); err != nil {
			diags.AddError("Invalid user_data", "Could not decode consequence user_data: "+err.Error())
			return nil, diags
		}
		consequence.UserData = userData
	}

	return consequence, diags
}

func flattenConsequence(consequence search.Consequence) (types.List, diag.Diagnostics) {
	var paramsValue attr.Value = types.StringNull()
	if consequence.Params != nil {
		raw, err := json.Marshal(consequence.GetParams())
		if err != nil {
			var diags diag.Diagnostics
			diags.AddError("Error encoding consequence params", err.Error())
			return types.ListNull(consequenceModelType), diags
		}
		paramsValue = types.StringValue(string(raw))
	}

	promoteValues := make([]attr.Value, 0, len(consequence.GetPromote()))
	for _, promote := range consequence.GetPromote() {
		actual := promote.GetActualInstance()
		switch p := actual.(type) {
		case search.PromoteObjectIDs:
			objectIDs := stringSliceValues(p.GetObjectIDs())
			value, diags := types.ObjectValue(promoteModelAttrTypes, map[string]attr.Value{
				"object_ids": types.SetValueMust(types.StringType, objectIDs),
				"position":   types.Int64Value(int64(p.GetPosition())),
			})
			if diags.HasError() {
				return types.ListNull(consequenceModelType), diags
			}
			promoteValues = append(promoteValues, value)
		case search.PromoteObjectID:
			value, diags := types.ObjectValue(promoteModelAttrTypes, map[string]attr.Value{
				"object_ids": types.SetValueMust(types.StringType, []attr.Value{types.StringValue(p.GetObjectID())}),
				"position":   types.Int64Value(int64(p.GetPosition())),
			})
			if diags.HasError() {
				return types.ListNull(consequenceModelType), diags
			}
			promoteValues = append(promoteValues, value)
		}
	}
	promoteList, promoteDiags := types.ListValue(promoteModelType, promoteValues)
	if promoteDiags.HasError() {
		return types.ListNull(consequenceModelType), promoteDiags
	}

	hideValues := make([]attr.Value, 0, len(consequence.GetHide()))
	for _, hidden := range consequence.GetHide() {
		hideValues = append(hideValues, types.StringValue(hidden.GetObjectID()))
	}
	hideSet, hideDiags := types.SetValue(types.StringType, hideValues)
	if hideDiags.HasError() {
		return types.ListNull(consequenceModelType), hideDiags
	}

	userDataValue := types.StringNull()
	if consequence.UserData != nil {
		raw, err := json.Marshal(consequence.UserData)
		if err != nil {
			var diags diag.Diagnostics
			diags.AddError("Error encoding consequence user_data", err.Error())
			return types.ListNull(consequenceModelType), diags
		}
		userDataValue = types.StringValue(string(raw))
	}

	entry, diags := types.ObjectValue(consequenceModelAttrTypes, map[string]attr.Value{
		"params_json": paramsValue,
		"promote":     promoteList,
		"hide":        hideSet,
		"user_data":   userDataValue,
	})
	if diags.HasError() {
		return types.ListNull(consequenceModelType), diags
	}

	return types.ListValue(consequenceModelType, []attr.Value{entry})
}

func expandValidity(list types.List) ([]search.TimeRange, diag.Diagnostics) {
	var diags diag.Diagnostics

	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}

	validity := make([]search.TimeRange, 0, len(list.Elements()))
	for i, value := range list.Elements() {
		objValue, ok := value.(types.Object)
		if !ok {
			diags.AddError("Invalid validity value", fmt.Sprintf("Validity entry %d is not an object.", i))
			return nil, diags
		}
		attrs := objValue.Attributes()
		rng := search.NewEmptyTimeRange()

		if v, ok := attrs["from"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
			parsed, err := time.Parse(time.RFC3339, v.ValueString())
			if err != nil {
				diags.AddError("Invalid validity.from", "Could not parse validity.from: "+err.Error())
				return nil, diags
			}
			value := parsed.UTC().Unix()
			rng.From = &value
		}
		if v, ok := attrs["until"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
			parsed, err := time.Parse(time.RFC3339, v.ValueString())
			if err != nil {
				diags.AddError("Invalid validity.until", "Could not parse validity.until: "+err.Error())
				return nil, diags
			}
			value := parsed.UTC().Unix()
			rng.Until = &value
		}

		validity = append(validity, *rng)
	}

	return validity, diags
}

func flattenValidity(validity []search.TimeRange) (types.List, diag.Diagnostics) {
	values := make([]attr.Value, 0, len(validity))
	for _, rng := range validity {
		value, diags := types.ObjectValue(validityModelAttrTypes, map[string]attr.Value{
			"from":  nullableUnixTimestamp(rng.From),
			"until": nullableUnixTimestamp(rng.Until),
		})
		if diags.HasError() {
			return types.ListNull(validityModelType), diags
		}
		values = append(values, value)
	}

	return types.ListValue(validityModelType, values)
}

func nullableString(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func nullableEnum[T ~string](value *T) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(string(*value))
}

func nullableBool(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}

func nullableUnixTimestamp(value *int64) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(time.Unix(*value, 0).UTC().Format(time.RFC3339))
}

func setStrings(value types.Set) []string {
	stringsValue := make([]string, 0, len(value.Elements()))
	for _, element := range value.Elements() {
		if stringValue, ok := element.(types.String); ok && !stringValue.IsNull() && !stringValue.IsUnknown() {
			stringsValue = append(stringsValue, stringValue.ValueString())
		}
	}
	return stringsValue
}

func stringSliceValues(values []string) []attr.Value {
	result := make([]attr.Value, 0, len(values))
	for _, value := range values {
		result = append(result, types.StringValue(value))
	}
	return result
}
