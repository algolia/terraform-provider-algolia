package composition

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExpandCompositionRule(t *testing.T) {
	tagsList, diags := types.ListValue(types.StringType, []attr.Value{types.StringValue("seasonal")})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics building tags list: %v", diags)
	}

	model := &CompositionRuleResourceModel{
		Conditions:  types.StringValue(`[{"filters":"brand:apple","context":"mobile"}]`),
		Consequence: types.StringValue(`{"behavior":{"injection":{"main":{"source":{"search":{"index":"products_featured"}}}}}}`),
		Description: types.StringValue("promote featured products on mobile"),
		Enabled:     types.BoolValue(true),
		Validity:    types.StringValue(`[{"from":1893456000,"until":1893542400}]`),
		Tags:        tagsList,
	}

	rule, expandDiags := expandCompositionRule("rule-1", model)
	if expandDiags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", expandDiags)
	}

	if got := rule.GetObjectID(); got != "rule-1" {
		t.Fatalf("objectID = %q, want %q", got, "rule-1")
	}
	if len(rule.Conditions) != 1 || rule.Conditions[0].GetFilters() != "brand:apple" || rule.Conditions[0].GetContext() != "mobile" {
		t.Fatalf("conditions = %#v, want [filters=brand:apple context=mobile]", rule.Conditions)
	}
	if rule.Consequence.Behavior.CompositionInjectionBehavior == nil {
		t.Fatal("expected consequence.behavior to decode as an injection behavior")
	}
	if got := rule.Consequence.Behavior.CompositionInjectionBehavior.Injection.Main.GetSource().InjectionMainSearchSource.Search.Index; got != "products_featured" {
		t.Fatalf("consequence.behavior.injection.main.source.search.index = %q, want %q", got, "products_featured")
	}
	if got := rule.GetDescription(); got != "promote featured products on mobile" {
		t.Fatalf("description = %q, want %q", got, "promote featured products on mobile")
	}
	if !rule.GetEnabled() {
		t.Fatal("expected enabled to be true")
	}
	if got := rule.GetValidity(); len(got) != 1 || got[0].GetFrom() != 1893456000 || got[0].GetUntil() != 1893542400 {
		t.Fatalf("validity = %#v, want [{from:1893456000 until:1893542400}]", got)
	}
	if got := rule.GetTags(); len(got) != 1 || got[0] != "seasonal" {
		t.Fatalf("tags = %#v, want [seasonal]", got)
	}
}

func TestExpandCompositionRuleMinimal(t *testing.T) {
	model := &CompositionRuleResourceModel{
		Conditions:  types.StringNull(),
		Consequence: types.StringValue(`{"behavior":{"multifeed":{"feeds":{}}}}`),
		Description: types.StringNull(),
		Enabled:     types.BoolNull(),
		Validity:    types.StringNull(),
		Tags:        types.ListNull(types.StringType),
	}

	rule, diags := expandCompositionRule("rule-2", model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if rule.Conditions != nil {
		t.Fatalf("conditions = %#v, want nil", rule.Conditions)
	}
	if rule.Description != nil {
		t.Fatalf("description = %#v, want nil", rule.Description)
	}
	if rule.Enabled != nil {
		t.Fatalf("enabled = %#v, want nil", rule.Enabled)
	}
	if rule.Validity != nil {
		t.Fatalf("validity = %#v, want nil", rule.Validity)
	}
	if rule.Tags != nil {
		t.Fatalf("tags = %#v, want nil", rule.Tags)
	}
}

func TestExpandCompositionRuleMissingConsequence(t *testing.T) {
	model := &CompositionRuleResourceModel{
		Consequence: types.StringNull(),
	}

	if _, diags := expandCompositionRule("rule-1", model); !diags.HasError() {
		t.Fatal("expected an error for a missing consequence")
	}
}

func TestExpandCompositionRuleInvalidConditionsJSON(t *testing.T) {
	model := &CompositionRuleResourceModel{
		Conditions:  types.StringValue(`not valid json`),
		Consequence: types.StringValue(`{"behavior":{"multifeed":{"feeds":{}}}}`),
	}

	if _, diags := expandCompositionRule("rule-1", model); !diags.HasError() {
		t.Fatal("expected an error for invalid conditions JSON")
	}
}

func TestExpandCompositionRuleInvalidConsequenceJSON(t *testing.T) {
	model := &CompositionRuleResourceModel{
		Consequence: types.StringValue(`not valid json`),
	}

	if _, diags := expandCompositionRule("rule-1", model); !diags.HasError() {
		t.Fatal("expected an error for invalid consequence JSON")
	}
}

func TestExpandCompositionRuleInvalidValidityJSON(t *testing.T) {
	model := &CompositionRuleResourceModel{
		Consequence: types.StringValue(`{"behavior":{"multifeed":{"feeds":{}}}}`),
		Validity:    types.StringValue(`not valid json`),
	}

	if _, diags := expandCompositionRule("rule-1", model); !diags.HasError() {
		t.Fatal("expected an error for invalid validity JSON")
	}
}

func TestParseCompositionRuleImportID(t *testing.T) {
	compositionID, objectID, err := parseCompositionRuleImportID("my-composition/rule-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compositionID != "my-composition" || objectID != "rule-1" {
		t.Fatalf("got (%q, %q), want (my-composition, rule-1)", compositionID, objectID)
	}
}

func TestParseCompositionRuleImportIDInvalid(t *testing.T) {
	for _, id := range []string{
		"",
		"my-composition",
		"my-composition/",
		"/rule-1",
	} {
		if _, _, err := parseCompositionRuleImportID(id); err == nil {
			t.Fatalf("expected an error for import ID %q", id)
		}
	}
}

func TestCompositionRuleResourceID(t *testing.T) {
	if got := compositionRuleResourceID("my-composition", "rule-1"); got != "my-composition/rule-1" {
		t.Fatalf("compositionRuleResourceID = %q, want my-composition/rule-1", got)
	}
}

func TestGenerateObjectIDIsUniqueUUIDv4(t *testing.T) {
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

	first, err := generateObjectID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := generateObjectID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first == second {
		t.Fatalf("expected generateObjectID to produce unique values, got %q twice", first)
	}
	if !uuidPattern.MatchString(first) {
		t.Fatalf("generated object_id %q does not look like a UUIDv4", first)
	}
	if !uuidPattern.MatchString(second) {
		t.Fatalf("generated object_id %q does not look like a UUIDv4", second)
	}
}
