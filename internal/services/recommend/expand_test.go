package recommend

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExpandRecommendRule(t *testing.T) {
	model := &RecommendRuleResourceModel{
		Condition:   types.StringValue(`{"filters":"brand:apple","context":"mobile"}`),
		Consequence: types.StringValue(`{"hide":[{"objectID":"42"}],"promote":[{"objectID":"7","position":0}]}`),
		Description: types.StringValue("hide discontinued items"),
		Enabled:     types.BoolValue(true),
		Validity:    types.StringValue(`[{"from":1893456000,"until":1893542400}]`),
	}

	rule, diags := expandRecommendRule("rule-1", model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := rule.GetObjectID(); got != "rule-1" {
		t.Fatalf("objectID = %q, want %q", got, "rule-1")
	}
	if rule.Condition == nil || rule.Condition.GetFilters() != "brand:apple" || rule.Condition.GetContext() != "mobile" {
		t.Fatalf("condition = %#v, want filters=brand:apple context=mobile", rule.Condition)
	}
	if rule.Consequence == nil {
		t.Fatal("expected consequence to be set")
	}
	consequence := rule.GetConsequence()
	if got := consequence.GetHide(); len(got) != 1 || got[0].GetObjectID() != "42" {
		t.Fatalf("consequence.hide = %#v, want [42]", got)
	}
	if got := consequence.GetPromote(); len(got) != 1 || got[0].GetObjectID() != "7" {
		t.Fatalf("consequence.promote = %#v, want [7]", got)
	}
	if got := rule.GetDescription(); got != "hide discontinued items" {
		t.Fatalf("description = %q, want %q", got, "hide discontinued items")
	}
	if !rule.GetEnabled() {
		t.Fatal("expected enabled to be true")
	}
	if got := rule.GetValidity(); len(got) != 1 || got[0].GetFrom() != 1893456000 || got[0].GetUntil() != 1893542400 {
		t.Fatalf("validity = %#v, want [{from:1893456000 until:1893542400}]", got)
	}
}

func TestExpandRecommendRuleMinimal(t *testing.T) {
	model := &RecommendRuleResourceModel{
		Condition:   types.StringNull(),
		Consequence: types.StringValue(`{"hide":[{"objectID":"42"}]}`),
		Description: types.StringNull(),
		Enabled:     types.BoolNull(),
		Validity:    types.StringNull(),
	}

	rule, diags := expandRecommendRule("rule-2", model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if rule.Condition != nil {
		t.Fatalf("condition = %#v, want nil", rule.Condition)
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
}

func TestExpandRecommendRuleMissingConsequence(t *testing.T) {
	model := &RecommendRuleResourceModel{
		Consequence: types.StringNull(),
	}

	if _, diags := expandRecommendRule("rule-1", model); !diags.HasError() {
		t.Fatal("expected an error for a missing consequence")
	}
}

func TestExpandRecommendRuleInvalidConditionJSON(t *testing.T) {
	model := &RecommendRuleResourceModel{
		Condition:   types.StringValue(`not valid json`),
		Consequence: types.StringValue(`{"hide":[{"objectID":"42"}]}`),
	}

	if _, diags := expandRecommendRule("rule-1", model); !diags.HasError() {
		t.Fatal("expected an error for invalid condition JSON")
	}
}

func TestExpandRecommendRuleInvalidConsequenceJSON(t *testing.T) {
	model := &RecommendRuleResourceModel{
		Consequence: types.StringValue(`not valid json`),
	}

	if _, diags := expandRecommendRule("rule-1", model); !diags.HasError() {
		t.Fatal("expected an error for invalid consequence JSON")
	}
}

func TestExpandRecommendRuleInvalidValidityJSON(t *testing.T) {
	model := &RecommendRuleResourceModel{
		Consequence: types.StringValue(`{"hide":[{"objectID":"42"}]}`),
		Validity:    types.StringValue(`not valid json`),
	}

	if _, diags := expandRecommendRule("rule-1", model); !diags.HasError() {
		t.Fatal("expected an error for invalid validity JSON")
	}
}

func TestParseRecommendRuleImportID(t *testing.T) {
	indexName, model, objectID, err := parseRecommendRuleImportID("products/related-products/rule-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if indexName != "products" || model != "related-products" || objectID != "rule-1" {
		t.Fatalf("got (%q, %q, %q), want (products, related-products, rule-1)", indexName, model, objectID)
	}
}

func TestParseRecommendRuleImportIDInvalid(t *testing.T) {
	for _, id := range []string{
		"",
		"products",
		"products/related-products",
		"products/related-products/",
		"products//rule-1",
		"/related-products/rule-1",
		"products/related-products/rule-1/extra",
	} {
		if _, _, _, err := parseRecommendRuleImportID(id); err == nil {
			t.Fatalf("expected an error for import ID %q", id)
		}
	}
}

func TestRecommendRuleResourceID(t *testing.T) {
	if got := recommendRuleResourceID("products", "related-products", "rule-1"); got != "products/related-products/rule-1" {
		t.Fatalf("recommendRuleResourceID = %q, want products/related-products/rule-1", got)
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

// TestAllowedRecommendModelStrings asserts baseline membership rather than
// an exact count/order: the Go client's RecommendModels enum may grow new
// values (it has in the past - "looking-similar" is documented upstream but
// not yet present in every client version), and this test should not need
// updating every time it does.
func TestAllowedRecommendModelStrings(t *testing.T) {
	values := allowedRecommendModelStrings()

	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}

	for _, want := range []string{"related-products", "bought-together", "trending-items", "trending-facets"} {
		if !set[want] {
			t.Fatalf("allowedRecommendModelStrings() = %v, want it to include %q", values, want)
		}
	}
}
