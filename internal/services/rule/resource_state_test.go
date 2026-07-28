package rule

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func consequenceValue(attrs map[string]attr.Value) types.List {
	full := map[string]attr.Value{
		"params_json":         types.StringNull(),
		"promote":             types.ListNull(promoteModelType),
		"hide":                types.SetNull(types.StringType),
		"user_data":           types.StringNull(),
		"filter_promotes":     types.BoolNull(),
		"redirect_index_name": types.StringNull(),
	}
	for name, value := range attrs {
		full[name] = value
	}

	return types.ListValueMust(consequenceModelType, []attr.Value{
		types.ObjectValueMust(consequenceModelAttrTypes, full),
	})
}

func consequenceAttr(t *testing.T, list types.List, name string) attr.Value {
	t.Helper()

	if list.IsNull() || len(list.Elements()) == 0 {
		t.Fatalf("consequence is empty: %s", list)
	}

	return list.Elements()[0].(types.Object).Attributes()[name]
}

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
		Consequence: consequenceValue(map[string]attr.Value{
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
		Description: types.StringValue("test rule"),
		Enabled:     types.BoolValue(true),
		Tags:        types.ListNull(types.StringType),
		Scope:       types.StringNull(),
		Validity: types.ListValueMust(validityModelType, []attr.Value{
			types.ObjectValueMust(validityModelAttrTypes, map[string]attr.Value{
				"from":  types.StringValue("2030-01-01T00:00:00Z"),
				"until": types.StringValue("2030-01-02T00:00:00Z"),
			}),
		}),
	}

	rule, rawParams, diags := buildRuleRequest(&model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := rule.GetObjectID(); got != "rule-1" {
		t.Fatalf("objectID = %q, want %q", got, "rule-1")
	}
	if got := rule.GetDescription(); got != "test rule" {
		t.Fatalf("description = %q, want %q", got, "test rule")
	}
	if string(rawParams) != `{"query":"iphone"}` {
		t.Fatalf("params = %s, want the configured document verbatim", rawParams)
	}
	consequence := rule.GetConsequence()
	if got := consequence.GetPromote(); len(got) != 1 {
		t.Fatalf("promote = %#v, want 1 item", got)
	}
	if got := consequence.GetHide(); len(got) != 1 || got[0].GetObjectID() != "3" {
		t.Fatalf("hide = %#v, want [3]", got)
	}
}

func TestBuildRuleRequestTagsScopeRedirectAndFilterPromotes(t *testing.T) {
	model := RuleResourceModel{
		IndexName: types.StringValue("products"),
		ObjectID:  types.StringValue("rule-1"),
		Tags:      types.ListValueMust(types.StringType, []attr.Value{types.StringValue("seasonal"), types.StringValue("promo")}),
		Scope:     types.StringValue("redirect"),
		Consequence: consequenceValue(map[string]attr.Value{
			"filter_promotes":     types.BoolValue(true),
			"redirect_index_name": types.StringValue("products_virtual_redirect"),
		}),
	}

	rule, _, diags := buildRuleRequest(&model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := rule.Tags; !reflect.DeepEqual(got, []string{"seasonal", "promo"}) {
		t.Errorf("tags = %#v, want [seasonal promo]", got)
	}
	if got := rule.GetScope(); got != "redirect" {
		t.Errorf("scope = %q, want redirect", got)
	}

	consequence := rule.GetConsequence()
	if got := consequence.GetFilterPromotes(); !got {
		t.Errorf("filterPromotes = %v, want true", got)
	}
	if consequence.Redirect == nil {
		t.Fatal("redirect should be set")
	}
	if got := consequence.Redirect.GetIndexName(); got != "products_virtual_redirect" {
		t.Errorf("redirect.indexName = %q, want products_virtual_redirect", got)
	}
}

func TestBuildRuleRequestTagsNullVersusEmpty(t *testing.T) {
	tests := []struct {
		name string
		tags types.List
		want string
	}{
		{
			name: "null tags are omitted",
			tags: types.ListNull(types.StringType),
			want: "",
		},
		{
			name: "configured empty tags are sent as an empty array",
			tags: types.ListValueMust(types.StringType, []attr.Value{}),
			want: "[]",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := RuleResourceModel{
				ObjectID:    types.StringValue("rule-1"),
				Tags:        test.tags,
				Consequence: consequenceValue(nil),
			}

			body, diags := ruleRequestBodyFromModel(&model)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}

			raw, ok := body["tags"]
			if test.want == "" {
				if ok {
					t.Fatalf("tags should be absent, got %#v", raw)
				}
				return
			}

			encoded, err := json.Marshal(raw)
			if err != nil {
				t.Fatalf("encode tags: %v", err)
			}
			if string(encoded) != test.want {
				t.Errorf("tags = %s, want %s", encoded, test.want)
			}
		})
	}
}

func TestRuleRequestBodyPreservesUnmodelledParams(t *testing.T) {
	model := RuleResourceModel{
		ObjectID: types.StringValue("rule-1"),
		Consequence: consequenceValue(map[string]attr.Value{
			"params_json": types.StringValue(`{"query":"iphone","brandNewAlgoliaParam":{"nested":[1,2]}}`),
		}),
	}

	body, diags := ruleRequestBodyFromModel(&model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encode body: %v", err)
	}

	var decoded struct {
		Consequence struct {
			Params map[string]any `json:"params"`
		} `json:"consequence"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got := decoded.Consequence.Params["query"]; got != "iphone" {
		t.Errorf("params.query = %#v, want iphone", got)
	}
	unknown, ok := decoded.Consequence.Params["brandNewAlgoliaParam"]
	if !ok {
		t.Fatalf("brandNewAlgoliaParam was dropped from the request body: %s", encoded)
	}
	if !reflect.DeepEqual(unknown, map[string]any{"nested": []any{float64(1), float64(2)}}) {
		t.Errorf("brandNewAlgoliaParam = %#v, want {nested:[1,2]}", unknown)
	}
}

func TestExtractRawParams(t *testing.T) {
	// A real GetRule payload, including the `_metadata` object the typed models
	// do not declare and an unmodelled consequence param.
	payload := []byte(`{"_metadata":{"lastUpdate":1785253378},"objectID":"rule-1",` +
		`"consequence":{"params":{"query":"iphone","brandNewAlgoliaParam":{"nested":[1,2]}},"filterPromotes":true}}`)

	want := `{"query":"iphone","brandNewAlgoliaParam":{"nested":[1,2]}}`
	if got := string(extractRawParams(payload)); got != want {
		t.Errorf("params = %s, want %s", got, want)
	}
}

func TestExtractRawParamsWhenAbsent(t *testing.T) {
	for name, payload := range map[string]string{
		"no params":      `{"objectID":"rule-1","consequence":{"hide":[{"objectID":"3"}]}}`,
		"no consequence": `{"objectID":"rule-1"}`,
		"not an object":  `[]`,
	} {
		t.Run(name, func(t *testing.T) {
			if got := extractRawParams([]byte(payload)); got != nil {
				t.Errorf("params = %s, want nil", got)
			}
		})
	}
}

func TestHydrateRuleModelPreservesUnmodelledParams(t *testing.T) {
	configured := `{"query":"iphone","brandNewAlgoliaParam":{"nested":[1,2]}}`

	model := RuleResourceModel{
		Consequence: consequenceValue(map[string]attr.Value{
			"params_json": types.StringValue(configured),
		}),
	}

	ruleResp := search.NewRule("rule-1", *search.NewConsequence())
	if diags := hydrateRuleModel("products", ruleResp, json.RawMessage(configured), &model); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	// The configured document is kept verbatim because it carries the same data
	// as the one the API returned.
	if got := consequenceAttr(t, model.Consequence, "params_json"); !got.Equal(types.StringValue(configured)) {
		t.Errorf("params_json = %s, want the configured document verbatim", got)
	}
}

func TestHydrateRuleModelParamsWithoutPrior(t *testing.T) {
	// On import there is no prior document, so the API's own bytes are stored:
	// verbatim, including key order, and with the unmodelled key intact.
	stored := `{"query":"iphone","brandNewAlgoliaParam":true}`

	model := RuleResourceModel{}
	ruleResp := search.NewRule("rule-1", *search.NewConsequence())
	if diags := hydrateRuleModel("products", ruleResp, json.RawMessage(stored), &model); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	want := types.StringValue(stored)
	if got := consequenceAttr(t, model.Consequence, "params_json"); !got.Equal(want) {
		t.Errorf("params_json = %s, want %s", got, want)
	}
}

func TestHydrateRuleModelParamsRewrittenOnDrift(t *testing.T) {
	model := RuleResourceModel{
		Consequence: consequenceValue(map[string]attr.Value{
			"params_json": types.StringValue(`{"query":"iphone"}`),
		}),
	}

	ruleResp := search.NewRule("rule-1", *search.NewConsequence())
	if diags := hydrateRuleModel("products", ruleResp, json.RawMessage(`{"query":"ipad"}`), &model); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	want := types.StringValue(`{"query":"ipad"}`)
	if got := consequenceAttr(t, model.Consequence, "params_json"); !got.Equal(want) {
		t.Errorf("params_json = %s, want %s", got, want)
	}
}

func TestHydrateRuleModelParamsIgnoresFormatting(t *testing.T) {
	configured := "{\n  \"query\": \"iphone\",\n  \"filters\": \"brand:apple\"\n}"

	model := RuleResourceModel{
		Consequence: consequenceValue(map[string]attr.Value{
			"params_json": types.StringValue(configured),
		}),
	}

	ruleResp := search.NewRule("rule-1", *search.NewConsequence())
	if diags := hydrateRuleModel("products", ruleResp, json.RawMessage(`{"filters":"brand:apple","query":"iphone"}`), &model); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := consequenceAttr(t, model.Consequence, "params_json"); !got.Equal(types.StringValue(configured)) {
		t.Errorf("params_json = %s, want the configured document verbatim", got)
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
	diags := hydrateRuleModel("products", ruleResp, json.RawMessage(`{"query":"iphone"}`), &model)
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
	if !model.Tags.IsNull() {
		t.Errorf("tags = %s, want null", model.Tags)
	}
	if !model.Scope.IsNull() {
		t.Errorf("scope = %s, want null", model.Scope)
	}
}

func TestHydrateRuleModelTagsScopeRedirectAndFilterPromotes(t *testing.T) {
	ruleResp := search.NewRule(
		"rule-1",
		*search.NewConsequence(
			search.WithConsequenceFilterPromotes(true),
			search.WithConsequenceRedirect(*search.NewConsequenceRedirect("products_virtual_redirect")),
		),
		search.WithRuleTags([]string{"seasonal", "promo"}),
		search.WithRuleScope("redirect"),
	)

	model := RuleResourceModel{IndexName: types.StringValue("products")}
	if diags := hydrateRuleModel("products", ruleResp, nil, &model); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	wantTags := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("seasonal"), types.StringValue("promo")})
	if !model.Tags.Equal(wantTags) {
		t.Errorf("tags = %s, want %s", model.Tags, wantTags)
	}
	if !model.Scope.Equal(types.StringValue("redirect")) {
		t.Errorf("scope = %s, want redirect", model.Scope)
	}
	if got := consequenceAttr(t, model.Consequence, "filter_promotes"); !got.Equal(types.BoolValue(true)) {
		t.Errorf("filter_promotes = %s, want true", got)
	}
	if got := consequenceAttr(t, model.Consequence, "redirect_index_name"); !got.Equal(types.StringValue("products_virtual_redirect")) {
		t.Errorf("redirect_index_name = %s, want products_virtual_redirect", got)
	}
}

func TestHydrateRuleModelTagsNullVersusEmpty(t *testing.T) {
	tests := []struct {
		name string
		tags []string
		want types.List
	}{
		{
			name: "absent tags stay null",
			tags: nil,
			want: types.ListNull(types.StringType),
		},
		{
			name: "empty tags stay a known empty list",
			tags: []string{},
			want: types.ListValueMust(types.StringType, []attr.Value{}),
		},
		{
			name: "tags round trip",
			tags: []string{"promo"},
			want: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("promo")}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ruleResp := search.NewRule("rule-1", *search.NewConsequence())
			ruleResp.Tags = test.tags

			model := RuleResourceModel{}
			if diags := hydrateRuleModel("products", ruleResp, nil, &model); diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}

			if !model.Tags.Equal(test.want) {
				t.Errorf("tags = %s, want %s", model.Tags, test.want)
			}
		})
	}
}

func TestHydrateRuleModel_ConsequenceHidePreservesPriorEmptiness(t *testing.T) {
	emptySet := types.SetValueMust(types.StringType, []attr.Value{})
	valuedSet := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("3")})
	apiHide := []search.ConsequenceHide{*search.NewConsequenceHide("3")}

	tests := []struct {
		name     string
		prior    types.List
		apiHide  []search.ConsequenceHide
		wantHide types.Set
	}{
		{
			name:     "prior null and API empty stays null",
			prior:    consequenceValue(map[string]attr.Value{"hide": types.SetNull(types.StringType)}),
			wantHide: types.SetNull(types.StringType),
		},
		{
			name:     "prior empty and API empty stays empty",
			prior:    consequenceValue(map[string]attr.Value{"hide": emptySet}),
			wantHide: emptySet,
		},
		{
			name:     "no prior consequence at all and API empty yields null",
			prior:    types.ListNull(consequenceModelType),
			wantHide: types.SetNull(types.StringType),
		},
		{
			name:     "API values replace a null prior",
			prior:    consequenceValue(map[string]attr.Value{"hide": types.SetNull(types.StringType)}),
			apiHide:  apiHide,
			wantHide: valuedSet,
		},
		{
			name:     "API values replace an empty prior",
			prior:    consequenceValue(map[string]attr.Value{"hide": emptySet}),
			apiHide:  apiHide,
			wantHide: valuedSet,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			consequenceOpts := []search.ConsequenceOption{}
			if test.apiHide != nil {
				consequenceOpts = append(consequenceOpts, search.WithConsequenceHide(test.apiHide))
			}

			ruleResp := search.NewRule("rule-1", *search.NewConsequence(consequenceOpts...))

			model := RuleResourceModel{
				IndexName:   types.StringValue("products"),
				Consequence: test.prior,
			}

			diags := hydrateRuleModel("products", ruleResp, nil, &model)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}

			hide, ok := consequenceAttr(t, model.Consequence, "hide").(types.Set)
			if !ok {
				t.Fatalf("hide is not a set: %#v", model.Consequence.Elements()[0])
			}
			if !hide.Equal(test.wantHide) {
				t.Errorf("hide = %s, want %s", hide, test.wantHide)
			}
		})
	}
}
