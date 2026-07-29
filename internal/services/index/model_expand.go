package index

import (
	"context"
	"encoding/json"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/utils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// isKnown returns true if the value is neither null nor unknown.
func isKnown(v interface {
	IsNull() bool
	IsUnknown() bool
}) bool {
	return !v.IsNull() && !v.IsUnknown()
}

// expandIndexSettings converts the Terraform resource model to Algolia IndexSettings.
func expandIndexSettings(ctx context.Context, model *IndexResourceModel) (*search.IndexSettings, diag.Diagnostics) {
	var diags diag.Diagnostics
	settings := search.NewEmptyIndexSettings()

	// Attributes block
	if isKnown(model.Attributes) {
		var attrs AttributesModel
		diags.Append(model.Attributes.As(ctx, &attrs, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if isKnown(attrs.SearchableAttributes) {
			settings.SearchableAttributes = expandStringList(ctx, attrs.SearchableAttributes)
		}
		if isKnown(attrs.AttributesToRetrieve) {
			settings.AttributesToRetrieve = expandStringList(ctx, attrs.AttributesToRetrieve)
		}
		if isKnown(attrs.UnretrievableAttributes) {
			settings.UnretrievableAttributes = expandStringList(ctx, attrs.UnretrievableAttributes)
		}
		if isKnown(attrs.AttributeForDistinct) {
			settings.AttributeForDistinct = utils.ToPtr(attrs.AttributeForDistinct.ValueString())
		}
	}

	// Ranking block
	if isKnown(model.Ranking) {
		var ranking RankingModel
		diags.Append(model.Ranking.As(ctx, &ranking, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if isKnown(ranking.Ranking) {
			settings.Ranking = expandStringList(ctx, ranking.Ranking)
		}
		if isKnown(ranking.CustomRanking) {
			settings.CustomRanking = expandStringList(ctx, ranking.CustomRanking)
		}
		if isKnown(ranking.RelevancyStrictness) {
			settings.RelevancyStrictness = utils.ToPtr(int32(ranking.RelevancyStrictness.ValueInt64()))
		}
	}

	// Faceting block
	if isKnown(model.Faceting) {
		var faceting FacetingModel
		diags.Append(model.Faceting.As(ctx, &faceting, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if isKnown(faceting.AttributesForFaceting) {
			settings.AttributesForFaceting = expandStringList(ctx, faceting.AttributesForFaceting)
		}
		if isKnown(faceting.MaxFacetHits) {
			settings.MaxFacetHits = utils.ToPtr(int32(faceting.MaxFacetHits.ValueInt64()))
		}
		if isKnown(faceting.MaxValuesPerFacet) {
			settings.MaxValuesPerFacet = utils.ToPtr(int32(faceting.MaxValuesPerFacet.ValueInt64()))
		}
		if isKnown(faceting.SortFacetValuesBy) {
			settings.SortFacetValuesBy = utils.ToPtr(faceting.SortFacetValuesBy.ValueString())
		}
	}

	// Highlighting block
	if isKnown(model.Highlighting) {
		var hl HighlightingModel
		diags.Append(model.Highlighting.As(ctx, &hl, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if isKnown(hl.AttributesToHighlight) {
			settings.AttributesToHighlight = expandStringList(ctx, hl.AttributesToHighlight)
		}
		if isKnown(hl.AttributesToSnippet) {
			settings.AttributesToSnippet = expandStringList(ctx, hl.AttributesToSnippet)
		}
		if isKnown(hl.HighlightPreTag) {
			settings.HighlightPreTag = utils.ToPtr(hl.HighlightPreTag.ValueString())
		}
		if isKnown(hl.HighlightPostTag) {
			settings.HighlightPostTag = utils.ToPtr(hl.HighlightPostTag.ValueString())
		}
		if isKnown(hl.SnippetEllipsisText) {
			settings.SnippetEllipsisText = utils.ToPtr(hl.SnippetEllipsisText.ValueString())
		}
		if isKnown(hl.RestrictHighlightAndSnippetArrays) {
			settings.RestrictHighlightAndSnippetArrays = utils.ToPtr(hl.RestrictHighlightAndSnippetArrays.ValueBool())
		}
	}

	// Pagination block
	if isKnown(model.Pagination) {
		var pag PaginationModel
		diags.Append(model.Pagination.As(ctx, &pag, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if isKnown(pag.HitsPerPage) {
			settings.HitsPerPage = utils.ToPtr(int32(pag.HitsPerPage.ValueInt64()))
		}
		if isKnown(pag.PaginationLimitedTo) {
			settings.PaginationLimitedTo = utils.ToPtr(int32(pag.PaginationLimitedTo.ValueInt64()))
		}
	}

	// Typos block
	if isKnown(model.Typos) {
		var typos TyposModel
		diags.Append(model.Typos.As(ctx, &typos, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if isKnown(typos.TypoTolerance) {
			settings.TypoTolerance = expandTypoTolerance(typos.TypoTolerance.ValueString())
		}
		if isKnown(typos.MinWordSizeFor1Typo) {
			settings.MinWordSizefor1Typo = utils.ToPtr(int32(typos.MinWordSizeFor1Typo.ValueInt64()))
		}
		if isKnown(typos.MinWordSizeFor2Typos) {
			settings.MinWordSizefor2Typos = utils.ToPtr(int32(typos.MinWordSizeFor2Typos.ValueInt64()))
		}
		if isKnown(typos.AllowTyposOnNumericTokens) {
			settings.AllowTyposOnNumericTokens = utils.ToPtr(typos.AllowTyposOnNumericTokens.ValueBool())
		}
		if isKnown(typos.DisableTypoToleranceOnAttributes) {
			settings.DisableTypoToleranceOnAttributes = expandStringList(ctx, typos.DisableTypoToleranceOnAttributes)
		}
		if isKnown(typos.DisableTypoToleranceOnWords) {
			settings.DisableTypoToleranceOnWords = expandStringList(ctx, typos.DisableTypoToleranceOnWords)
		}
	}

	// Languages block
	if isKnown(model.Languages) {
		var lang LanguagesModel
		diags.Append(model.Languages.As(ctx, &lang, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if isKnown(lang.IndexLanguages) {
			settings.IndexLanguages = expandSupportedLanguageList(ctx, lang.IndexLanguages)
		}
		if isKnown(lang.QueryLanguages) {
			settings.QueryLanguages = expandSupportedLanguageList(ctx, lang.QueryLanguages)
		}
		if isKnown(lang.IgnorePluralsLanguages) {
			settings.IgnorePlurals = search.ArrayOfSupportedLanguageAsIgnorePlurals(expandSupportedLanguageList(ctx, lang.IgnorePluralsLanguages))
		} else if isKnown(lang.IgnorePlurals) {
			settings.IgnorePlurals = search.BoolAsIgnorePlurals(lang.IgnorePlurals.ValueBool())
		}
		if isKnown(lang.RemoveStopWordsLanguages) {
			settings.RemoveStopWords = search.ArrayOfSupportedLanguageAsRemoveStopWords(expandSupportedLanguageList(ctx, lang.RemoveStopWordsLanguages))
		} else if isKnown(lang.RemoveStopWords) {
			settings.RemoveStopWords = search.BoolAsRemoveStopWords(lang.RemoveStopWords.ValueBool())
		}
		if isKnown(lang.DecompoundQuery) {
			settings.DecompoundQuery = utils.ToPtr(lang.DecompoundQuery.ValueBool())
		}
		if isKnown(lang.RemoveWordsIfNoResults) {
			v := search.RemoveWordsIfNoResults(lang.RemoveWordsIfNoResults.ValueString())
			settings.RemoveWordsIfNoResults = &v
		}
		if isKnown(lang.AttributesToTransliterate) {
			settings.AttributesToTransliterate = expandStringList(ctx, lang.AttributesToTransliterate)
		}
		if isKnown(lang.CamelCaseAttributes) {
			settings.CamelCaseAttributes = expandStringList(ctx, lang.CamelCaseAttributes)
		}
		if isKnown(lang.DecompoundedAttributes) {
			var decompounded map[string]any
			if err := json.Unmarshal([]byte(lang.DecompoundedAttributes.ValueString()), &decompounded); err != nil {
				diags.AddError("Invalid decompounded_attributes", "Failed to parse JSON: "+err.Error())
				return nil, diags
			}
			settings.DecompoundedAttributes = decompounded
		}
		if isKnown(lang.CustomNormalization) {
			var customNorm map[string]map[string]string
			if err := json.Unmarshal([]byte(lang.CustomNormalization.ValueString()), &customNorm); err != nil {
				diags.AddError("Invalid custom_normalization", "Failed to parse JSON: "+err.Error())
				return nil, diags
			}
			settings.CustomNormalization = &customNorm
		}
		if isKnown(lang.KeepDiacriticsOnCharacters) {
			settings.KeepDiacriticsOnCharacters = utils.ToPtr(lang.KeepDiacriticsOnCharacters.ValueString())
		}
	}

	// Query Strategy block
	if isKnown(model.QueryStrategy) {
		var qs QueryStrategyModel
		diags.Append(model.QueryStrategy.As(ctx, &qs, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if isKnown(qs.QueryType) {
			v := search.QueryType(qs.QueryType.ValueString())
			settings.QueryType = &v
		}
		if isKnown(qs.AdvancedSyntax) {
			settings.AdvancedSyntax = utils.ToPtr(qs.AdvancedSyntax.ValueBool())
		}
		if isKnown(qs.AdvancedSyntaxFeatures) {
			settings.AdvancedSyntaxFeatures = expandAdvancedSyntaxFeaturesList(ctx, qs.AdvancedSyntaxFeatures)
		}
		if isKnown(qs.OptionalWords) {
			words := expandStringList(ctx, qs.OptionalWords)
			optWords := search.ArrayOfStringAsOptionalWords(words)
			settings.OptionalWords = *utils.NewNullable(optWords)
		}
		if isKnown(qs.DisablePrefixOnAttributes) {
			settings.DisablePrefixOnAttributes = expandStringList(ctx, qs.DisablePrefixOnAttributes)
		}
		if isKnown(qs.DisableExactOnAttributes) {
			settings.DisableExactOnAttributes = expandStringList(ctx, qs.DisableExactOnAttributes)
		}
		if isKnown(qs.ExactOnSingleWordQuery) {
			v := search.ExactOnSingleWordQuery(qs.ExactOnSingleWordQuery.ValueString())
			settings.ExactOnSingleWordQuery = &v
		}
		if isKnown(qs.AlternativesAsExact) {
			settings.AlternativesAsExact = expandAlternativesAsExactList(ctx, qs.AlternativesAsExact)
		}
	}

	// Performance block
	if isKnown(model.Performance) {
		var perf PerformanceModel
		diags.Append(model.Performance.As(ctx, &perf, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if isKnown(perf.NumericAttributesForFiltering) {
			settings.NumericAttributesForFiltering = expandStringList(ctx, perf.NumericAttributesForFiltering)
		}
		if isKnown(perf.AllowCompressionOfIntegerArray) {
			settings.AllowCompressionOfIntegerArray = utils.ToPtr(perf.AllowCompressionOfIntegerArray.ValueBool())
		}
	}

	// Advanced block
	if isKnown(model.Advanced) {
		var adv AdvancedModel
		diags.Append(model.Advanced.As(ctx, &adv, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if isKnown(adv.Distinct) {
			settings.Distinct = search.Int32AsDistinct(int32(adv.Distinct.ValueInt64()))
		}
		if isKnown(adv.MinProximity) {
			settings.MinProximity = utils.ToPtr(int32(adv.MinProximity.ValueInt64()))
		}
		if isKnown(adv.ReplaceSynonymsInHighlight) {
			settings.ReplaceSynonymsInHighlight = utils.ToPtr(adv.ReplaceSynonymsInHighlight.ValueBool())
		}
		if isKnown(adv.SeparatorsToIndex) {
			settings.SeparatorsToIndex = utils.ToPtr(adv.SeparatorsToIndex.ValueString())
		}
		if isKnown(adv.ResponseFields) {
			settings.ResponseFields = expandStringList(ctx, adv.ResponseFields)
		}
		if isKnown(adv.UserData) {
			var userData any
			if err := json.Unmarshal([]byte(adv.UserData.ValueString()), &userData); err != nil {
				diags.AddError("Invalid user_data", "Failed to parse JSON: "+err.Error())
				return nil, diags
			}
			settings.UserData = userData
		}
		if isKnown(adv.EnableRules) {
			settings.EnableRules = utils.ToPtr(adv.EnableRules.ValueBool())
		}
		if isKnown(adv.EnablePersonalization) {
			settings.EnablePersonalization = utils.ToPtr(adv.EnablePersonalization.ValueBool())
		}
		if isKnown(adv.Replicas) {
			settings.Replicas = expandStringList(ctx, adv.Replicas)
		}
		if isKnown(adv.EnableReRanking) {
			settings.EnableReRanking = utils.ToPtr(adv.EnableReRanking.ValueBool())
		}
		if isKnown(adv.ReRankingApplyFilter) {
			var reRankingFilter search.ReRankingApplyFilter
			if err := json.Unmarshal([]byte(adv.ReRankingApplyFilter.ValueString()), &reRankingFilter); err != nil {
				diags.AddError("Invalid re_ranking_apply_filter", "Failed to parse JSON: "+err.Error())
				return nil, diags
			}
			settings.ReRankingApplyFilter = *utils.NewNullable(&reRankingFilter)
		}
		if isKnown(adv.Mode) {
			v := search.Mode(adv.Mode.ValueString())
			settings.Mode = &v
		}
		if isKnown(adv.SemanticSearch) {
			var semanticSearch search.SemanticSearch
			if err := json.Unmarshal([]byte(adv.SemanticSearch.ValueString()), &semanticSearch); err != nil {
				diags.AddError("Invalid semantic_search", "Failed to parse JSON: "+err.Error())
				return nil, diags
			}
			settings.SemanticSearch = &semanticSearch
		}
		if isKnown(adv.AttributeCriteriaComputedByMinProximity) {
			settings.AttributeCriteriaComputedByMinProximity = utils.ToPtr(adv.AttributeCriteriaComputedByMinProximity.ValueBool())
		}
	}

	return settings, diags
}

// expandTypoTolerance converts a string typo tolerance value to the Algolia TypoTolerance union type.
func expandTypoTolerance(v string) *search.TypoTolerance {
	switch v {
	case "true":
		return search.BoolAsTypoTolerance(true)
	case "false":
		return search.BoolAsTypoTolerance(false)
	case "min":
		return search.TypoToleranceEnumAsTypoTolerance(search.TYPO_TOLERANCE_ENUM_MIN)
	case "strict":
		return search.TypoToleranceEnumAsTypoTolerance(search.TYPO_TOLERANCE_ENUM_STRICT)
	default:
		return nil
	}
}

// expandStringList converts a Terraform types.List of strings to a Go []string.
func expandStringList(ctx context.Context, list types.List) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	elements := make([]types.String, 0, len(list.Elements()))
	list.ElementsAs(ctx, &elements, false)
	result := make([]string, len(elements))
	for i, elem := range elements {
		result[i] = elem.ValueString()
	}
	return result
}

// expandSupportedLanguageList converts a Terraform types.List of strings to a []SupportedLanguage.
func expandSupportedLanguageList(ctx context.Context, list types.List) []search.SupportedLanguage {
	strs := expandStringList(ctx, list)
	if strs == nil {
		return nil
	}
	result := make([]search.SupportedLanguage, len(strs))
	for i, s := range strs {
		result[i] = search.SupportedLanguage(s)
	}
	return result
}

// expandAdvancedSyntaxFeaturesList converts a Terraform types.List of strings to a []AdvancedSyntaxFeatures.
func expandAdvancedSyntaxFeaturesList(ctx context.Context, list types.List) []search.AdvancedSyntaxFeatures {
	strs := expandStringList(ctx, list)
	if strs == nil {
		return nil
	}
	result := make([]search.AdvancedSyntaxFeatures, len(strs))
	for i, s := range strs {
		result[i] = search.AdvancedSyntaxFeatures(s)
	}
	return result
}

// expandAlternativesAsExactList converts a Terraform types.List of strings to a []AlternativesAsExact.
func expandAlternativesAsExactList(ctx context.Context, list types.List) []search.AlternativesAsExact {
	strs := expandStringList(ctx, list)
	if strs == nil {
		return nil
	}
	result := make([]search.AlternativesAsExact, len(strs))
	for i, s := range strs {
		result[i] = search.AlternativesAsExact(s)
	}
	return result
}
