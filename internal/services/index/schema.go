package index

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func indexResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages an Algolia index and its settings.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the Algolia index.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"deletion_protection": schema.BoolAttribute{
				Description: "Whether to prevent deletion of the index. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"primary": schema.StringAttribute{
				Description: "The name of the primary index (for replicas).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					useStateForKnownString(),
				},
			},
			"entries": schema.Int64Attribute{
				Description: "The number of records in the index.",
				Computed:    true,
			},
			"data_size": schema.Int64Attribute{
				Description: "The size of the index data in bytes.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation date of the index.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					useStateForKnownString(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The last update date of the index.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"attributes": schema.SingleNestedBlock{
				Description: "Configuration for searchable and retrievable attributes.",
				Attributes:  attributesBlockSchema(),
			},
			"ranking": schema.SingleNestedBlock{
				Description: "Configuration for ranking and custom ranking.",
				Attributes:  rankingBlockSchema(),
			},
			"faceting": schema.SingleNestedBlock{
				Description: "Configuration for faceting behavior.",
				Attributes:  facetingBlockSchema(),
			},
			"highlighting": schema.SingleNestedBlock{
				Description: "Configuration for highlighting and snippeting.",
				Attributes:  highlightingBlockSchema(),
			},
			"pagination": schema.SingleNestedBlock{
				Description: "Configuration for pagination.",
				Attributes:  paginationBlockSchema(),
			},
			"typos": schema.SingleNestedBlock{
				Description: "Configuration for typo tolerance.",
				Attributes:  typosBlockSchema(),
			},
			"languages": schema.SingleNestedBlock{
				Description: "Configuration for language-specific settings.",
				Attributes:  languagesBlockSchema(),
			},
			"query_strategy": schema.SingleNestedBlock{
				Description: "Configuration for query strategy.",
				Attributes:  queryStrategyBlockSchema(),
			},
			"performance": schema.SingleNestedBlock{
				Description: "Configuration for performance settings.",
				Attributes:  performanceBlockSchema(),
			},
			"advanced": schema.SingleNestedBlock{
				Description: "Configuration for advanced settings.",
				Attributes:  advancedBlockSchema(),
			},
		},
	}
}

func attributesBlockSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"searchable_attributes": schema.ListAttribute{
			Description: "The complete list of attributes used for searching.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"attributes_to_retrieve": schema.ListAttribute{
			Description: "The complete list of attributes that will be returned in search results.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"unretrievable_attributes": schema.ListAttribute{
			Description: "List of attributes that cannot be retrieved at query time.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"attribute_for_distinct": schema.StringAttribute{
			Description: "The name of the attribute used for deduplication with distinct.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				useStateForKnownString(),
			},
		},
	}
}

func rankingBlockSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ranking": schema.ListAttribute{
			Description: "The ranking criteria.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"custom_ranking": schema.ListAttribute{
			Description: "The custom ranking criteria.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"relevancy_strictness": schema.Int64Attribute{
			Description: "The relevancy strictness for ranking.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Int64{
				useStateForKnownInt64(),
			},
		},
	}
}

func facetingBlockSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"attributes_for_faceting": schema.ListAttribute{
			Description: "The complete list of attributes that will be used for faceting.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"max_facet_hits": schema.Int64Attribute{
			Description: "Maximum number of facet hits to return during a search for facet values.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Int64{
				useStateForKnownInt64(),
			},
		},
		"max_values_per_facet": schema.Int64Attribute{
			Description: "Maximum number of facet values to return for each facet.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Int64{
				useStateForKnownInt64(),
			},
		},
		"sort_facet_values_by": schema.StringAttribute{
			Description: "How to sort facet values. One of: count, alpha.",
			Optional:    true,
			Computed:    true,
			Validators: []validator.String{
				stringvalidator.OneOf("count", "alpha"),
			},
			PlanModifiers: []planmodifier.String{
				useStateForKnownString(),
			},
		},
	}
}

func highlightingBlockSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"attributes_to_highlight": schema.ListAttribute{
			Description: "List of attributes to highlight.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"attributes_to_snippet": schema.ListAttribute{
			Description: "List of attributes to snippet, with an optional maximum number of words to snippet.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"highlight_pre_tag": schema.StringAttribute{
			Description: "The HTML string to insert before a highlighted part in all highlight and snippet results.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				useStateForKnownString(),
			},
		},
		"highlight_post_tag": schema.StringAttribute{
			Description: "The HTML string to insert after a highlighted part in all highlight and snippet results.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				useStateForKnownString(),
			},
		},
		"snippet_ellipsis_text": schema.StringAttribute{
			Description: "String used as an ellipsis indicator when a snippet is truncated.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				useStateForKnownString(),
			},
		},
		"restrict_highlight_and_snippet_arrays": schema.BoolAttribute{
			Description: "Whether to restrict highlighting and snippeting to items that matched the query.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Bool{
				useStateForKnownBool(),
			},
		},
	}
}

func paginationBlockSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"hits_per_page": schema.Int64Attribute{
			Description: "The number of hits per page.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Int64{
				useStateForKnownInt64(),
			},
		},
		"pagination_limited_to": schema.Int64Attribute{
			Description: "The maximum number of hits accessible via pagination.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Int64{
				useStateForKnownInt64(),
			},
		},
	}
}

func typosBlockSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"typo_tolerance": schema.StringAttribute{
			Description: "Whether typo tolerance is enabled and how it is applied. One of: true, false, min, strict.",
			Optional:    true,
			Computed:    true,
			Validators: []validator.String{
				stringvalidator.OneOf("true", "false", "min", "strict"),
			},
			PlanModifiers: []planmodifier.String{
				useStateForKnownString(),
			},
		},
		"min_word_size_for_1_typo": schema.Int64Attribute{
			Description: "Minimum number of characters a word in the query string must contain to accept matches with one typo.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Int64{
				useStateForKnownInt64(),
			},
		},
		"min_word_size_for_2_typos": schema.Int64Attribute{
			Description: "Minimum number of characters a word in the query string must contain to accept matches with two typos.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Int64{
				useStateForKnownInt64(),
			},
		},
		"allow_typos_on_numeric_tokens": schema.BoolAttribute{
			Description: "Whether to allow typos on numbers in the query string.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Bool{
				useStateForKnownBool(),
			},
		},
		"disable_typo_tolerance_on_attributes": schema.ListAttribute{
			Description: "List of attributes on which typo tolerance is disabled.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"disable_typo_tolerance_on_words": schema.ListAttribute{
			Description: "List of words on which typo tolerance is disabled.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
	}
}

func languagesBlockSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"index_languages": schema.ListAttribute{
			Description: "List of languages for language-specific processing steps (plurals, stop-words, etc.).",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"query_languages": schema.ListAttribute{
			Description: "List of languages to be used by language-specific query processing steps.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"ignore_plurals": schema.BoolAttribute{
			Description: "Whether to treat singular, plurals, and other forms of declensions as matching terms.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Bool{
				useStateForKnownBool(),
			},
		},
		"ignore_plurals_languages": schema.ListAttribute{
			Description: "List of specific languages for which ignore_plurals is enabled.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"remove_stop_words": schema.BoolAttribute{
			Description: "Whether to remove stop words from the query before executing it.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Bool{
				useStateForKnownBool(),
			},
		},
		"remove_stop_words_languages": schema.ListAttribute{
			Description: "List of specific languages for which remove_stop_words is enabled.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"decompound_query": schema.BoolAttribute{
			Description: "Whether to split compound words into their component word parts.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Bool{
				useStateForKnownBool(),
			},
		},
		"remove_words_if_no_results": schema.StringAttribute{
			Description: "Strategy to remove words from the query when it doesn't match any results. One of: none, lastWords, firstWords, allOptional.",
			Optional:    true,
			Computed:    true,
			Validators: []validator.String{
				stringvalidator.OneOf("none", "lastWords", "firstWords", "allOptional"),
			},
			PlanModifiers: []planmodifier.String{
				useStateForKnownString(),
			},
		},
		"attributes_to_transliterate": schema.ListAttribute{
			Description: "List of attributes to apply transliteration.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"camel_case_attributes": schema.ListAttribute{
			Description: "List of attributes on which to do a decomposition of camel case words.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"decompounded_attributes": schema.StringAttribute{
			Description: "Attributes for which to enable decompounding, as a JSON-encoded object.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				useStateForKnownString(),
			},
		},
		"custom_normalization": schema.StringAttribute{
			Description: "Custom normalization rules, as a JSON-encoded object.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				useStateForKnownString(),
			},
		},
		"keep_diacritics_on_characters": schema.StringAttribute{
			Description: "Characters for which diacritics should be preserved.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				useStateForKnownString(),
			},
		},
	}
}

func queryStrategyBlockSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"query_type": schema.StringAttribute{
			Description: "Determines how query words are matched. One of: prefixLast, prefixAll, prefixNone.",
			Optional:    true,
			Computed:    true,
			Validators: []validator.String{
				stringvalidator.OneOf("prefixLast", "prefixAll", "prefixNone"),
			},
			PlanModifiers: []planmodifier.String{
				useStateForKnownString(),
			},
		},
		"advanced_syntax": schema.BoolAttribute{
			Description: "Whether to enable the advanced query syntax.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Bool{
				useStateForKnownBool(),
			},
		},
		"advanced_syntax_features": schema.ListAttribute{
			Description: "Advanced syntax features to be activated when advanced_syntax is enabled.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"optional_words": schema.ListAttribute{
			Description: "Words which should be considered optional when found in a query.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"disable_prefix_on_attributes": schema.ListAttribute{
			Description: "List of attributes on which prefix matching is disabled.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"disable_exact_on_attributes": schema.ListAttribute{
			Description: "List of attributes on which exact filtering is disabled.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"exact_on_single_word_query": schema.StringAttribute{
			Description: "Determines how the exact ranking criterion is computed when the query contains only one word. One of: attribute, none, word.",
			Optional:    true,
			Computed:    true,
			Validators: []validator.String{
				stringvalidator.OneOf("attribute", "none", "word"),
			},
			PlanModifiers: []planmodifier.String{
				useStateForKnownString(),
			},
		},
		"alternatives_as_exact": schema.ListAttribute{
			Description: "List of alternatives that should be considered an exact match by the exact ranking criterion.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
	}
}

func performanceBlockSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"numeric_attributes_for_filtering": schema.ListAttribute{
			Description: "List of numeric attributes that can be used as numerical filters.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"allow_compression_of_integer_array": schema.BoolAttribute{
			Description: "Whether to enable compression of integer arrays.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Bool{
				useStateForKnownBool(),
			},
		},
	}
}

func advancedBlockSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"distinct": schema.Int64Attribute{
			Description: "Enables de-duplication or grouping of results. 0 = disabled, 1 = single result per value, 2+ = group size.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Int64{
				useStateForKnownInt64(),
			},
		},
		"min_proximity": schema.Int64Attribute{
			Description: "Precision of the proximity ranking criterion.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Int64{
				useStateForKnownInt64(),
			},
		},
		"replace_synonyms_in_highlight": schema.BoolAttribute{
			Description: "Whether to highlight and snippet the original word that matches the synonym or the synonym itself.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Bool{
				useStateForKnownBool(),
			},
		},
		"separators_to_index": schema.StringAttribute{
			Description: "Separators to index as part of the record.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				useStateForKnownString(),
			},
		},
		"response_fields": schema.ListAttribute{
			Description: "Properties to include in the API response of search and browse requests.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"user_data": schema.StringAttribute{
			Description: "Custom user data, as a JSON-encoded string.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				useStateForKnownString(),
			},
		},
		"enable_rules": schema.BoolAttribute{
			Description: "Whether Rules should be globally enabled.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Bool{
				useStateForKnownBool(),
			},
		},
		"enable_personalization": schema.BoolAttribute{
			Description: "Whether to enable Personalization.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Bool{
				useStateForKnownBool(),
			},
		},
		"replicas": schema.ListAttribute{
			Description: "List of replica index names. Setting this declares the index's complete replica " +
				"list: any replica Algolia currently reports but this list omits is unlinked. Virtual " +
				"replicas appear here in their `virtual(<name>)` form, so if you also manage them with " +
				"`algolia_virtual_index` resources - which add themselves to this same setting - list " +
				"them here too. Applying a list that omits one unlinks it, and an unlinked virtual " +
				"replica is empty, since it is a view over this index's records rather than a copy; the " +
				"provider warns when a write would do that. Leave this unset to keep whatever replicas " +
				"the index already has.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				useStateForKnownList(),
			},
		},
		"enable_re_ranking": schema.BoolAttribute{
			Description: "Whether to enable AI Re-Ranking.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Bool{
				useStateForKnownBool(),
			},
		},
		"re_ranking_apply_filter": schema.StringAttribute{
			Description: "Filter to apply for AI Re-Ranking, as a JSON-encoded string.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				useStateForKnownString(),
			},
		},
		"mode": schema.StringAttribute{
			Description: "The search mode. One of: neuralSearch, keywordSearch.",
			Optional:    true,
			Computed:    true,
			Validators: []validator.String{
				stringvalidator.OneOf("neuralSearch", "keywordSearch"),
			},
			PlanModifiers: []planmodifier.String{
				useStateForKnownString(),
			},
		},
		"semantic_search": schema.StringAttribute{
			Description: "Semantic search settings, as a JSON-encoded string.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				useStateForKnownString(),
			},
		},
		"attribute_criteria_computed_by_min_proximity": schema.BoolAttribute{
			Description: "Whether the best matching attribute is determined by minimum proximity.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Bool{
				useStateForKnownBool(),
			},
		},
	}
}
