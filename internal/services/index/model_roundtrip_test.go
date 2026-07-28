package index

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// This file exercises expand and flatten as a pair: for every settings block,
// every union-typed field and every JSON-encoded field, flatten(expand(model))
// must reproduce model exactly.
//
// Each direction is easy to get subtly wrong on its own -- a field added to one
// and forgotten in the other, or a union arm that expands one way and flattens
// another -- and neither unit test suite catches it, because they only ever look
// at one direction. Terraform sees such a mismatch as a value that changes
// between plan and apply, or as a permanent diff that never converges.

// settingsToResponse bridges expand's output to flatten's input the way the
// Algolia API does: IndexSettings is serialised onto the wire and comes back as a
// SettingsResponse. Anything the two structs disagree about (a differing JSON tag,
// a field present in one and missing from the other) surfaces here rather than
// only in an acceptance test.
func settingsToResponse(t *testing.T, settings *search.IndexSettings) *search.SettingsResponse {
	t.Helper()

	encoded, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("marshalling expanded IndexSettings: %v", err)
	}

	var resp search.SettingsResponse
	if err := json.Unmarshal(encoded, &resp); err != nil {
		t.Fatalf("decoding %s into SettingsResponse: %v", encoded, err)
	}

	return &resp
}

// blockObject builds a block object exactly as flatten does, so that a fixture
// and a flattened result are directly comparable.
func blockObject[T any](t *testing.T, attrTypes map[string]attr.Type, block T) types.Object {
	t.Helper()

	obj, diags := types.ObjectValueFrom(context.Background(), attrTypes, &block)
	if diags.HasError() {
		t.Fatalf("building block fixture: %v", diags)
	}

	return obj
}

// roundTrip runs model through expand, over the wire and back through flatten.
func roundTrip(t *testing.T, model IndexResourceModel) IndexResourceModel {
	t.Helper()

	ctx := context.Background()

	settings, diags := expandIndexSettings(ctx, &model)
	if diags.HasError() {
		t.Fatalf("expandIndexSettings: %v", diags)
	}

	var result IndexResourceModel
	result.Name = model.Name

	if diags := flattenSettingsResponse(ctx, settingsToResponse(t, settings), &result); diags.HasError() {
		t.Fatalf("flattenSettingsResponse: %v", diags)
	}

	return result
}

// assertBlocksEqual compares every settings block of want and got.
func assertBlocksEqual(t *testing.T, want, got IndexResourceModel) {
	t.Helper()

	blocks := []struct {
		name string
		want types.Object
		got  types.Object
	}{
		{"attributes", want.Attributes, got.Attributes},
		{"ranking", want.Ranking, got.Ranking},
		{"faceting", want.Faceting, got.Faceting},
		{"highlighting", want.Highlighting, got.Highlighting},
		{"pagination", want.Pagination, got.Pagination},
		{"typos", want.Typos, got.Typos},
		{"languages", want.Languages, got.Languages},
		{"query_strategy", want.QueryStrategy, got.QueryStrategy},
		{"performance", want.Performance, got.Performance},
		{"advanced", want.Advanced, got.Advanced},
	}

	for _, block := range blocks {
		if !block.want.Equal(block.got) {
			t.Errorf("%s block did not round-trip\n want: %s\n  got: %s", block.name, block.want, block.got)
		}
	}
}

func stringList(t *testing.T, values ...string) types.List {
	t.Helper()

	elems := make([]attr.Value, len(values))
	for i, v := range values {
		elems[i] = types.StringValue(v)
	}

	list, diags := types.ListValue(types.StringType, elems)
	if diags.HasError() {
		t.Fatalf("building string list fixture: %v", diags)
	}

	return list
}

// fullAttributes and friends build a block with every attribute set, so that a
// field dropped by either direction shows up as a null in the comparison.

func fullAttributes(t *testing.T) types.Object {
	return blockObject(t, attributesAttrTypes, AttributesModel{
		SearchableAttributes:    stringList(t, "title", "unordered(description)"),
		AttributesToRetrieve:    stringList(t, "title", "description"),
		UnretrievableAttributes: stringList(t, "internal_score"),
		AttributeForDistinct:    types.StringValue("sku"),
	})
}

func fullRanking(t *testing.T) types.Object {
	return blockObject(t, rankingAttrTypes, RankingModel{
		Ranking:             stringList(t, "typo", "geo", "words", "filters", "proximity", "attribute", "exact", "custom"),
		CustomRanking:       stringList(t, "desc(popularity)", "asc(price)"),
		RelevancyStrictness: types.Int64Value(90),
	})
}

func fullFaceting(t *testing.T) types.Object {
	return blockObject(t, facetingAttrTypes, FacetingModel{
		AttributesForFaceting: stringList(t, "brand", "searchable(category)"),
		MaxFacetHits:          types.Int64Value(50),
		MaxValuesPerFacet:     types.Int64Value(200),
		SortFacetValuesBy:     types.StringValue("count"),
	})
}

func fullHighlighting(t *testing.T) types.Object {
	return blockObject(t, highlightingAttrTypes, HighlightingModel{
		AttributesToHighlight:             stringList(t, "title", "description"),
		AttributesToSnippet:               stringList(t, "description:20"),
		HighlightPreTag:                   types.StringValue("<mark>"),
		HighlightPostTag:                  types.StringValue("</mark>"),
		SnippetEllipsisText:               types.StringValue("..."),
		RestrictHighlightAndSnippetArrays: types.BoolValue(true),
	})
}

func fullPagination(t *testing.T) types.Object {
	return blockObject(t, paginationAttrTypes, PaginationModel{
		HitsPerPage:         types.Int64Value(25),
		PaginationLimitedTo: types.Int64Value(500),
	})
}

// fullTypos takes the typo_tolerance value because it is a union: "true"/"false"
// travel as a JSON boolean and "min"/"strict" as a JSON string.
func fullTypos(t *testing.T, typoTolerance string) types.Object {
	return blockObject(t, typosAttrTypes, TyposModel{
		TypoTolerance:                    types.StringValue(typoTolerance),
		MinWordSizeFor1Typo:              types.Int64Value(4),
		MinWordSizeFor2Typos:             types.Int64Value(8),
		AllowTyposOnNumericTokens:        types.BoolValue(false),
		DisableTypoToleranceOnAttributes: stringList(t, "sku"),
		DisableTypoToleranceOnWords:      stringList(t, "iphone"),
	})
}

// fullLanguages leaves the union fields to the caller. ignore_plurals and
// remove_stop_words each collapse a bool-or-language-list union onto two
// Terraform fields, and only one of the two can be set at a time: expand prefers
// the language list, and flatten nulls the bool whenever languages come back.
func fullLanguages(t *testing.T, ignorePlurals, removeStopWords types.Bool, ignorePluralsLangs, removeStopWordsLangs types.List) types.Object {
	return blockObject(t, languagesAttrTypes, LanguagesModel{
		IndexLanguages:             stringList(t, "de", "nl"),
		QueryLanguages:             stringList(t, "de", "nl"),
		IgnorePlurals:              ignorePlurals,
		IgnorePluralsLanguages:     ignorePluralsLangs,
		RemoveStopWords:            removeStopWords,
		RemoveStopWordsLanguages:   removeStopWordsLangs,
		DecompoundQuery:            types.BoolValue(true),
		RemoveWordsIfNoResults:     types.StringValue("lastWords"),
		AttributesToTransliterate:  stringList(t, "title"),
		CamelCaseAttributes:        stringList(t, "description"),
		DecompoundedAttributes:     types.StringValue(`{"de":["name","description"]}`),
		CustomNormalization:        types.StringValue(`{"default":{"ä":"ae","ö":"oe"}}`),
		KeepDiacriticsOnCharacters: types.StringValue("øé"),
	})
}

func fullQueryStrategy(t *testing.T) types.Object {
	return blockObject(t, queryStrategyAttrTypes, QueryStrategyModel{
		QueryType:                 types.StringValue("prefixLast"),
		AdvancedSyntax:            types.BoolValue(true),
		AdvancedSyntaxFeatures:    stringList(t, "exactPhrase", "excludeWords"),
		OptionalWords:             stringList(t, "the", "and"),
		DisablePrefixOnAttributes: stringList(t, "sku"),
		DisableExactOnAttributes:  stringList(t, "description"),
		ExactOnSingleWordQuery:    types.StringValue("attribute"),
		AlternativesAsExact:       stringList(t, "ignorePlurals", "singleWordSynonym"),
	})
}

func fullPerformance(t *testing.T) types.Object {
	return blockObject(t, performanceAttrTypes, PerformanceModel{
		NumericAttributesForFiltering:  stringList(t, "price", "equalOnly(stock)"),
		AllowCompressionOfIntegerArray: types.BoolValue(true),
	})
}

// fullAdvanced takes distinct because it is a union of bool and int32, and
// re_ranking_apply_filter because it is a union of string and nested array.
func fullAdvanced(t *testing.T, distinct int64, reRankingApplyFilter string) types.Object {
	return blockObject(t, advancedAttrTypes, AdvancedModel{
		Distinct:                   types.Int64Value(distinct),
		MinProximity:               types.Int64Value(3),
		ReplaceSynonymsInHighlight: types.BoolValue(false),
		SeparatorsToIndex:          types.StringValue("+#"),
		ResponseFields:             stringList(t, "hits", "nbHits"),
		// Object keys have to be alphabetical and numbers integral: this value is
		// decoded into an any and re-encoded by encoding/json, which sorts keys and
		// renders whole floats without a fraction.
		UserData:                                types.StringValue(`{"environment":"test","tags":["a","b"],"version":2}`),
		EnableRules:                             types.BoolValue(true),
		EnablePersonalization:                   types.BoolValue(false),
		Replicas:                                stringList(t, "tf-test-replica"),
		EnableReRanking:                         types.BoolValue(true),
		ReRankingApplyFilter:                    types.StringValue(reRankingApplyFilter),
		Mode:                                    types.StringValue("neuralSearch"),
		SemanticSearch:                          types.StringValue(`{"eventSources":["tf-test-events"]}`),
		AttributeCriteriaComputedByMinProximity: types.BoolValue(true),
	})
}

// fullModel populates all ten settings blocks. The parameters are the union arms
// that cannot all be exercised at once.
func fullModel(t *testing.T, typoTolerance string, distinct int64, reRankingApplyFilter string, ignorePluralsAsLanguages bool) IndexResourceModel {
	t.Helper()

	ignorePlurals, removeStopWords := types.BoolValue(true), types.BoolValue(true)
	ignorePluralsLangs, removeStopWordsLangs := types.ListNull(types.StringType), types.ListNull(types.StringType)
	if ignorePluralsAsLanguages {
		ignorePlurals, removeStopWords = types.BoolNull(), types.BoolNull()
		ignorePluralsLangs, removeStopWordsLangs = stringList(t, "de", "nl"), stringList(t, "de", "nl")
	}

	return IndexResourceModel{
		Name:          types.StringValue("tf-test-roundtrip"),
		Attributes:    fullAttributes(t),
		Ranking:       fullRanking(t),
		Faceting:      fullFaceting(t),
		Highlighting:  fullHighlighting(t),
		Pagination:    fullPagination(t),
		Typos:         fullTypos(t, typoTolerance),
		Languages:     fullLanguages(t, ignorePlurals, removeStopWords, ignorePluralsLangs, removeStopWordsLangs),
		QueryStrategy: fullQueryStrategy(t),
		Performance:   fullPerformance(t),
		Advanced:      fullAdvanced(t, distinct, reRankingApplyFilter),
	}
}

func TestExpandFlattenRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		// typoTolerance covers the bool-or-enum union.
		typoTolerance string
		// distinct covers the bool-or-int32 union: 0 and 1 are the booleans, 2+ are
		// group sizes.
		distinct int64
		// reRankingApplyFilter covers the string-or-nested-array union.
		reRankingApplyFilter string
		// ignorePluralsAsLanguages switches ignore_plurals and remove_stop_words
		// from their boolean arm to their language-list arm.
		ignorePluralsAsLanguages bool
	}{
		{
			name:                 "typo_tolerance true, distinct off, filter array",
			typoTolerance:        "true",
			distinct:             0,
			reRankingApplyFilter: `["brand:apple"]`,
		},
		{
			name:                 "typo_tolerance false, distinct on, filter array",
			typoTolerance:        "false",
			distinct:             1,
			reRankingApplyFilter: `["brand:apple","category:phone"]`,
		},
		{
			name:                 "typo_tolerance min, distinct group size",
			typoTolerance:        "min",
			distinct:             3,
			reRankingApplyFilter: `["brand:apple"]`,
		},
		{
			name:                 "typo_tolerance strict, languages as lists",
			typoTolerance:        "strict",
			distinct:             2,
			reRankingApplyFilter: `["brand:apple"]`,

			ignorePluralsAsLanguages: true,
		},
		{
			name:                 "re_ranking_apply_filter as a bare string",
			typoTolerance:        "min",
			distinct:             1,
			reRankingApplyFilter: `"brand:apple"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := fullModel(t, tt.typoTolerance, tt.distinct, tt.reRankingApplyFilter, tt.ignorePluralsAsLanguages)
			assertBlocksEqual(t, want, roundTrip(t, want))
		})
	}
}

// TestExpandFlattenRoundTripNullBlocks checks the other end of the range: a model
// with no blocks set must not gain values on the way back. flatten always builds
// all ten block objects, so each one has to come back with every attribute null.
func TestExpandFlattenRoundTripNullBlocks(t *testing.T) {
	model := IndexResourceModel{
		Name:          types.StringValue("tf-test-roundtrip-empty"),
		Attributes:    types.ObjectNull(attributesAttrTypes),
		Ranking:       types.ObjectNull(rankingAttrTypes),
		Faceting:      types.ObjectNull(facetingAttrTypes),
		Highlighting:  types.ObjectNull(highlightingAttrTypes),
		Pagination:    types.ObjectNull(paginationAttrTypes),
		Typos:         types.ObjectNull(typosAttrTypes),
		Languages:     types.ObjectNull(languagesAttrTypes),
		QueryStrategy: types.ObjectNull(queryStrategyAttrTypes),
		Performance:   types.ObjectNull(performanceAttrTypes),
		Advanced:      types.ObjectNull(advancedAttrTypes),
	}

	got := roundTrip(t, model)

	blocks := map[string]types.Object{
		"attributes":     got.Attributes,
		"ranking":        got.Ranking,
		"faceting":       got.Faceting,
		"highlighting":   got.Highlighting,
		"pagination":     got.Pagination,
		"typos":          got.Typos,
		"languages":      got.Languages,
		"query_strategy": got.QueryStrategy,
		"performance":    got.Performance,
		"advanced":       got.Advanced,
	}

	for name, block := range blocks {
		if block.IsNull() {
			continue
		}
		for attrName, value := range block.Attributes() {
			if !value.IsNull() {
				t.Errorf("%s.%s = %s after round-tripping an empty model, want null", name, attrName, value)
			}
		}
	}
}

// TestExpandFlattenRoundTripPartialBlock covers a block where only some
// attributes are set, the shape a minimal Terraform configuration produces.
func TestExpandFlattenRoundTripPartialBlock(t *testing.T) {
	model := IndexResourceModel{
		Name: types.StringValue("tf-test-roundtrip-partial"),
		Attributes: blockObject(t, attributesAttrTypes, AttributesModel{
			SearchableAttributes:    stringList(t, "title"),
			AttributesToRetrieve:    types.ListNull(types.StringType),
			UnretrievableAttributes: types.ListNull(types.StringType),
			AttributeForDistinct:    types.StringNull(),
		}),
		Advanced: blockObject(t, advancedAttrTypes, AdvancedModel{
			Distinct:                                types.Int64Null(),
			MinProximity:                            types.Int64Null(),
			ReplaceSynonymsInHighlight:              types.BoolNull(),
			SeparatorsToIndex:                       types.StringNull(),
			ResponseFields:                          types.ListNull(types.StringType),
			UserData:                                types.StringNull(),
			EnableRules:                             types.BoolValue(false),
			EnablePersonalization:                   types.BoolNull(),
			Replicas:                                types.ListNull(types.StringType),
			EnableReRanking:                         types.BoolNull(),
			ReRankingApplyFilter:                    types.StringNull(),
			Mode:                                    types.StringNull(),
			SemanticSearch:                          types.StringNull(),
			AttributeCriteriaComputedByMinProximity: types.BoolNull(),
		}),
	}

	got := roundTrip(t, model)

	if !model.Attributes.Equal(got.Attributes) {
		t.Errorf("attributes block did not round-trip\n want: %s\n  got: %s", model.Attributes, got.Attributes)
	}
	if !model.Advanced.Equal(got.Advanced) {
		t.Errorf("advanced block did not round-trip\n want: %s\n  got: %s", model.Advanced, got.Advanced)
	}
}

// TestExpandFlattenRoundTripEmptyLists distinguishes an empty list from a null
// one. Terraform treats `searchable_attributes = []` (turn the feature off) and an
// absent block (leave the default) as different configurations, and the two must
// not collapse into each other across a round trip.
func TestExpandFlattenRoundTripEmptyLists(t *testing.T) {
	empty, diags := types.ListValue(types.StringType, []attr.Value{})
	if diags.HasError() {
		t.Fatalf("building empty list fixture: %v", diags)
	}

	model := IndexResourceModel{
		Name: types.StringValue("tf-test-roundtrip-empty-lists"),
		Attributes: blockObject(t, attributesAttrTypes, AttributesModel{
			SearchableAttributes:    empty,
			AttributesToRetrieve:    empty,
			UnretrievableAttributes: empty,
			AttributeForDistinct:    types.StringValue(""),
		}),
	}

	got := roundTrip(t, model)

	if !model.Attributes.Equal(got.Attributes) {
		t.Errorf("attributes block with empty lists did not round-trip\n want: %s\n  got: %s", model.Attributes, got.Attributes)
	}
}
