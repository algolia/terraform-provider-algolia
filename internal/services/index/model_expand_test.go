package index

import (
	"context"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// expandTypoTolerance
// ---------------------------------------------------------------------------

func TestExpandTypoTolerance(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantNil  bool
		wantBool *bool
		wantEnum *search.TypoToleranceEnum
	}{
		{
			name:     "true returns bool true",
			input:    "true",
			wantBool: boolPtr(true),
		},
		{
			name:     "false returns bool false",
			input:    "false",
			wantBool: boolPtr(false),
		},
		{
			name:     "min returns enum min",
			input:    "min",
			wantEnum: typoEnumPtr(search.TYPO_TOLERANCE_ENUM_MIN),
		},
		{
			name:     "strict returns enum strict",
			input:    "strict",
			wantEnum: typoEnumPtr(search.TYPO_TOLERANCE_ENUM_STRICT),
		},
		{
			name:    "unknown returns nil",
			input:   "unknown",
			wantNil: true,
		},
		{
			name:    "empty string returns nil",
			input:   "",
			wantNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := expandTypoTolerance(tc.input)

			if tc.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected non-nil, got nil")
			}

			actual := got.GetActualInstance()

			if tc.wantBool != nil {
				v, ok := actual.(bool)
				if !ok {
					t.Fatalf("expected bool instance, got %T", actual)
				}
				if v != *tc.wantBool {
					t.Errorf("expected %v, got %v", *tc.wantBool, v)
				}
			}

			if tc.wantEnum != nil {
				v, ok := actual.(search.TypoToleranceEnum)
				if !ok {
					t.Fatalf("expected TypoToleranceEnum instance, got %T", actual)
				}
				if v != *tc.wantEnum {
					t.Errorf("expected %v, got %v", *tc.wantEnum, v)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// expandDistinct (tested via expandIndexSettings advanced block)
// ---------------------------------------------------------------------------

func TestExpandDistinct(t *testing.T) {
	// expandIndexSettings reads Distinct from the Advanced block as an Int64
	// and calls search.Int32AsDistinct. We verify the constructor produces the
	// expected actual-instance values for a range of integers.
	tests := []struct {
		name  string
		input int32
		want  int32
	}{
		{"zero", 0, 0},
		{"one", 1, 1},
		{"two", 2, 2},
		{"four", 4, 4},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := search.Int32AsDistinct(tc.input)
			if d == nil {
				t.Fatal("expected non-nil Distinct")
			}
			actual := d.GetActualInstance()
			v, ok := actual.(int32)
			if !ok {
				t.Fatalf("expected int32 instance, got %T", actual)
			}
			if v != tc.want {
				t.Errorf("expected %d, got %d", tc.want, v)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// expandIgnorePlurals (tested via expandIndexSettings languages block)
// ---------------------------------------------------------------------------

func TestExpandIgnorePlurals(t *testing.T) {
	t.Run("bool only", func(t *testing.T) {
		ip := search.BoolAsIgnorePlurals(true)
		if ip == nil {
			t.Fatal("expected non-nil IgnorePlurals")
		}
		v, ok := ip.GetActualInstance().(bool)
		if !ok {
			t.Fatalf("expected bool instance, got %T", ip.GetActualInstance())
		}
		if !v {
			t.Error("expected true, got false")
		}
	})

	t.Run("languages only", func(t *testing.T) {
		langs := []search.SupportedLanguage{
			search.SupportedLanguage("en"),
			search.SupportedLanguage("fr"),
		}
		ip := search.ArrayOfSupportedLanguageAsIgnorePlurals(langs)
		if ip == nil {
			t.Fatal("expected non-nil IgnorePlurals")
		}
		v, ok := ip.GetActualInstance().([]search.SupportedLanguage)
		if !ok {
			t.Fatalf("expected []SupportedLanguage instance, got %T", ip.GetActualInstance())
		}
		if len(v) != 2 {
			t.Fatalf("expected 2 languages, got %d", len(v))
		}
		if string(v[0]) != "en" {
			t.Errorf("expected 'en', got %q", string(v[0]))
		}
		if string(v[1]) != "fr" {
			t.Errorf("expected 'fr', got %q", string(v[1]))
		}
	})
}

// ---------------------------------------------------------------------------
// expandRemoveStopWords (tested via expandIndexSettings languages block)
// ---------------------------------------------------------------------------

func TestExpandRemoveStopWords(t *testing.T) {
	t.Run("bool only", func(t *testing.T) {
		rsw := search.BoolAsRemoveStopWords(false)
		if rsw == nil {
			t.Fatal("expected non-nil RemoveStopWords")
		}
		v, ok := rsw.GetActualInstance().(bool)
		if !ok {
			t.Fatalf("expected bool instance, got %T", rsw.GetActualInstance())
		}
		if v {
			t.Error("expected false, got true")
		}
	})

	t.Run("languages only", func(t *testing.T) {
		langs := []search.SupportedLanguage{
			search.SupportedLanguage("de"),
			search.SupportedLanguage("es"),
		}
		rsw := search.ArrayOfSupportedLanguageAsRemoveStopWords(langs)
		if rsw == nil {
			t.Fatal("expected non-nil RemoveStopWords")
		}
		v, ok := rsw.GetActualInstance().([]search.SupportedLanguage)
		if !ok {
			t.Fatalf("expected []SupportedLanguage instance, got %T", rsw.GetActualInstance())
		}
		if len(v) != 2 {
			t.Fatalf("expected 2 languages, got %d", len(v))
		}
		if string(v[0]) != "de" {
			t.Errorf("expected 'de', got %q", string(v[0]))
		}
		if string(v[1]) != "es" {
			t.Errorf("expected 'es', got %q", string(v[1]))
		}
	})
}

// ---------------------------------------------------------------------------
// expandStringList
// ---------------------------------------------------------------------------

func TestExpandStringList(t *testing.T) {
	ctx := context.Background()

	t.Run("null list returns nil", func(t *testing.T) {
		nullList := types.ListNull(types.StringType)
		got := expandStringList(ctx, nullList)
		if got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})

	t.Run("empty list returns empty slice", func(t *testing.T) {
		emptyList, diags := types.ListValue(types.StringType, []attr.Value{})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		got := expandStringList(ctx, emptyList)
		if got == nil {
			t.Fatal("expected non-nil slice for empty list")
		}
		if len(got) != 0 {
			t.Fatalf("expected length 0, got %d", len(got))
		}
	})

	t.Run("valued list returns correct strings", func(t *testing.T) {
		list := types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("title"),
			types.StringValue("description"),
			types.StringValue("tags"),
		})
		got := expandStringList(ctx, list)
		if len(got) != 3 {
			t.Fatalf("expected 3 elements, got %d", len(got))
		}
		expected := []string{"title", "description", "tags"}
		for i, want := range expected {
			if got[i] != want {
				t.Errorf("index %d: expected %q, got %q", i, want, got[i])
			}
		}
	})
}

// ---------------------------------------------------------------------------
// expandSupportedLanguageList
// ---------------------------------------------------------------------------

func TestExpandSupportedLanguageList(t *testing.T) {
	ctx := context.Background()

	t.Run("null list returns nil", func(t *testing.T) {
		nullList := types.ListNull(types.StringType)
		got := expandSupportedLanguageList(ctx, nullList)
		if got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})

	t.Run("valued list returns SupportedLanguage values", func(t *testing.T) {
		list := types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("en"),
			types.StringValue("fr"),
			types.StringValue("ja"),
		})
		got := expandSupportedLanguageList(ctx, list)
		if len(got) != 3 {
			t.Fatalf("expected 3 elements, got %d", len(got))
		}
		expected := []string{"en", "fr", "ja"}
		for i, want := range expected {
			if string(got[i]) != want {
				t.Errorf("index %d: expected %q, got %q", i, want, string(got[i]))
			}
		}
	})
}

// ---------------------------------------------------------------------------
// expandIndexSettings
// ---------------------------------------------------------------------------

func TestExpandIndexSettings(t *testing.T) {
	ctx := context.Background()

	t.Run("nil blocks produce empty settings", func(t *testing.T) {
		model := &IndexResourceModel{
			Name:               types.StringValue("test-index"),
			DeletionProtection: types.BoolValue(false),
			Attributes:         types.ObjectNull(attributesAttrTypes),
			Ranking:            types.ObjectNull(rankingAttrTypes),
			Faceting:           types.ObjectNull(facetingAttrTypes),
			Highlighting:       types.ObjectNull(highlightingAttrTypes),
			Pagination:         types.ObjectNull(paginationAttrTypes),
			Typos:              types.ObjectNull(typosAttrTypes),
			Languages:          types.ObjectNull(languagesAttrTypes),
			QueryStrategy:      types.ObjectNull(queryStrategyAttrTypes),
			Performance:        types.ObjectNull(performanceAttrTypes),
			Advanced:           types.ObjectNull(advancedAttrTypes),
		}

		settings, diags := expandIndexSettings(ctx, model)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if settings == nil {
			t.Fatal("expected non-nil settings")
		}
		// With all null blocks, no fields should be set.
		if settings.SearchableAttributes != nil {
			t.Error("expected SearchableAttributes to be nil")
		}
		if settings.TypoTolerance != nil {
			t.Error("expected TypoTolerance to be nil")
		}
		if settings.Distinct != nil {
			t.Error("expected Distinct to be nil")
		}
	})

	t.Run("populated attributes block", func(t *testing.T) {
		searchableAttrs := types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("title"),
			types.StringValue("body"),
		})
		attrsObj, diags := types.ObjectValueFrom(ctx, attributesAttrTypes, &AttributesModel{
			SearchableAttributes:    searchableAttrs,
			AttributesToRetrieve:    types.ListNull(types.StringType),
			UnretrievableAttributes: types.ListNull(types.StringType),
			AttributeForDistinct:    types.StringValue("sku"),
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags building attributes object: %v", diags)
		}

		model := &IndexResourceModel{
			Name:               types.StringValue("test-index"),
			DeletionProtection: types.BoolValue(false),
			Attributes:         attrsObj,
			Ranking:            types.ObjectNull(rankingAttrTypes),
			Faceting:           types.ObjectNull(facetingAttrTypes),
			Highlighting:       types.ObjectNull(highlightingAttrTypes),
			Pagination:         types.ObjectNull(paginationAttrTypes),
			Typos:              types.ObjectNull(typosAttrTypes),
			Languages:          types.ObjectNull(languagesAttrTypes),
			QueryStrategy:      types.ObjectNull(queryStrategyAttrTypes),
			Performance:        types.ObjectNull(performanceAttrTypes),
			Advanced:           types.ObjectNull(advancedAttrTypes),
		}

		settings, settingsDiags := expandIndexSettings(ctx, model)
		if settingsDiags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", settingsDiags)
		}

		if settings.SearchableAttributes == nil {
			t.Fatal("expected SearchableAttributes to be set")
		}
		if len(settings.SearchableAttributes) != 2 {
			t.Fatalf("expected 2 searchable attributes, got %d", len(settings.SearchableAttributes))
		}
		if settings.SearchableAttributes[0] != "title" {
			t.Errorf("expected 'title', got %q", settings.SearchableAttributes[0])
		}
		if settings.SearchableAttributes[1] != "body" {
			t.Errorf("expected 'body', got %q", settings.SearchableAttributes[1])
		}

		if settings.AttributeForDistinct == nil {
			t.Fatal("expected AttributeForDistinct to be set")
		}
		if *settings.AttributeForDistinct != "sku" {
			t.Errorf("expected 'sku', got %q", *settings.AttributeForDistinct)
		}

		// Null lists should remain nil.
		if settings.AttributesToRetrieve != nil {
			t.Error("expected AttributesToRetrieve to be nil")
		}
		if settings.UnretrievableAttributes != nil {
			t.Error("expected UnretrievableAttributes to be nil")
		}
	})

	t.Run("populated typos block", func(t *testing.T) {
		disableOnAttrs := types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("code"),
		})
		typosObj, diags := types.ObjectValueFrom(ctx, typosAttrTypes, &TyposModel{
			TypoTolerance:                    types.StringValue("strict"),
			MinWordSizeFor1Typo:              types.Int64Value(4),
			MinWordSizeFor2Typos:             types.Int64Value(8),
			AllowTyposOnNumericTokens:        types.BoolValue(false),
			DisableTypoToleranceOnAttributes: disableOnAttrs,
			DisableTypoToleranceOnWords:      types.ListNull(types.StringType),
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags building typos object: %v", diags)
		}

		model := &IndexResourceModel{
			Name:               types.StringValue("test-index"),
			DeletionProtection: types.BoolValue(false),
			Attributes:         types.ObjectNull(attributesAttrTypes),
			Ranking:            types.ObjectNull(rankingAttrTypes),
			Faceting:           types.ObjectNull(facetingAttrTypes),
			Highlighting:       types.ObjectNull(highlightingAttrTypes),
			Pagination:         types.ObjectNull(paginationAttrTypes),
			Typos:              typosObj,
			Languages:          types.ObjectNull(languagesAttrTypes),
			QueryStrategy:      types.ObjectNull(queryStrategyAttrTypes),
			Performance:        types.ObjectNull(performanceAttrTypes),
			Advanced:           types.ObjectNull(advancedAttrTypes),
		}

		settings, settingsDiags := expandIndexSettings(ctx, model)
		if settingsDiags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", settingsDiags)
		}

		if settings.TypoTolerance == nil {
			t.Fatal("expected TypoTolerance to be set")
		}
		v, ok := settings.TypoTolerance.GetActualInstance().(search.TypoToleranceEnum)
		if !ok {
			t.Fatalf("expected TypoToleranceEnum, got %T", settings.TypoTolerance.GetActualInstance())
		}
		if v != search.TYPO_TOLERANCE_ENUM_STRICT {
			t.Errorf("expected STRICT, got %v", v)
		}

		if settings.MinWordSizefor1Typo == nil || *settings.MinWordSizefor1Typo != 4 {
			t.Errorf("expected MinWordSizefor1Typo=4, got %v", settings.MinWordSizefor1Typo)
		}
		if settings.MinWordSizefor2Typos == nil || *settings.MinWordSizefor2Typos != 8 {
			t.Errorf("expected MinWordSizefor2Typos=8, got %v", settings.MinWordSizefor2Typos)
		}
		if settings.AllowTyposOnNumericTokens == nil || *settings.AllowTyposOnNumericTokens != false {
			t.Error("expected AllowTyposOnNumericTokens=false")
		}
		if len(settings.DisableTypoToleranceOnAttributes) != 1 || settings.DisableTypoToleranceOnAttributes[0] != "code" {
			t.Errorf("expected DisableTypoToleranceOnAttributes=['code'], got %v", settings.DisableTypoToleranceOnAttributes)
		}
	})

	t.Run("populated advanced block with distinct", func(t *testing.T) {
		advObj, diags := types.ObjectValueFrom(ctx, advancedAttrTypes, &AdvancedModel{
			Distinct:                                types.Int64Value(2),
			MinProximity:                            types.Int64Null(),
			ReplaceSynonymsInHighlight:              types.BoolNull(),
			SeparatorsToIndex:                       types.StringNull(),
			ResponseFields:                          types.ListNull(types.StringType),
			UserData:                                types.StringNull(),
			RenderingContent:                        types.StringNull(),
			EnableRules:                             types.BoolNull(),
			EnablePersonalization:                   types.BoolNull(),
			Replicas:                                types.ListNull(types.StringType),
			EnableReRanking:                         types.BoolNull(),
			ReRankingApplyFilter:                    types.StringNull(),
			Mode:                                    types.StringNull(),
			SemanticSearch:                          types.StringNull(),
			AttributeCriteriaComputedByMinProximity: types.BoolNull(),
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags building advanced object: %v", diags)
		}

		model := &IndexResourceModel{
			Name:               types.StringValue("test-index"),
			DeletionProtection: types.BoolValue(false),
			Attributes:         types.ObjectNull(attributesAttrTypes),
			Ranking:            types.ObjectNull(rankingAttrTypes),
			Faceting:           types.ObjectNull(facetingAttrTypes),
			Highlighting:       types.ObjectNull(highlightingAttrTypes),
			Pagination:         types.ObjectNull(paginationAttrTypes),
			Typos:              types.ObjectNull(typosAttrTypes),
			Languages:          types.ObjectNull(languagesAttrTypes),
			QueryStrategy:      types.ObjectNull(queryStrategyAttrTypes),
			Performance:        types.ObjectNull(performanceAttrTypes),
			Advanced:           advObj,
		}

		settings, settingsDiags := expandIndexSettings(ctx, model)
		if settingsDiags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", settingsDiags)
		}

		if settings.Distinct == nil {
			t.Fatal("expected Distinct to be set")
		}
		dv, ok := settings.Distinct.GetActualInstance().(int32)
		if !ok {
			t.Fatalf("expected int32 instance, got %T", settings.Distinct.GetActualInstance())
		}
		if dv != 2 {
			t.Errorf("expected distinct=2, got %d", dv)
		}
	})

	t.Run("rendering content", func(t *testing.T) {
		for _, tt := range []struct {
			name      string
			value     types.String
			wantError bool
			wantSet   bool
		}{
			{name: "null", value: types.StringNull()},
			{name: "empty string", value: types.StringValue(""), wantError: true},
			{name: "null JSON", value: types.StringValue(`null`), wantError: true},
			{name: "trailing JSON", value: types.StringValue(`{} {}`), wantError: true},
			{name: "empty object", value: types.StringValue(`{}`), wantSet: true},
			{name: "populated", value: types.StringValue(`{"facetOrdering":{"facets":{"order":["brand"]}}}`), wantSet: true},
			{name: "unsupported field", value: types.StringValue(`{"futureField":true}`), wantError: true},
			{name: "case variant field", value: types.StringValue(`{"FacetOrdering":{"facets":{"order":["brand"]}}}`), wantError: true},
		} {
			t.Run(tt.name, func(t *testing.T) {
				advObj, diags := types.ObjectValueFrom(ctx, advancedAttrTypes, &AdvancedModel{
					Distinct:                                types.Int64Null(),
					MinProximity:                            types.Int64Null(),
					ReplaceSynonymsInHighlight:              types.BoolNull(),
					SeparatorsToIndex:                       types.StringNull(),
					ResponseFields:                          types.ListNull(types.StringType),
					UserData:                                types.StringNull(),
					RenderingContent:                        tt.value,
					EnableRules:                             types.BoolNull(),
					EnablePersonalization:                   types.BoolNull(),
					Replicas:                                types.ListNull(types.StringType),
					EnableReRanking:                         types.BoolNull(),
					ReRankingApplyFilter:                    types.StringNull(),
					Mode:                                    types.StringNull(),
					SemanticSearch:                          types.StringNull(),
					AttributeCriteriaComputedByMinProximity: types.BoolNull(),
				})
				if diags.HasError() {
					t.Fatalf("building advanced block: %v", diags)
				}

				model := &IndexResourceModel{
					Advanced: advObj,
				}
				settings, expandDiags := expandIndexSettings(ctx, model)
				if expandDiags.HasError() != tt.wantError {
					t.Fatalf("diagnostics error = %t, want %t: %v", expandDiags.HasError(), tt.wantError, expandDiags)
				}
				if tt.wantError {
					return
				}
				if (settings.RenderingContent != nil) != tt.wantSet {
					t.Errorf("RenderingContent set = %t, want %t", settings.RenderingContent != nil, tt.wantSet)
				}
			})
		}
	})
}

// ---------------------------------------------------------------------------
// expandAdvancedSyntaxFeaturesList
// ---------------------------------------------------------------------------

func TestExpandAdvancedSyntaxFeaturesList(t *testing.T) {
	ctx := context.Background()

	t.Run("null list returns nil", func(t *testing.T) {
		nullList := types.ListNull(types.StringType)
		got := expandAdvancedSyntaxFeaturesList(ctx, nullList)
		if got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})

	t.Run("valued list", func(t *testing.T) {
		list := types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("exactPhrase"),
			types.StringValue("excludeWords"),
		})
		got := expandAdvancedSyntaxFeaturesList(ctx, list)
		if len(got) != 2 {
			t.Fatalf("expected 2 elements, got %d", len(got))
		}
		if string(got[0]) != "exactPhrase" {
			t.Errorf("expected 'exactPhrase', got %q", string(got[0]))
		}
		if string(got[1]) != "excludeWords" {
			t.Errorf("expected 'excludeWords', got %q", string(got[1]))
		}
	})
}

// ---------------------------------------------------------------------------
// expandAlternativesAsExactList
// ---------------------------------------------------------------------------

func TestExpandAlternativesAsExactList(t *testing.T) {
	ctx := context.Background()

	t.Run("null list returns nil", func(t *testing.T) {
		nullList := types.ListNull(types.StringType)
		got := expandAlternativesAsExactList(ctx, nullList)
		if got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})

	t.Run("valued list", func(t *testing.T) {
		list := types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("ignorePlurals"),
			types.StringValue("singleWordSynonym"),
		})
		got := expandAlternativesAsExactList(ctx, list)
		if len(got) != 2 {
			t.Fatalf("expected 2 elements, got %d", len(got))
		}
		if string(got[0]) != "ignorePlurals" {
			t.Errorf("expected 'ignorePlurals', got %q", string(got[0]))
		}
		if string(got[1]) != "singleWordSynonym" {
			t.Errorf("expected 'singleWordSynonym', got %q", string(got[1]))
		}
	})
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func boolPtr(b bool) *bool {
	return &b
}

func typoEnumPtr(e search.TypoToleranceEnum) *search.TypoToleranceEnum {
	return &e
}
