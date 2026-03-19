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

// expandIndexSettings converts the Terraform resource model to Algolia IndexSettings.
func expandIndexSettings(ctx context.Context, model *IndexResourceModel) (*search.IndexSettings, diag.Diagnostics) {
	var diags diag.Diagnostics
	settings := search.NewEmptyIndexSettings()

	// Attributes block
	if !model.Attributes.IsNull() {
		var attrs AttributesModel
		diags.Append(model.Attributes.As(ctx, &attrs, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if !attrs.SearchableAttributes.IsNull() {
			settings.SearchableAttributes = expandStringList(ctx, attrs.SearchableAttributes)
		}
		if !attrs.AttributesToRetrieve.IsNull() {
			settings.AttributesToRetrieve = expandStringList(ctx, attrs.AttributesToRetrieve)
		}
		if !attrs.UnretrievableAttributes.IsNull() {
			settings.UnretrievableAttributes = expandStringList(ctx, attrs.UnretrievableAttributes)
		}
		if !attrs.AttributeForDistinct.IsNull() {
			settings.AttributeForDistinct = utils.ToPtr(attrs.AttributeForDistinct.ValueString())
		}
	}

	// Ranking block
	if !model.Ranking.IsNull() {
		var ranking RankingModel
		diags.Append(model.Ranking.As(ctx, &ranking, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if !ranking.Ranking.IsNull() {
			settings.Ranking = expandStringList(ctx, ranking.Ranking)
		}
		if !ranking.CustomRanking.IsNull() {
			settings.CustomRanking = expandStringList(ctx, ranking.CustomRanking)
		}
		if !ranking.RelevancyStrictness.IsNull() {
			settings.RelevancyStrictness = utils.ToPtr(int32(ranking.RelevancyStrictness.ValueInt64()))
		}
	}

	// Faceting block
	if !model.Faceting.IsNull() {
		var faceting FacetingModel
		diags.Append(model.Faceting.As(ctx, &faceting, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if !faceting.AttributesForFaceting.IsNull() {
			settings.AttributesForFaceting = expandStringList(ctx, faceting.AttributesForFaceting)
		}
		if !faceting.MaxFacetHits.IsNull() {
			settings.MaxFacetHits = utils.ToPtr(int32(faceting.MaxFacetHits.ValueInt64()))
		}
		if !faceting.MaxValuesPerFacet.IsNull() {
			settings.MaxValuesPerFacet = utils.ToPtr(int32(faceting.MaxValuesPerFacet.ValueInt64()))
		}
		if !faceting.SortFacetValuesBy.IsNull() {
			settings.SortFacetValuesBy = utils.ToPtr(faceting.SortFacetValuesBy.ValueString())
		}
	}

	// Highlighting block
	if !model.Highlighting.IsNull() {
		var hl HighlightingModel
		diags.Append(model.Highlighting.As(ctx, &hl, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if !hl.AttributesToHighlight.IsNull() {
			settings.AttributesToHighlight = expandStringList(ctx, hl.AttributesToHighlight)
		}
		if !hl.AttributesToSnippet.IsNull() {
			settings.AttributesToSnippet = expandStringList(ctx, hl.AttributesToSnippet)
		}
		if !hl.HighlightPreTag.IsNull() {
			settings.HighlightPreTag = utils.ToPtr(hl.HighlightPreTag.ValueString())
		}
		if !hl.HighlightPostTag.IsNull() {
			settings.HighlightPostTag = utils.ToPtr(hl.HighlightPostTag.ValueString())
		}
		if !hl.SnippetEllipsisText.IsNull() {
			settings.SnippetEllipsisText = utils.ToPtr(hl.SnippetEllipsisText.ValueString())
		}
		if !hl.RestrictHighlightAndSnippetArrays.IsNull() {
			settings.RestrictHighlightAndSnippetArrays = utils.ToPtr(hl.RestrictHighlightAndSnippetArrays.ValueBool())
		}
	}

	// Pagination block
	if !model.Pagination.IsNull() {
		var pag PaginationModel
		diags.Append(model.Pagination.As(ctx, &pag, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if !pag.HitsPerPage.IsNull() {
			settings.HitsPerPage = utils.ToPtr(int32(pag.HitsPerPage.ValueInt64()))
		}
		if !pag.PaginationLimitedTo.IsNull() {
			settings.PaginationLimitedTo = utils.ToPtr(int32(pag.PaginationLimitedTo.ValueInt64()))
		}
	}

	// Typos block
	if !model.Typos.IsNull() {
		var typos TyposModel
		diags.Append(model.Typos.As(ctx, &typos, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if !typos.TypoTolerance.IsNull() {
			settings.TypoTolerance = expandTypoTolerance(typos.TypoTolerance.ValueString())
		}
		if !typos.MinWordSizeFor1Typo.IsNull() {
			settings.MinWordSizefor1Typo = utils.ToPtr(int32(typos.MinWordSizeFor1Typo.ValueInt64()))
		}
		if !typos.MinWordSizeFor2Typos.IsNull() {
			settings.MinWordSizefor2Typos = utils.ToPtr(int32(typos.MinWordSizeFor2Typos.ValueInt64()))
		}
		if !typos.AllowTyposOnNumericTokens.IsNull() {
			settings.AllowTyposOnNumericTokens = utils.ToPtr(typos.AllowTyposOnNumericTokens.ValueBool())
		}
		if !typos.DisableTypoToleranceOnAttributes.IsNull() {
			settings.DisableTypoToleranceOnAttributes = expandStringList(ctx, typos.DisableTypoToleranceOnAttributes)
		}
		if !typos.DisableTypoToleranceOnWords.IsNull() {
			settings.DisableTypoToleranceOnWords = expandStringList(ctx, typos.DisableTypoToleranceOnWords)
		}
	}

	// Languages block
	if !model.Languages.IsNull() {
		var lang LanguagesModel
		diags.Append(model.Languages.As(ctx, &lang, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if !lang.IndexLanguages.IsNull() {
			settings.IndexLanguages = expandSupportedLanguageList(ctx, lang.IndexLanguages)
		}
		if !lang.QueryLanguages.IsNull() {
			settings.QueryLanguages = expandSupportedLanguageList(ctx, lang.QueryLanguages)
		}
		if !lang.IgnorePluralsLanguages.IsNull() {
			settings.IgnorePlurals = search.ArrayOfSupportedLanguageAsIgnorePlurals(expandSupportedLanguageList(ctx, lang.IgnorePluralsLanguages))
		} else if !lang.IgnorePlurals.IsNull() {
			settings.IgnorePlurals = search.BoolAsIgnorePlurals(lang.IgnorePlurals.ValueBool())
		}
		if !lang.RemoveStopWordsLanguages.IsNull() {
			settings.RemoveStopWords = search.ArrayOfSupportedLanguageAsRemoveStopWords(expandSupportedLanguageList(ctx, lang.RemoveStopWordsLanguages))
		} else if !lang.RemoveStopWords.IsNull() {
			settings.RemoveStopWords = search.BoolAsRemoveStopWords(lang.RemoveStopWords.ValueBool())
		}
		if !lang.DecompoundQuery.IsNull() {
			settings.DecompoundQuery = utils.ToPtr(lang.DecompoundQuery.ValueBool())
		}
		if !lang.RemoveWordsIfNoResults.IsNull() {
			v := search.RemoveWordsIfNoResults(lang.RemoveWordsIfNoResults.ValueString())
			settings.RemoveWordsIfNoResults = &v
		}
		if !lang.AttributesToTransliterate.IsNull() {
			settings.AttributesToTransliterate = expandStringList(ctx, lang.AttributesToTransliterate)
		}
		if !lang.CamelCaseAttributes.IsNull() {
			settings.CamelCaseAttributes = expandStringList(ctx, lang.CamelCaseAttributes)
		}
		if !lang.DecompoundedAttributes.IsNull() {
			var decompounded map[string]any
			if err := json.Unmarshal([]byte(lang.DecompoundedAttributes.ValueString()), &decompounded); err != nil {
				diags.AddError("Invalid decompounded_attributes", "Failed to parse JSON: "+err.Error())
				return nil, diags
			}
			settings.DecompoundedAttributes = decompounded
		}
		if !lang.CustomNormalization.IsNull() {
			var customNorm map[string]map[string]string
			if err := json.Unmarshal([]byte(lang.CustomNormalization.ValueString()), &customNorm); err != nil {
				diags.AddError("Invalid custom_normalization", "Failed to parse JSON: "+err.Error())
				return nil, diags
			}
			settings.CustomNormalization = &customNorm
		}
		if !lang.KeepDiacriticsOnCharacters.IsNull() {
			settings.KeepDiacriticsOnCharacters = utils.ToPtr(lang.KeepDiacriticsOnCharacters.ValueString())
		}
	}

	// Query Strategy block
	if !model.QueryStrategy.IsNull() {
		var qs QueryStrategyModel
		diags.Append(model.QueryStrategy.As(ctx, &qs, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if !qs.QueryType.IsNull() {
			v := search.QueryType(qs.QueryType.ValueString())
			settings.QueryType = &v
		}
		if !qs.AdvancedSyntax.IsNull() {
			settings.AdvancedSyntax = utils.ToPtr(qs.AdvancedSyntax.ValueBool())
		}
		if !qs.AdvancedSyntaxFeatures.IsNull() {
			settings.AdvancedSyntaxFeatures = expandAdvancedSyntaxFeaturesList(ctx, qs.AdvancedSyntaxFeatures)
		}
		if !qs.OptionalWords.IsNull() {
			words := expandStringList(ctx, qs.OptionalWords)
			optWords := search.ArrayOfStringAsOptionalWords(words)
			settings.OptionalWords = *utils.NewNullable(optWords)
		}
		if !qs.DisablePrefixOnAttributes.IsNull() {
			settings.DisablePrefixOnAttributes = expandStringList(ctx, qs.DisablePrefixOnAttributes)
		}
		if !qs.DisableExactOnAttributes.IsNull() {
			settings.DisableExactOnAttributes = expandStringList(ctx, qs.DisableExactOnAttributes)
		}
		if !qs.ExactOnSingleWordQuery.IsNull() {
			v := search.ExactOnSingleWordQuery(qs.ExactOnSingleWordQuery.ValueString())
			settings.ExactOnSingleWordQuery = &v
		}
		if !qs.AlternativesAsExact.IsNull() {
			settings.AlternativesAsExact = expandAlternativesAsExactList(ctx, qs.AlternativesAsExact)
		}
	}

	// Performance block
	if !model.Performance.IsNull() {
		var perf PerformanceModel
		diags.Append(model.Performance.As(ctx, &perf, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if !perf.NumericAttributesForFiltering.IsNull() {
			settings.NumericAttributesForFiltering = expandStringList(ctx, perf.NumericAttributesForFiltering)
		}
		if !perf.AllowCompressionOfIntegerArray.IsNull() {
			settings.AllowCompressionOfIntegerArray = utils.ToPtr(perf.AllowCompressionOfIntegerArray.ValueBool())
		}
	}

	// Advanced block
	if !model.Advanced.IsNull() {
		var adv AdvancedModel
		diags.Append(model.Advanced.As(ctx, &adv, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		if !adv.Distinct.IsNull() {
			settings.Distinct = search.Int32AsDistinct(int32(adv.Distinct.ValueInt64()))
		}
		if !adv.MinProximity.IsNull() {
			settings.MinProximity = utils.ToPtr(int32(adv.MinProximity.ValueInt64()))
		}
		if !adv.ReplaceSynonymsInHighlight.IsNull() {
			settings.ReplaceSynonymsInHighlight = utils.ToPtr(adv.ReplaceSynonymsInHighlight.ValueBool())
		}
		if !adv.SeparatorsToIndex.IsNull() {
			settings.SeparatorsToIndex = utils.ToPtr(adv.SeparatorsToIndex.ValueString())
		}
		if !adv.ResponseFields.IsNull() {
			settings.ResponseFields = expandStringList(ctx, adv.ResponseFields)
		}
		if !adv.UserData.IsNull() {
			var userData any
			if err := json.Unmarshal([]byte(adv.UserData.ValueString()), &userData); err != nil {
				diags.AddError("Invalid user_data", "Failed to parse JSON: "+err.Error())
				return nil, diags
			}
			settings.UserData = userData
		}
		if !adv.EnableRules.IsNull() {
			settings.EnableRules = utils.ToPtr(adv.EnableRules.ValueBool())
		}
		if !adv.EnablePersonalization.IsNull() {
			settings.EnablePersonalization = utils.ToPtr(adv.EnablePersonalization.ValueBool())
		}
		if !adv.Replicas.IsNull() {
			settings.Replicas = expandStringList(ctx, adv.Replicas)
		}
		if !adv.EnableReRanking.IsNull() {
			settings.EnableReRanking = utils.ToPtr(adv.EnableReRanking.ValueBool())
		}
		if !adv.ReRankingApplyFilter.IsNull() {
			var reRankingFilter search.ReRankingApplyFilter
			if err := json.Unmarshal([]byte(adv.ReRankingApplyFilter.ValueString()), &reRankingFilter); err != nil {
				diags.AddError("Invalid re_ranking_apply_filter", "Failed to parse JSON: "+err.Error())
				return nil, diags
			}
			settings.ReRankingApplyFilter = &reRankingFilter
		}
		if !adv.Mode.IsNull() {
			v := search.Mode(adv.Mode.ValueString())
			settings.Mode = &v
		}
		if !adv.SemanticSearch.IsNull() {
			var semanticSearch search.SemanticSearch
			if err := json.Unmarshal([]byte(adv.SemanticSearch.ValueString()), &semanticSearch); err != nil {
				diags.AddError("Invalid semantic_search", "Failed to parse JSON: "+err.Error())
				return nil, diags
			}
			settings.SemanticSearch = &semanticSearch
		}
		if !adv.AttributeCriteriaComputedByMinProximity.IsNull() {
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
