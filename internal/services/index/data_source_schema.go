package index

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func indexDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read the settings of an existing Algolia index.",
		Attributes: map[string]datasourceschema.Attribute{
			"name": datasourceschema.StringAttribute{
				Description: "The name of the Algolia index.",
				Required:    true,
			},
			"primary": datasourceschema.StringAttribute{
				Description: "The name of the primary index (for replicas).",
				Computed:    true,
			},
			"entries": datasourceschema.Int64Attribute{
				Description: "The number of records in the index.",
				Computed:    true,
			},
			"data_size": datasourceschema.Int64Attribute{
				Description: "The size of the index data in bytes.",
				Computed:    true,
			},
			"created_at": datasourceschema.StringAttribute{
				Description: "The creation date of the index.",
				Computed:    true,
			},
			"updated_at": datasourceschema.StringAttribute{
				Description: "The last update date of the index.",
				Computed:    true,
			},
		},
		Blocks: map[string]datasourceschema.Block{
			"attributes": datasourceschema.SingleNestedBlock{
				Description: "Configuration for searchable and retrievable attributes.",
				Attributes:  attributesDataSourceBlockSchema(),
			},
			"ranking": datasourceschema.SingleNestedBlock{
				Description: "Configuration for ranking and custom ranking.",
				Attributes:  rankingDataSourceBlockSchema(),
			},
			"faceting": datasourceschema.SingleNestedBlock{
				Description: "Configuration for faceting behavior.",
				Attributes:  facetingDataSourceBlockSchema(),
			},
			"highlighting": datasourceschema.SingleNestedBlock{
				Description: "Configuration for highlighting and snippeting.",
				Attributes:  highlightingDataSourceBlockSchema(),
			},
			"pagination": datasourceschema.SingleNestedBlock{
				Description: "Configuration for pagination.",
				Attributes:  paginationDataSourceBlockSchema(),
			},
			"typos": datasourceschema.SingleNestedBlock{
				Description: "Configuration for typo tolerance.",
				Attributes:  typosDataSourceBlockSchema(),
			},
			"languages": datasourceschema.SingleNestedBlock{
				Description: "Configuration for language-specific settings.",
				Attributes:  languagesDataSourceBlockSchema(),
			},
			"query_strategy": datasourceschema.SingleNestedBlock{
				Description: "Configuration for query strategy.",
				Attributes:  queryStrategyDataSourceBlockSchema(),
			},
			"performance": datasourceschema.SingleNestedBlock{
				Description: "Configuration for performance settings.",
				Attributes:  performanceDataSourceBlockSchema(),
			},
			"advanced": datasourceschema.SingleNestedBlock{
				Description: "Configuration for advanced settings.",
				Attributes:  advancedDataSourceBlockSchema(),
			},
		},
	}
}

func attributesDataSourceBlockSchema() map[string]datasourceschema.Attribute {
	return map[string]datasourceschema.Attribute{
		"searchable_attributes": datasourceschema.ListAttribute{
			Description: "The complete list of attributes used for searching.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"attributes_to_retrieve": datasourceschema.ListAttribute{
			Description: "The complete list of attributes that will be returned in search results.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"unretrievable_attributes": datasourceschema.ListAttribute{
			Description: "List of attributes that cannot be retrieved at query time.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"attribute_for_distinct": datasourceschema.StringAttribute{
			Description: "The name of the attribute used for deduplication with distinct.",
			Computed:    true,
		},
	}
}

func rankingDataSourceBlockSchema() map[string]datasourceschema.Attribute {
	return map[string]datasourceschema.Attribute{
		"ranking": datasourceschema.ListAttribute{
			Description: "The ranking criteria.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"custom_ranking": datasourceschema.ListAttribute{
			Description: "The custom ranking criteria.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"relevancy_strictness": datasourceschema.Int64Attribute{
			Description: "The relevancy strictness for ranking.",
			Computed:    true,
		},
	}
}

func facetingDataSourceBlockSchema() map[string]datasourceschema.Attribute {
	return map[string]datasourceschema.Attribute{
		"attributes_for_faceting": datasourceschema.ListAttribute{
			Description: "The complete list of attributes that will be used for faceting.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"max_facet_hits": datasourceschema.Int64Attribute{
			Description: "Maximum number of facet hits to return during a search for facet values.",
			Computed:    true,
		},
		"max_values_per_facet": datasourceschema.Int64Attribute{
			Description: "Maximum number of facet values to return for each facet.",
			Computed:    true,
		},
		"sort_facet_values_by": datasourceschema.StringAttribute{
			Description: "How to sort facet values.",
			Computed:    true,
		},
	}
}

func highlightingDataSourceBlockSchema() map[string]datasourceschema.Attribute {
	return map[string]datasourceschema.Attribute{
		"attributes_to_highlight": datasourceschema.ListAttribute{
			Description: "List of attributes to highlight.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"attributes_to_snippet": datasourceschema.ListAttribute{
			Description: "List of attributes to snippet.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"highlight_pre_tag": datasourceschema.StringAttribute{
			Description: "The HTML string to insert before a highlighted part.",
			Computed:    true,
		},
		"highlight_post_tag": datasourceschema.StringAttribute{
			Description: "The HTML string to insert after a highlighted part.",
			Computed:    true,
		},
		"snippet_ellipsis_text": datasourceschema.StringAttribute{
			Description: "String used as an ellipsis indicator when a snippet is truncated.",
			Computed:    true,
		},
		"restrict_highlight_and_snippet_arrays": datasourceschema.BoolAttribute{
			Description: "Whether to restrict highlighting and snippeting to items that matched the query.",
			Computed:    true,
		},
	}
}

func paginationDataSourceBlockSchema() map[string]datasourceschema.Attribute {
	return map[string]datasourceschema.Attribute{
		"hits_per_page": datasourceschema.Int64Attribute{
			Description: "The number of hits per page.",
			Computed:    true,
		},
		"pagination_limited_to": datasourceschema.Int64Attribute{
			Description: "The maximum number of hits accessible via pagination.",
			Computed:    true,
		},
	}
}

func typosDataSourceBlockSchema() map[string]datasourceschema.Attribute {
	return map[string]datasourceschema.Attribute{
		"typo_tolerance": datasourceschema.StringAttribute{
			Description: "Whether typo tolerance is enabled and how it is applied.",
			Computed:    true,
		},
		"min_word_size_for_1_typo": datasourceschema.Int64Attribute{
			Description: "Minimum word size for one typo.",
			Computed:    true,
		},
		"min_word_size_for_2_typos": datasourceschema.Int64Attribute{
			Description: "Minimum word size for two typos.",
			Computed:    true,
		},
		"allow_typos_on_numeric_tokens": datasourceschema.BoolAttribute{
			Description: "Whether to allow typos on numbers.",
			Computed:    true,
		},
		"disable_typo_tolerance_on_attributes": datasourceschema.ListAttribute{
			Description: "List of attributes on which typo tolerance is disabled.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"disable_typo_tolerance_on_words": datasourceschema.ListAttribute{
			Description: "List of words on which typo tolerance is disabled.",
			Computed:    true,
			ElementType: types.StringType,
		},
	}
}

func languagesDataSourceBlockSchema() map[string]datasourceschema.Attribute {
	return map[string]datasourceschema.Attribute{
		"index_languages": datasourceschema.ListAttribute{
			Description: "List of languages for language-specific processing steps.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"query_languages": datasourceschema.ListAttribute{
			Description: "List of languages for query processing.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"ignore_plurals": datasourceschema.BoolAttribute{
			Description: "Whether to treat singular and plurals as matching terms.",
			Computed:    true,
		},
		"ignore_plurals_languages": datasourceschema.ListAttribute{
			Description: "List of languages for which ignore_plurals is enabled.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"remove_stop_words": datasourceschema.BoolAttribute{
			Description: "Whether to remove stop words.",
			Computed:    true,
		},
		"remove_stop_words_languages": datasourceschema.ListAttribute{
			Description: "List of languages for which remove_stop_words is enabled.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"decompound_query": datasourceschema.BoolAttribute{
			Description: "Whether to split compound words.",
			Computed:    true,
		},
		"remove_words_if_no_results": datasourceschema.StringAttribute{
			Description: "Strategy to remove words when query doesn't match.",
			Computed:    true,
		},
		"attributes_to_transliterate": datasourceschema.ListAttribute{
			Description: "List of attributes to apply transliteration.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"camel_case_attributes": datasourceschema.ListAttribute{
			Description: "List of attributes for camel case decomposition.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"decompounded_attributes": datasourceschema.StringAttribute{
			Description: "Decompounding attributes as a JSON-encoded object.",
			Computed:    true,
		},
		"custom_normalization": datasourceschema.StringAttribute{
			Description: "Custom normalization rules as a JSON-encoded object.",
			Computed:    true,
		},
		"keep_diacritics_on_characters": datasourceschema.StringAttribute{
			Description: "Characters for which diacritics should be preserved.",
			Computed:    true,
		},
	}
}

func queryStrategyDataSourceBlockSchema() map[string]datasourceschema.Attribute {
	return map[string]datasourceschema.Attribute{
		"query_type": datasourceschema.StringAttribute{
			Description: "Determines how query words are matched.",
			Computed:    true,
		},
		"advanced_syntax": datasourceschema.BoolAttribute{
			Description: "Whether the advanced query syntax is enabled.",
			Computed:    true,
		},
		"advanced_syntax_features": datasourceschema.ListAttribute{
			Description: "Advanced syntax features to be activated.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"optional_words": datasourceschema.ListAttribute{
			Description: "Words which should be considered optional.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"disable_prefix_on_attributes": datasourceschema.ListAttribute{
			Description: "Attributes on which prefix matching is disabled.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"disable_exact_on_attributes": datasourceschema.ListAttribute{
			Description: "Attributes on which exact filtering is disabled.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"exact_on_single_word_query": datasourceschema.StringAttribute{
			Description: "How exact ranking is computed for single word queries.",
			Computed:    true,
		},
		"alternatives_as_exact": datasourceschema.ListAttribute{
			Description: "Alternatives considered as exact match.",
			Computed:    true,
			ElementType: types.StringType,
		},
	}
}

func performanceDataSourceBlockSchema() map[string]datasourceschema.Attribute {
	return map[string]datasourceschema.Attribute{
		"numeric_attributes_for_filtering": datasourceschema.ListAttribute{
			Description: "List of numeric attributes for filtering.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"allow_compression_of_integer_array": datasourceschema.BoolAttribute{
			Description: "Whether integer array compression is enabled.",
			Computed:    true,
		},
	}
}

func advancedDataSourceBlockSchema() map[string]datasourceschema.Attribute {
	return map[string]datasourceschema.Attribute{
		"distinct": datasourceschema.Int64Attribute{
			Description: "De-duplication or grouping of results.",
			Computed:    true,
		},
		"min_proximity": datasourceschema.Int64Attribute{
			Description: "Precision of the proximity ranking criterion.",
			Computed:    true,
		},
		"replace_synonyms_in_highlight": datasourceschema.BoolAttribute{
			Description: "Whether to highlight the original word or the synonym.",
			Computed:    true,
		},
		"separators_to_index": datasourceschema.StringAttribute{
			Description: "Separators to index.",
			Computed:    true,
		},
		"response_fields": datasourceschema.ListAttribute{
			Description: "Properties to include in the API response.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"user_data": datasourceschema.StringAttribute{
			Description: "Custom user data as a JSON-encoded string.",
			Computed:    true,
		},
		"enable_rules": datasourceschema.BoolAttribute{
			Description: "Whether Rules are enabled.",
			Computed:    true,
		},
		"enable_personalization": datasourceschema.BoolAttribute{
			Description: "Whether Personalization is enabled.",
			Computed:    true,
		},
		"replicas": datasourceschema.ListAttribute{
			Description: "List of replica index names.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"enable_re_ranking": datasourceschema.BoolAttribute{
			Description: "Whether AI Re-Ranking is enabled.",
			Computed:    true,
		},
		"re_ranking_apply_filter": datasourceschema.StringAttribute{
			Description: "Filter for AI Re-Ranking as a JSON-encoded string.",
			Computed:    true,
		},
		"mode": datasourceschema.StringAttribute{
			Description: "The search mode.",
			Computed:    true,
		},
		"semantic_search": datasourceschema.StringAttribute{
			Description: "Semantic search settings as a JSON-encoded string.",
			Computed:    true,
		},
		"attribute_criteria_computed_by_min_proximity": datasourceschema.BoolAttribute{
			Description: "Whether the best matching attribute is determined by minimum proximity.",
			Computed:    true,
		},
	}
}
