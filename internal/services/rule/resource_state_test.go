package rule

import (
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildRuleRequest(t *testing.T) {
	model := RuleResourceModel{
		IndexName: types.StringValue("products"),
		ObjectID:  types.StringValue("rule-1"),
		Conditions: types.ListValueMust(conditionModelType, []attr.Value{
			types.ObjectValueMust(conditionModelAttrTypes, map[string]attr.Value{
				"pattern":      types.StringValue("{facet:brand}"),
				"anchoring":    types.StringValue("contains"),
				"alternatives": types.BoolValue(true),
				"context":      types.StringValue("mobile"),
				"filters":      types.StringNull(),
			}),
		}),
		Consequence: types.ListValueMust(consequenceModelType, []attr.Value{
			types.ObjectValueMust(consequenceModelAttrTypes, map[string]attr.Value{
				"params_json": types.StringValue(`{"query":"iphone"}`),
				"promote": types.ListValueMust(promoteModelType, []attr.Value{
					types.ObjectValueMust(promoteModelAttrTypes, map[string]attr.Value{
						"object_ids": types.SetValueMust(types.StringType, []attr.Value{types.StringValue("1"), types.StringValue("2")}),
						"position":   types.Int64Value(0),
					}),
				}),
				"hide":      types.SetValueMust(types.StringType, []attr.Value{types.StringValue("3")}),
				"user_data": types.StringValue(`{"banner":"promo"}`),
			}),
		}),
		Description: types.StringValue("test rule"),
		Enabled:     types.BoolValue(true),
		Validity: types.ListValueMust(validityModelType, []attr.Value{
			types.ObjectValueMust(validityModelAttrTypes, map[string]attr.Value{
				"from":  types.StringValue("2030-01-01T00:00:00Z"),
				"until": types.StringValue("2030-01-02T00:00:00Z"),
			}),
		}),
	}

	rule, diags := buildRuleRequest(&model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := rule.GetObjectID(); got != "rule-1" {
		t.Fatalf("objectID = %q, want %q", got, "rule-1")
	}
	if got := rule.GetDescription(); got != "test rule" {
		t.Fatalf("description = %q, want %q", got, "test rule")
	}
	consequence := rule.GetConsequence()
	params := consequence.GetParams()
	query := params.GetQuery()
	if got := query.GetActualInstance().(string); got != "iphone" {
		t.Fatalf("query = %q, want %q", got, "iphone")
	}
	if got := consequence.GetPromote(); len(got) != 1 {
		t.Fatalf("promote = %#v, want 1 item", got)
	}
	if got := consequence.GetHide(); len(got) != 1 || got[0].GetObjectID() != "3" {
		t.Fatalf("hide = %#v, want [3]", got)
	}
}

func TestHydrateRuleModel(t *testing.T) {
	ruleResp := search.NewRule(
		"rule-1",
		*search.NewConsequence(
			search.WithConsequenceParams(*search.NewConsequenceParams(search.WithConsequenceParamsQuery(*search.StringAsConsequenceQuery(`iphone`)))),
			search.WithConsequenceHide([]search.ConsequenceHide{*search.NewConsequenceHide("3")}),
			search.WithConsequenceUserData(map[string]any{"banner": "promo"}),
		),
		search.WithRuleConditions([]search.Condition{*search.NewCondition(
			search.WithConditionPattern("{facet:brand}"),
			search.WithConditionAnchoring(search.ANCHORING_CONTAINS),
			search.WithConditionAlternatives(true),
			search.WithConditionContext("mobile"),
		)}),
		search.WithRuleDescription("desc"),
		search.WithRuleEnabled(true),
		search.WithRuleValidity([]search.TimeRange{*search.NewTimeRange(
			search.WithTimeRangeFrom(1893456000),
			search.WithTimeRangeUntil(1893542400),
		)}),
	)

	model := RuleResourceModel{IndexName: types.StringValue("products")}
	diags := hydrateRuleModel("products", ruleResp, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.ID.ValueString(); got != "products/rule-1" {
		t.Fatalf("id = %q, want composite id", got)
	}
	if got := model.ObjectID.ValueString(); got != "rule-1" {
		t.Fatalf("object_id = %q, want %q", got, "rule-1")
	}
	if got := model.Description.ValueString(); got != "desc" {
		t.Fatalf("description = %q, want desc", got)
	}
	if model.Consequence.IsNull() {
		t.Fatal("consequence should be set")
	}
}
