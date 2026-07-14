package recommend

import (
	"encoding/json"
	"testing"

	recommendapi "github.com/algolia/algoliasearch-client-go/v4/algolia/recommend"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func newTestRecommendRule() *recommendapi.RecommendRule {
	filters := "brand:apple"
	description := "hide discontinued items"
	enabled := true
	objectID := "rule-1"

	rule := recommendapi.NewEmptyRecommendRule()
	rule.ObjectID = &objectID
	rule.Condition = &recommendapi.Condition{Filters: &filters}
	rule.Consequence = &recommendapi.Consequence{
		Hide: []recommendapi.HideConsequenceObject{{ObjectID: strPtr("42")}},
	}
	rule.Description = &description
	rule.Enabled = &enabled
	rule.Validity = []recommendapi.TimeRange{{From: int64Ptr(1893456000), Until: int64Ptr(1893542400)}}

	return rule
}

func strPtr(v string) *string { return &v }
func int64Ptr(v int64) *int64 { return &v }

func TestFlattenRecommendRule_AdoptsAPIEncodingWhenNoPrior(t *testing.T) {
	rule := newTestRecommendRule()

	var model RecommendRuleResourceModel
	diags := flattenRecommendRule("products", "related-products", rule, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.ID.ValueString(); got != "products/related-products/rule-1" {
		t.Fatalf("id = %q, want composite id", got)
	}
	if got := model.IndexName.ValueString(); got != "products" {
		t.Fatalf("index_name = %q, want products", got)
	}
	if got := model.Model.ValueString(); got != "related-products" {
		t.Fatalf("model = %q, want related-products", got)
	}
	if got := model.ObjectID.ValueString(); got != "rule-1" {
		t.Fatalf("object_id = %q, want rule-1", got)
	}
	if model.Condition.IsNull() {
		t.Fatal("condition should be set")
	}
	if model.Consequence.IsNull() {
		t.Fatal("consequence should be set")
	}
	if got := model.Description.ValueString(); got != "hide discontinued items" {
		t.Fatalf("description = %q, want hide discontinued items", got)
	}
	if !model.Enabled.ValueBool() {
		t.Fatal("enabled = false, want true")
	}
	if model.Validity.IsNull() {
		t.Fatal("validity should be set")
	}

	var decodedCondition map[string]any
	if err := json.Unmarshal([]byte(model.Condition.ValueString()), &decodedCondition); err != nil {
		t.Fatalf("condition is not valid JSON: %v", err)
	}
	if decodedCondition["filters"] != "brand:apple" {
		t.Fatalf("condition.filters = %v, want brand:apple", decodedCondition["filters"])
	}
}

func TestFlattenRecommendRule_PreservesSemanticallyEqualPrior(t *testing.T) {
	rule := newTestRecommendRule()

	// Deliberately different key order/whitespace from what json.Marshal
	// would produce for rule.Condition, but semantically identical.
	priorCondition := `{
		"filters": "brand:apple"
	}`

	model := RecommendRuleResourceModel{
		Condition: types.StringValue(priorCondition),
	}

	diags := flattenRecommendRule("products", "related-products", rule, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.Condition.ValueString(); got != priorCondition {
		t.Fatalf("condition = %q, want the preserved prior value %q", got, priorCondition)
	}
}

func TestFlattenRecommendRule_AdoptsAPIEncodingWhenPriorDiffers(t *testing.T) {
	rule := newTestRecommendRule()

	model := RecommendRuleResourceModel{
		Condition: types.StringValue(`{"filters":"brand:samsung"}`),
	}

	diags := flattenRecommendRule("products", "related-products", rule, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.Condition.ValueString(); got == `{"filters":"brand:samsung"}` {
		t.Fatalf("condition = %q, want it to be replaced by the API's value", got)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(model.Condition.ValueString()), &decoded); err != nil {
		t.Fatalf("condition is not valid JSON: %v", err)
	}
	if decoded["filters"] != "brand:apple" {
		t.Fatalf("condition.filters = %v, want brand:apple", decoded["filters"])
	}
}

func TestFlattenRecommendRule_NilConditionAndValidity(t *testing.T) {
	objectID := "rule-1"
	rule := recommendapi.NewEmptyRecommendRule()
	rule.ObjectID = &objectID
	rule.Consequence = &recommendapi.Consequence{Hide: []recommendapi.HideConsequenceObject{{ObjectID: strPtr("42")}}}

	t.Run("no prior configuration yields null", func(t *testing.T) {
		var model RecommendRuleResourceModel
		diags := flattenRecommendRule("products", "related-products", rule, &model)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if !model.Condition.IsNull() {
			t.Fatalf("condition = %v, want null", model.Condition.ValueString())
		}
		if !model.Validity.IsNull() {
			t.Fatalf("validity = %v, want null", model.Validity.ValueString())
		}
		if !model.Description.IsNull() {
			t.Fatalf("description = %v, want null", model.Description.ValueString())
		}
		// Enabled defaults to true when the API omits it.
		if !model.Enabled.ValueBool() {
			t.Fatal("enabled = false, want true (default)")
		}
	})

	t.Run("a configured prior value is preserved even when the API returns nothing", func(t *testing.T) {
		model := RecommendRuleResourceModel{
			Condition: types.StringValue(`{"filters":"brand:apple"}`),
			Validity:  types.StringValue(`[{"from":1,"until":2}]`),
		}
		diags := flattenRecommendRule("products", "related-products", rule, &model)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if got := model.Condition.ValueString(); got != `{"filters":"brand:apple"}` {
			t.Fatalf("condition = %q, want the preserved prior value", got)
		}
		if got := model.Validity.ValueString(); got != `[{"from":1,"until":2}]` {
			t.Fatalf("validity = %q, want the preserved prior value", got)
		}
	})
}

func TestFlattenRecommendRule_DisabledAndNoDescription(t *testing.T) {
	objectID := "rule-1"
	enabled := false
	rule := recommendapi.NewEmptyRecommendRule()
	rule.ObjectID = &objectID
	rule.Enabled = &enabled
	rule.Consequence = &recommendapi.Consequence{Hide: []recommendapi.HideConsequenceObject{{ObjectID: strPtr("42")}}}

	var model RecommendRuleResourceModel
	diags := flattenRecommendRule("products", "related-products", rule, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Enabled.ValueBool() {
		t.Fatal("enabled = true, want false")
	}
	if !model.Description.IsNull() {
		t.Fatalf("description = %v, want null", model.Description.ValueString())
	}
}
