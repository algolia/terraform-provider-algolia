package index

import (
	"context"
	"encoding/json"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/utils"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Attribute types maps for each block, matching the schema exactly.

var attributesAttrTypes = map[string]attr.Type{
	"searchable_attributes":   types.ListType{ElemType: types.StringType},
	"attributes_to_retrieve":  types.ListType{ElemType: types.StringType},
	"unretrievable_attributes": types.ListType{ElemType: types.StringType},
	"attribute_for_distinct":  types.StringType,
}

var rankingAttrTypes = map[string]attr.Type{
	"ranking":               types.ListType{ElemType: types.StringType},
	"custom_ranking":        types.ListType{ElemType: types.StringType},
	"relevancy_strictness":  types.Int64Type,
}

var facetingAttrTypes = map[string]attr.Type{
	"attributes_for_faceting": types.ListType{ElemType: types.StringType},
	"max_facet_hits":          types.Int64Type,
	"max_values_per_facet":    types.Int64Type,
	"sort_facet_values_by":    types.StringType,
}

var highlightingAttrTypes = map[string]attr.Type{
	"attributes_to_highlight":               types.ListType{ElemType: types.StringType},
	"attributes_to_snippet":                 types.ListType{ElemType: types.StringType},
	"highlight_pre_tag":                     types.StringType,
	"highlight_post_tag":                    types.StringType,
	"snippet_ellipsis_text":                 types.StringType,
	"restrict_highlight_and_snippet_arrays": types.BoolType,
}

var paginationAttrTypes = map[string]attr.Type{
	"hits_per_page":          types.Int64Type,
	"pagination_limited_to":  types.Int64Type,
}

var typosAttrTypes = map[string]attr.Type{
	"typo_tolerance":                      types.StringType,
	"min_word_size_for_1_typo":            types.Int64Type,
	"min_word_size_for_2_typos":           types.Int64Type,
	"allow_typos_on_numeric_tokens":       types.BoolType,
	"disable_typo_tolerance_on_attributes": types.ListType{ElemType: types.StringType},
	"disable_typo_tolerance_on_words":     types.ListType{ElemType: types.StringType},
}

var languagesAttrTypes = map[string]attr.Type{
	"index_languages":              types.ListType{ElemType: types.StringType},
	"query_languages":              types.ListType{ElemType: types.StringType},
	"ignore_plurals":               types.BoolType,
	"ignore_plurals_languages":     types.ListType{ElemType: types.StringType},
	"remove_stop_words":            types.BoolType,
	"remove_stop_words_languages":  types.ListType{ElemType: types.StringType},
	"decompound_query":             types.BoolType,
	"remove_words_if_no_results":   types.StringType,
	"attributes_to_transliterate":  types.ListType{ElemType: types.StringType},
	"camel_case_attributes":        types.ListType{ElemType: types.StringType},
	"decompounded_attributes":      types.StringType,
	"custom_normalization":         types.StringType,
	"keep_diacritics_on_characters": types.StringType,
}

var queryStrategyAttrTypes = map[string]attr.Type{
	"query_type":                  types.StringType,
	"advanced_syntax":             types.BoolType,
	"advanced_syntax_features":    types.ListType{ElemType: types.StringType},
	"optional_words":              types.ListType{ElemType: types.StringType},
	"disable_prefix_on_attributes": types.ListType{ElemType: types.StringType},
	"disable_exact_on_attributes": types.ListType{ElemType: types.StringType},
	"exact_on_single_word_query":  types.StringType,
	"alternatives_as_exact":       types.ListType{ElemType: types.StringType},
}

var performanceAttrTypes = map[string]attr.Type{
	"numeric_attributes_for_filtering":  types.ListType{ElemType: types.StringType},
	"allow_compression_of_integer_array": types.BoolType,
}

var advancedAttrTypes = map[string]attr.Type{
	"distinct":                  types.Int64Type,
	"min_proximity":             types.Int64Type,
	"replace_synonyms_in_highlight": types.BoolType,
	"separators_to_index":       types.StringType,
	"response_fields":           types.ListType{ElemType: types.StringType},
	"user_data":                 types.StringType,
	"enable_rules":              types.BoolType,
	"enable_personalization":    types.BoolType,
	"replicas":                  types.ListType{ElemType: types.StringType},
	"enable_re_ranking":         types.BoolType,
	"re_ranking_apply_filter":   types.StringType,
	"mode":                      types.StringType,
	"semantic_search":           types.StringType,
	"attribute_criteria_computed_by_min_proximity": types.BoolType,
}

// flattenSettingsResponse converts an Algolia SettingsResponse to the Terraform resource model.
func flattenSettingsResponse(ctx context.Context, settings *search.SettingsResponse, model *IndexResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	// Map top-level fields. Preserve DeletionProtection from current state (not an API field).
	model.Primary = flattenNullableString(settings.Primary)

	// Attributes block.
	diags.Append(flattenAttributesBlock(ctx, settings, model)...)
	if diags.HasError() {
		return diags
	}

	// Ranking block.
	diags.Append(flattenRankingBlock(ctx, settings, model)...)
	if diags.HasError() {
		return diags
	}

	// Faceting block.
	diags.Append(flattenFacetingBlock(ctx, settings, model)...)
	if diags.HasError() {
		return diags
	}

	// Highlighting block.
	diags.Append(flattenHighlightingBlock(ctx, settings, model)...)
	if diags.HasError() {
		return diags
	}

	// Pagination block.
	diags.Append(flattenPaginationBlock(ctx, settings, model)...)
	if diags.HasError() {
		return diags
	}

	// Typos block.
	diags.Append(flattenTyposBlock(ctx, settings, model)...)
	if diags.HasError() {
		return diags
	}

	// Languages block.
	diags.Append(flattenLanguagesBlock(ctx, settings, model)...)
	if diags.HasError() {
		return diags
	}

	// Query strategy block.
	diags.Append(flattenQueryStrategyBlock(ctx, settings, model)...)
	if diags.HasError() {
		return diags
	}

	// Performance block.
	diags.Append(flattenPerformanceBlock(ctx, settings, model)...)
	if diags.HasError() {
		return diags
	}

	// Advanced block.
	diags.Append(flattenAdvancedBlock(ctx, settings, model)...)
	if diags.HasError() {
		return diags
	}

	return diags
}

// flattenAttributesBlock flattens the attributes block from SettingsResponse.
func flattenAttributesBlock(ctx context.Context, settings *search.SettingsResponse, model *IndexResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	block := AttributesModel{
		SearchableAttributes:    flattenStringList(ctx, settings.SearchableAttributes),
		AttributesToRetrieve:    flattenStringList(ctx, settings.AttributesToRetrieve),
		UnretrievableAttributes: flattenStringList(ctx, settings.UnretrievableAttributes),
		AttributeForDistinct:    flattenNullableString(settings.AttributeForDistinct),
	}

	objVal, d := types.ObjectValueFrom(ctx, attributesAttrTypes, &block)
	diags.Append(d...)
	if !diags.HasError() {
		model.Attributes = objVal
	}

	return diags
}

// flattenRankingBlock flattens the ranking block from SettingsResponse.
func flattenRankingBlock(ctx context.Context, settings *search.SettingsResponse, model *IndexResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	block := RankingModel{
		Ranking:             flattenStringList(ctx, settings.Ranking),
		CustomRanking:       flattenStringList(ctx, settings.CustomRanking),
		RelevancyStrictness: flattenNullableInt32(settings.RelevancyStrictness),
	}

	objVal, d := types.ObjectValueFrom(ctx, rankingAttrTypes, &block)
	diags.Append(d...)
	if !diags.HasError() {
		model.Ranking = objVal
	}

	return diags
}

// flattenFacetingBlock flattens the faceting block from SettingsResponse.
func flattenFacetingBlock(ctx context.Context, settings *search.SettingsResponse, model *IndexResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	block := FacetingModel{
		AttributesForFaceting: flattenStringList(ctx, settings.AttributesForFaceting),
		MaxFacetHits:          flattenNullableInt32(settings.MaxFacetHits),
		MaxValuesPerFacet:     flattenNullableInt32(settings.MaxValuesPerFacet),
		SortFacetValuesBy:     flattenNullableString(settings.SortFacetValuesBy),
	}

	objVal, d := types.ObjectValueFrom(ctx, facetingAttrTypes, &block)
	diags.Append(d...)
	if !diags.HasError() {
		model.Faceting = objVal
	}

	return diags
}

// flattenHighlightingBlock flattens the highlighting block from SettingsResponse.
func flattenHighlightingBlock(ctx context.Context, settings *search.SettingsResponse, model *IndexResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	block := HighlightingModel{
		AttributesToHighlight:             flattenStringList(ctx, settings.AttributesToHighlight),
		AttributesToSnippet:               flattenStringList(ctx, settings.AttributesToSnippet),
		HighlightPreTag:                   flattenNullableString(settings.HighlightPreTag),
		HighlightPostTag:                  flattenNullableString(settings.HighlightPostTag),
		SnippetEllipsisText:               flattenNullableString(settings.SnippetEllipsisText),
		RestrictHighlightAndSnippetArrays: flattenNullableBool(settings.RestrictHighlightAndSnippetArrays),
	}

	objVal, d := types.ObjectValueFrom(ctx, highlightingAttrTypes, &block)
	diags.Append(d...)
	if !diags.HasError() {
		model.Highlighting = objVal
	}

	return diags
}

// flattenPaginationBlock flattens the pagination block from SettingsResponse.
func flattenPaginationBlock(ctx context.Context, settings *search.SettingsResponse, model *IndexResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	block := PaginationModel{
		HitsPerPage:         flattenNullableInt32(settings.HitsPerPage),
		PaginationLimitedTo: flattenNullableInt32(settings.PaginationLimitedTo),
	}

	objVal, d := types.ObjectValueFrom(ctx, paginationAttrTypes, &block)
	diags.Append(d...)
	if !diags.HasError() {
		model.Pagination = objVal
	}

	return diags
}

// flattenTyposBlock flattens the typos block from SettingsResponse.
func flattenTyposBlock(ctx context.Context, settings *search.SettingsResponse, model *IndexResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	block := TyposModel{
		TypoTolerance:                    flattenTypoTolerance(settings.TypoTolerance),
		MinWordSizeFor1Typo:              flattenNullableInt32(settings.MinWordSizefor1Typo),
		MinWordSizeFor2Typos:             flattenNullableInt32(settings.MinWordSizefor2Typos),
		AllowTyposOnNumericTokens:        flattenNullableBool(settings.AllowTyposOnNumericTokens),
		DisableTypoToleranceOnAttributes: flattenStringList(ctx, settings.DisableTypoToleranceOnAttributes),
		DisableTypoToleranceOnWords:      flattenStringList(ctx, settings.DisableTypoToleranceOnWords),
	}

	objVal, d := types.ObjectValueFrom(ctx, typosAttrTypes, &block)
	diags.Append(d...)
	if !diags.HasError() {
		model.Typos = objVal
	}

	return diags
}

// flattenLanguagesBlock flattens the languages block from SettingsResponse.
func flattenLanguagesBlock(ctx context.Context, settings *search.SettingsResponse, model *IndexResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	block := LanguagesModel{
		IndexLanguages:             flattenSupportedLanguageList(ctx, settings.IndexLanguages),
		QueryLanguages:             flattenSupportedLanguageList(ctx, settings.QueryLanguages),
		DecompoundQuery:            flattenNullableBool(settings.DecompoundQuery),
		RemoveWordsIfNoResults:     flattenRemoveWordsIfNoResults(settings.RemoveWordsIfNoResults),
		AttributesToTransliterate:  flattenStringList(ctx, settings.AttributesToTransliterate),
		CamelCaseAttributes:        flattenStringList(ctx, settings.CamelCaseAttributes),
		DecompoundedAttributes:     flattenDecompoundedAttributes(settings.DecompoundedAttributes),
		CustomNormalization:        flattenCustomNormalization(settings.CustomNormalization),
		KeepDiacriticsOnCharacters: flattenNullableString(settings.KeepDiacriticsOnCharacters),
	}

	// Handle IgnorePlurals union type.
	flattenIgnorePlurals(ctx, settings.IgnorePlurals, &block)

	// Handle RemoveStopWords union type.
	flattenRemoveStopWords(ctx, settings.RemoveStopWords, &block)

	objVal, d := types.ObjectValueFrom(ctx, languagesAttrTypes, &block)
	diags.Append(d...)
	if !diags.HasError() {
		model.Languages = objVal
	}

	return diags
}

// flattenQueryStrategyBlock flattens the query_strategy block from SettingsResponse.
func flattenQueryStrategyBlock(ctx context.Context, settings *search.SettingsResponse, model *IndexResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	block := QueryStrategyModel{
		QueryType:                 flattenQueryType(settings.QueryType),
		AdvancedSyntax:            flattenNullableBool(settings.AdvancedSyntax),
		AdvancedSyntaxFeatures:    flattenAdvancedSyntaxFeaturesList(ctx, settings.AdvancedSyntaxFeatures),
		OptionalWords:             flattenOptionalWords(ctx, settings.OptionalWords),
		DisablePrefixOnAttributes: flattenStringList(ctx, settings.DisablePrefixOnAttributes),
		DisableExactOnAttributes:  flattenStringList(ctx, settings.DisableExactOnAttributes),
		ExactOnSingleWordQuery:    flattenExactOnSingleWordQuery(settings.ExactOnSingleWordQuery),
		AlternativesAsExact:       flattenAlternativesAsExactList(ctx, settings.AlternativesAsExact),
	}

	objVal, d := types.ObjectValueFrom(ctx, queryStrategyAttrTypes, &block)
	diags.Append(d...)
	if !diags.HasError() {
		model.QueryStrategy = objVal
	}

	return diags
}

// flattenPerformanceBlock flattens the performance block from SettingsResponse.
func flattenPerformanceBlock(ctx context.Context, settings *search.SettingsResponse, model *IndexResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	block := PerformanceModel{
		NumericAttributesForFiltering:  flattenStringList(ctx, settings.NumericAttributesForFiltering),
		AllowCompressionOfIntegerArray: flattenNullableBool(settings.AllowCompressionOfIntegerArray),
	}

	objVal, d := types.ObjectValueFrom(ctx, performanceAttrTypes, &block)
	diags.Append(d...)
	if !diags.HasError() {
		model.Performance = objVal
	}

	return diags
}

// flattenAdvancedBlock flattens the advanced block from SettingsResponse.
func flattenAdvancedBlock(ctx context.Context, settings *search.SettingsResponse, model *IndexResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	block := AdvancedModel{
		Distinct:                                flattenDistinct(settings.Distinct),
		MinProximity:                            flattenNullableInt32(settings.MinProximity),
		ReplaceSynonymsInHighlight:              flattenNullableBool(settings.ReplaceSynonymsInHighlight),
		SeparatorsToIndex:                       flattenNullableString(settings.SeparatorsToIndex),
		ResponseFields:                          flattenStringList(ctx, settings.ResponseFields),
		UserData:                                flattenUserData(settings.UserData),
		EnableRules:                             flattenNullableBool(settings.EnableRules),
		EnablePersonalization:                   flattenNullableBool(settings.EnablePersonalization),
		Replicas:                                flattenStringList(ctx, settings.Replicas),
		EnableReRanking:                         flattenNullableBool(settings.EnableReRanking),
		ReRankingApplyFilter:                    flattenReRankingApplyFilter(settings.ReRankingApplyFilter),
		Mode:                                    flattenMode(settings.Mode),
		SemanticSearch:                          flattenSemanticSearch(settings.SemanticSearch),
		AttributeCriteriaComputedByMinProximity: flattenNullableBool(settings.AttributeCriteriaComputedByMinProximity),
	}

	objVal, d := types.ObjectValueFrom(ctx, advancedAttrTypes, &block)
	diags.Append(d...)
	if !diags.HasError() {
		model.Advanced = objVal
	}

	return diags
}

// --- Helper functions ---

// flattenStringList converts a []string to a types.List of StringType.
func flattenStringList(ctx context.Context, strings []string) types.List {
	if strings == nil {
		return types.ListNull(types.StringType)
	}

	elems := make([]attr.Value, len(strings))
	for i, s := range strings {
		elems[i] = types.StringValue(s)
	}

	listVal, _ := types.ListValue(types.StringType, elems)
	return listVal
}

// flattenSupportedLanguageList converts a []SupportedLanguage to a types.List of StringType.
func flattenSupportedLanguageList(ctx context.Context, langs []search.SupportedLanguage) types.List {
	if langs == nil {
		return types.ListNull(types.StringType)
	}

	elems := make([]attr.Value, len(langs))
	for i, l := range langs {
		elems[i] = types.StringValue(string(l))
	}

	listVal, _ := types.ListValue(types.StringType, elems)
	return listVal
}

// flattenNullableString converts a *string to types.String.
func flattenNullableString(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

// flattenNullableBool converts a *bool to types.Bool.
func flattenNullableBool(b *bool) types.Bool {
	if b == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*b)
}

// flattenNullableInt32 converts a *int32 to types.Int64.
func flattenNullableInt32(i *int32) types.Int64 {
	if i == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*i))
}

// flattenTypoTolerance converts a *TypoTolerance union type to types.String.
// TypoTolerance can be either a TypoToleranceEnum (string: "min"/"strict"/"true"/"false") or a *bool.
func flattenTypoTolerance(tt *search.TypoTolerance) types.String {
	if tt == nil {
		return types.StringNull()
	}

	switch v := tt.GetActualInstance().(type) {
	case search.TypoToleranceEnum:
		return types.StringValue(string(v))
	case bool:
		if v {
			return types.StringValue("true")
		}
		return types.StringValue("false")
	default:
		return types.StringNull()
	}
}

// flattenDistinct converts a *Distinct union type to types.Int64.
// Distinct can be either a *bool (false=0, true=1) or a *int32.
func flattenDistinct(d *search.Distinct) types.Int64 {
	if d == nil {
		return types.Int64Null()
	}

	switch v := d.GetActualInstance().(type) {
	case bool:
		if v {
			return types.Int64Value(1)
		}
		return types.Int64Value(0)
	case int32:
		return types.Int64Value(int64(v))
	default:
		return types.Int64Null()
	}
}

// flattenIgnorePlurals handles the IgnorePlurals union type.
// It can be a *bool, a *[]SupportedLanguage, or a *BooleanString.
func flattenIgnorePlurals(ctx context.Context, ip *search.IgnorePlurals, block *LanguagesModel) {
	if ip == nil {
		block.IgnorePlurals = types.BoolNull()
		block.IgnorePluralsLanguages = types.ListNull(types.StringType)
		return
	}

	switch v := ip.GetActualInstance().(type) {
	case bool:
		block.IgnorePlurals = types.BoolValue(v)
		block.IgnorePluralsLanguages = types.ListNull(types.StringType)
	case []search.SupportedLanguage:
		block.IgnorePlurals = types.BoolNull()
		block.IgnorePluralsLanguages = flattenSupportedLanguageList(ctx, v)
	case search.BooleanString:
		block.IgnorePlurals = types.BoolValue(v == search.BOOLEAN_STRING_TRUE)
		block.IgnorePluralsLanguages = types.ListNull(types.StringType)
	default:
		block.IgnorePlurals = types.BoolNull()
		block.IgnorePluralsLanguages = types.ListNull(types.StringType)
	}
}

// flattenRemoveStopWords handles the RemoveStopWords union type.
// It can be a *bool or a *[]SupportedLanguage.
func flattenRemoveStopWords(ctx context.Context, rsw *search.RemoveStopWords, block *LanguagesModel) {
	if rsw == nil {
		block.RemoveStopWords = types.BoolNull()
		block.RemoveStopWordsLanguages = types.ListNull(types.StringType)
		return
	}

	switch v := rsw.GetActualInstance().(type) {
	case bool:
		block.RemoveStopWords = types.BoolValue(v)
		block.RemoveStopWordsLanguages = types.ListNull(types.StringType)
	case []search.SupportedLanguage:
		block.RemoveStopWords = types.BoolNull()
		block.RemoveStopWordsLanguages = flattenSupportedLanguageList(ctx, v)
	default:
		block.RemoveStopWords = types.BoolNull()
		block.RemoveStopWordsLanguages = types.ListNull(types.StringType)
	}
}

// flattenOptionalWords handles the Nullable[OptionalWords] union type.
// OptionalWords can be a *[]string or a *string.
func flattenOptionalWords(ctx context.Context, ow utils.Nullable[search.OptionalWords]) types.List {
	if !ow.IsSet() || ow.Get() == nil {
		return types.ListNull(types.StringType)
	}

	val := ow.Get()
	switch v := val.GetActualInstance().(type) {
	case []string:
		return flattenStringList(ctx, v)
	case string:
		// Single string: wrap in a list.
		elems := []attr.Value{types.StringValue(v)}
		listVal, _ := types.ListValue(types.StringType, elems)
		return listVal
	default:
		return types.ListNull(types.StringType)
	}
}

// flattenReRankingApplyFilter converts a *ReRankingApplyFilter union to types.String (JSON-encoded).
func flattenReRankingApplyFilter(rr *search.ReRankingApplyFilter) types.String {
	if rr == nil {
		return types.StringNull()
	}

	switch v := rr.GetActualInstance().(type) {
	case string:
		return types.StringValue(v)
	case []search.ReRankingApplyFilter:
		data, err := json.Marshal(v)
		if err != nil {
			return types.StringNull()
		}
		return types.StringValue(string(data))
	default:
		return types.StringNull()
	}
}

// flattenQueryType converts a *QueryType enum to types.String.
func flattenQueryType(qt *search.QueryType) types.String {
	if qt == nil {
		return types.StringNull()
	}
	return types.StringValue(string(*qt))
}

// flattenRemoveWordsIfNoResults converts a *RemoveWordsIfNoResults enum to types.String.
func flattenRemoveWordsIfNoResults(rw *search.RemoveWordsIfNoResults) types.String {
	if rw == nil {
		return types.StringNull()
	}
	return types.StringValue(string(*rw))
}

// flattenExactOnSingleWordQuery converts a *ExactOnSingleWordQuery enum to types.String.
func flattenExactOnSingleWordQuery(e *search.ExactOnSingleWordQuery) types.String {
	if e == nil {
		return types.StringNull()
	}
	return types.StringValue(string(*e))
}

// flattenMode converts a *Mode enum to types.String.
func flattenMode(m *search.Mode) types.String {
	if m == nil {
		return types.StringNull()
	}
	return types.StringValue(string(*m))
}

// flattenAdvancedSyntaxFeaturesList converts a []AdvancedSyntaxFeatures to a types.List.
func flattenAdvancedSyntaxFeaturesList(ctx context.Context, features []search.AdvancedSyntaxFeatures) types.List {
	if features == nil {
		return types.ListNull(types.StringType)
	}

	elems := make([]attr.Value, len(features))
	for i, f := range features {
		elems[i] = types.StringValue(string(f))
	}

	listVal, _ := types.ListValue(types.StringType, elems)
	return listVal
}

// flattenAlternativesAsExactList converts a []AlternativesAsExact to a types.List.
func flattenAlternativesAsExactList(ctx context.Context, alternatives []search.AlternativesAsExact) types.List {
	if alternatives == nil {
		return types.ListNull(types.StringType)
	}

	elems := make([]attr.Value, len(alternatives))
	for i, a := range alternatives {
		elems[i] = types.StringValue(string(a))
	}

	listVal, _ := types.ListValue(types.StringType, elems)
	return listVal
}

// flattenDecompoundedAttributes converts a map[string]any to a JSON-encoded types.String.
func flattenDecompoundedAttributes(da map[string]any) types.String {
	if da == nil {
		return types.StringNull()
	}

	data, err := json.Marshal(da)
	if err != nil {
		return types.StringNull()
	}
	return types.StringValue(string(data))
}

// flattenCustomNormalization converts a *map[string]map[string]string to a JSON-encoded types.String.
func flattenCustomNormalization(cn *map[string]map[string]string) types.String {
	if cn == nil {
		return types.StringNull()
	}

	data, err := json.Marshal(*cn)
	if err != nil {
		return types.StringNull()
	}
	return types.StringValue(string(data))
}

// flattenUserData converts an any value to a JSON-encoded types.String.
func flattenUserData(ud any) types.String {
	if ud == nil {
		return types.StringNull()
	}

	data, err := json.Marshal(ud)
	if err != nil {
		return types.StringNull()
	}
	return types.StringValue(string(data))
}

// flattenSemanticSearch converts a *SemanticSearch to a JSON-encoded types.String.
func flattenSemanticSearch(ss *search.SemanticSearch) types.String {
	if ss == nil {
		return types.StringNull()
	}

	data, err := json.Marshal(ss)
	if err != nil {
		return types.StringNull()
	}
	return types.StringValue(string(data))
}

