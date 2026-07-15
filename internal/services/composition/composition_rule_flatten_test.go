package composition

import (
	"encoding/json"
	"testing"

	compositionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/composition"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func newTestCompositionRule() *compositionapi.CompositionRule {
	filters := "brand:apple"
	description := "promote featured products"
	enabled := true

	rule := compositionapi.NewEmptyCompositionRule()
	rule.ObjectID = "rule-1"
	rule.Conditions = []compositionapi.Condition{{Filters: &filters}}
	rule.Consequence = *compositionapi.NewCompositionRuleConsequence(
		*compositionapi.CompositionInjectionBehaviorAsCompositionBehavior(
			compositionapi.NewCompositionInjectionBehavior(
				*compositionapi.NewInjection(
					*compositionapi.NewInjectionMain(
						compositionapi.WithInjectionMainSource(
							*compositionapi.InjectionMainSearchSourceAsInjectionMainSource(
								compositionapi.NewInjectionMainSearchSource(*compositionapi.NewMainSearch("products_featured")),
							),
						),
					),
				),
			),
		),
	)
	rule.Description = &description
	rule.Enabled = &enabled
	rule.Validity = []compositionapi.TimeRange{{From: int64Ptr(1893456000), Until: int64Ptr(1893542400)}}
	rule.Tags = []string{"seasonal"}

	return rule
}

func int64Ptr(v int64) *int64 { return &v }

func TestFlattenCompositionRule_AdoptsAPIEncodingWhenNoPrior(t *testing.T) {
	rule := newTestCompositionRule()

	var model CompositionRuleResourceModel
	diags := flattenCompositionRule("my-composition", rule, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.ID.ValueString(); got != "my-composition/rule-1" {
		t.Fatalf("id = %q, want composite id", got)
	}
	if got := model.CompositionID.ValueString(); got != "my-composition" {
		t.Fatalf("composition_id = %q, want my-composition", got)
	}
	if got := model.ObjectID.ValueString(); got != "rule-1" {
		t.Fatalf("object_id = %q, want rule-1", got)
	}
	if model.Conditions.IsNull() {
		t.Fatal("conditions should be set")
	}
	if model.Consequence.IsNull() {
		t.Fatal("consequence should be set")
	}
	if got := model.Description.ValueString(); got != "promote featured products" {
		t.Fatalf("description = %q, want promote featured products", got)
	}
	if !model.Enabled.ValueBool() {
		t.Fatal("enabled = false, want true")
	}
	if model.Validity.IsNull() {
		t.Fatal("validity should be set")
	}
	if model.Tags.IsNull() {
		t.Fatal("tags should be set")
	}
	if len(model.Tags.Elements()) != 1 {
		t.Fatalf("tags = %v, want 1 element", model.Tags.Elements())
	}

	var decodedConditions []map[string]any
	if err := json.Unmarshal([]byte(model.Conditions.ValueString()), &decodedConditions); err != nil {
		t.Fatalf("conditions is not valid JSON: %v", err)
	}
	if len(decodedConditions) != 1 || decodedConditions[0]["filters"] != "brand:apple" {
		t.Fatalf("conditions = %v, want [{filters: brand:apple}]", decodedConditions)
	}
}

func TestFlattenCompositionRule_PreservesSemanticallyEqualPrior(t *testing.T) {
	rule := newTestCompositionRule()

	priorConditions := `[
		{
			"filters": "brand:apple"
		}
	]`

	model := CompositionRuleResourceModel{
		Conditions: types.StringValue(priorConditions),
	}

	diags := flattenCompositionRule("my-composition", rule, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.Conditions.ValueString(); got != priorConditions {
		t.Fatalf("conditions = %q, want the preserved prior value %q", got, priorConditions)
	}
}

func TestFlattenCompositionRule_AdoptsAPIEncodingWhenPriorDiffers(t *testing.T) {
	rule := newTestCompositionRule()

	model := CompositionRuleResourceModel{
		Conditions: types.StringValue(`[{"filters":"brand:samsung"}]`),
	}

	diags := flattenCompositionRule("my-composition", rule, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.Conditions.ValueString(); got == `[{"filters":"brand:samsung"}]` {
		t.Fatalf("conditions = %q, want it to be replaced by the API's value", got)
	}
}

func TestFlattenCompositionRule_NilConditionsAndValidityAndTags(t *testing.T) {
	rule := compositionapi.NewEmptyCompositionRule()
	rule.ObjectID = "rule-1"
	rule.Consequence = *compositionapi.NewCompositionRuleConsequence(
		*compositionapi.CompositionMultifeedBehaviorAsCompositionBehavior(
			compositionapi.NewCompositionMultifeedBehavior(*compositionapi.NewMultifeed(map[string]compositionapi.FeedInjection{})),
		),
	)

	t.Run("no prior configuration yields null", func(t *testing.T) {
		var model CompositionRuleResourceModel
		diags := flattenCompositionRule("my-composition", rule, &model)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if !model.Conditions.IsNull() {
			t.Fatalf("conditions = %v, want null", model.Conditions.ValueString())
		}
		if !model.Validity.IsNull() {
			t.Fatalf("validity = %v, want null", model.Validity.ValueString())
		}
		if !model.Description.IsNull() {
			t.Fatalf("description = %v, want null", model.Description.ValueString())
		}
		if !model.Tags.IsNull() {
			t.Fatalf("tags = %v, want null", model.Tags)
		}
		// Enabled defaults to true when the API omits it.
		if !model.Enabled.ValueBool() {
			t.Fatal("enabled = false, want true (default)")
		}
	})

	t.Run("a configured prior value is preserved even when the API returns nothing", func(t *testing.T) {
		model := CompositionRuleResourceModel{
			Conditions: types.StringValue(`[{"filters":"brand:apple"}]`),
			Validity:   types.StringValue(`[{"from":1,"until":2}]`),
		}
		diags := flattenCompositionRule("my-composition", rule, &model)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if got := model.Conditions.ValueString(); got != `[{"filters":"brand:apple"}]` {
			t.Fatalf("conditions = %q, want the preserved prior value", got)
		}
		if got := model.Validity.ValueString(); got != `[{"from":1,"until":2}]` {
			t.Fatalf("validity = %q, want the preserved prior value", got)
		}
	})
}

func TestFlattenCompositionRule_DisabledAndNoDescription(t *testing.T) {
	enabled := false
	rule := compositionapi.NewEmptyCompositionRule()
	rule.ObjectID = "rule-1"
	rule.Enabled = &enabled
	rule.Consequence = *compositionapi.NewCompositionRuleConsequence(
		*compositionapi.CompositionMultifeedBehaviorAsCompositionBehavior(
			compositionapi.NewCompositionMultifeedBehavior(*compositionapi.NewMultifeed(map[string]compositionapi.FeedInjection{})),
		),
	)

	var model CompositionRuleResourceModel
	diags := flattenCompositionRule("my-composition", rule, &model)
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
