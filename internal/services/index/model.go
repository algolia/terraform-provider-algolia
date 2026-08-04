package index

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// IndexResourceModel describes the resource data model.
type IndexResourceModel struct {
	Name               types.String `tfsdk:"name"`
	DeletionProtection types.Bool   `tfsdk:"deletion_protection"`
	Primary            types.String `tfsdk:"primary"`
	Entries            types.Int64  `tfsdk:"entries"`
	DataSize           types.Int64  `tfsdk:"data_size"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
	Attributes         types.Object `tfsdk:"attributes"`
	Ranking            types.Object `tfsdk:"ranking"`
	Faceting           types.Object `tfsdk:"faceting"`
	Highlighting       types.Object `tfsdk:"highlighting"`
	Pagination         types.Object `tfsdk:"pagination"`
	Typos              types.Object `tfsdk:"typos"`
	Languages          types.Object `tfsdk:"languages"`
	QueryStrategy      types.Object `tfsdk:"query_strategy"`
	Performance        types.Object `tfsdk:"performance"`
	Advanced           types.Object `tfsdk:"advanced"`
}

// AttributesModel describes the attributes block.
type AttributesModel struct {
	SearchableAttributes    types.List   `tfsdk:"searchable_attributes"`
	AttributesToRetrieve    types.List   `tfsdk:"attributes_to_retrieve"`
	UnretrievableAttributes types.List   `tfsdk:"unretrievable_attributes"`
	AttributeForDistinct    types.String `tfsdk:"attribute_for_distinct"`
}

// RankingModel describes the ranking block.
type RankingModel struct {
	Ranking             types.List  `tfsdk:"ranking"`
	CustomRanking       types.List  `tfsdk:"custom_ranking"`
	RelevancyStrictness types.Int64 `tfsdk:"relevancy_strictness"`
}

// FacetingModel describes the faceting block.
type FacetingModel struct {
	AttributesForFaceting types.List   `tfsdk:"attributes_for_faceting"`
	MaxFacetHits          types.Int64  `tfsdk:"max_facet_hits"`
	MaxValuesPerFacet     types.Int64  `tfsdk:"max_values_per_facet"`
	SortFacetValuesBy     types.String `tfsdk:"sort_facet_values_by"`
}

// HighlightingModel describes the highlighting block.
type HighlightingModel struct {
	AttributesToHighlight             types.List   `tfsdk:"attributes_to_highlight"`
	AttributesToSnippet               types.List   `tfsdk:"attributes_to_snippet"`
	HighlightPreTag                   types.String `tfsdk:"highlight_pre_tag"`
	HighlightPostTag                  types.String `tfsdk:"highlight_post_tag"`
	SnippetEllipsisText               types.String `tfsdk:"snippet_ellipsis_text"`
	RestrictHighlightAndSnippetArrays types.Bool   `tfsdk:"restrict_highlight_and_snippet_arrays"`
}

// PaginationModel describes the pagination block.
type PaginationModel struct {
	HitsPerPage         types.Int64 `tfsdk:"hits_per_page"`
	PaginationLimitedTo types.Int64 `tfsdk:"pagination_limited_to"`
}

// TyposModel describes the typos block.
type TyposModel struct {
	TypoTolerance                    types.String `tfsdk:"typo_tolerance"`
	MinWordSizeFor1Typo              types.Int64  `tfsdk:"min_word_size_for_1_typo"`
	MinWordSizeFor2Typos             types.Int64  `tfsdk:"min_word_size_for_2_typos"`
	AllowTyposOnNumericTokens        types.Bool   `tfsdk:"allow_typos_on_numeric_tokens"`
	DisableTypoToleranceOnAttributes types.List   `tfsdk:"disable_typo_tolerance_on_attributes"`
	DisableTypoToleranceOnWords      types.List   `tfsdk:"disable_typo_tolerance_on_words"`
}

// LanguagesModel describes the languages block.
type LanguagesModel struct {
	IndexLanguages             types.List   `tfsdk:"index_languages"`
	QueryLanguages             types.List   `tfsdk:"query_languages"`
	IgnorePlurals              types.Bool   `tfsdk:"ignore_plurals"`
	IgnorePluralsLanguages     types.List   `tfsdk:"ignore_plurals_languages"`
	RemoveStopWords            types.Bool   `tfsdk:"remove_stop_words"`
	RemoveStopWordsLanguages   types.List   `tfsdk:"remove_stop_words_languages"`
	DecompoundQuery            types.Bool   `tfsdk:"decompound_query"`
	RemoveWordsIfNoResults     types.String `tfsdk:"remove_words_if_no_results"`
	AttributesToTransliterate  types.List   `tfsdk:"attributes_to_transliterate"`
	CamelCaseAttributes        types.List   `tfsdk:"camel_case_attributes"`
	DecompoundedAttributes     types.String `tfsdk:"decompounded_attributes"`
	CustomNormalization        types.String `tfsdk:"custom_normalization"`
	KeepDiacriticsOnCharacters types.String `tfsdk:"keep_diacritics_on_characters"`
}

// QueryStrategyModel describes the query_strategy block.
type QueryStrategyModel struct {
	QueryType                 types.String `tfsdk:"query_type"`
	AdvancedSyntax            types.Bool   `tfsdk:"advanced_syntax"`
	AdvancedSyntaxFeatures    types.List   `tfsdk:"advanced_syntax_features"`
	OptionalWords             types.List   `tfsdk:"optional_words"`
	DisablePrefixOnAttributes types.List   `tfsdk:"disable_prefix_on_attributes"`
	DisableExactOnAttributes  types.List   `tfsdk:"disable_exact_on_attributes"`
	ExactOnSingleWordQuery    types.String `tfsdk:"exact_on_single_word_query"`
	AlternativesAsExact       types.List   `tfsdk:"alternatives_as_exact"`
}

// PerformanceModel describes the performance block.
type PerformanceModel struct {
	NumericAttributesForFiltering  types.List `tfsdk:"numeric_attributes_for_filtering"`
	AllowCompressionOfIntegerArray types.Bool `tfsdk:"allow_compression_of_integer_array"`
}

// AdvancedModel describes the advanced block.
type AdvancedModel struct {
	Distinct                                types.Int64  `tfsdk:"distinct"`
	MinProximity                            types.Int64  `tfsdk:"min_proximity"`
	ReplaceSynonymsInHighlight              types.Bool   `tfsdk:"replace_synonyms_in_highlight"`
	SeparatorsToIndex                       types.String `tfsdk:"separators_to_index"`
	ResponseFields                          types.List   `tfsdk:"response_fields"`
	UserData                                types.String `tfsdk:"user_data"`
	RenderingContent                        types.String `tfsdk:"rendering_content"`
	EnableRules                             types.Bool   `tfsdk:"enable_rules"`
	EnablePersonalization                   types.Bool   `tfsdk:"enable_personalization"`
	Replicas                                types.List   `tfsdk:"replicas"`
	EnableReRanking                         types.Bool   `tfsdk:"enable_re_ranking"`
	ReRankingApplyFilter                    types.String `tfsdk:"re_ranking_apply_filter"`
	Mode                                    types.String `tfsdk:"mode"`
	SemanticSearch                          types.String `tfsdk:"semantic_search"`
	AttributeCriteriaComputedByMinProximity types.Bool   `tfsdk:"attribute_criteria_computed_by_min_proximity"`
}
