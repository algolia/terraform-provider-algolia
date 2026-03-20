resource "algolia_index" "example" {
  name                = "my-index"       # Required. Changing this destroys and recreates.
  deletion_protection = true             # Defaults to true. Must set false before destroy.

  attributes {
    searchable_attributes    = ["name", "description"]
    attributes_to_retrieve   = ["name", "description", "price"]
    unretrievable_attributes = ["internal_id"]
    attribute_for_distinct   = "product_id"
  }

  ranking {
    ranking              = ["typo", "geo", "words", "filters", "proximity", "attribute", "exact", "custom"]
    custom_ranking       = ["desc(popularity)", "asc(price)"]
    relevancy_strictness = 90
  }

  faceting {
    attributes_for_faceting = ["searchable(category)", "brand", "price"]
    max_facet_hits          = 10
    max_values_per_facet    = 100
    sort_facet_values_by    = "count"  # "count" or "alpha"
  }

  highlighting {
    highlight_pre_tag                     = "<em>"
    highlight_post_tag                    = "</em>"
    attributes_to_highlight               = ["name", "description"]
    attributes_to_snippet                 = ["description:20"]
    snippet_ellipsis_text                 = "..."
    restrict_highlight_and_snippet_arrays = false
  }

  pagination {
    hits_per_page         = 20
    pagination_limited_to = 1000
  }

  typos {
    typo_tolerance                       = "true"  # "true", "false", "min", or "strict"
    min_word_size_for_1_typo             = 4
    min_word_size_for_2_typos            = 8
    allow_typos_on_numeric_tokens        = true
    disable_typo_tolerance_on_attributes = ["sku"]
    disable_typo_tolerance_on_words      = ["iphone"]
  }

  languages {
    index_languages               = ["en", "fr"]
    query_languages               = ["en"]
    ignore_plurals                = true          # bool - applies to all languages
    # ignore_plurals_languages    = ["en", "fr"]  # OR list specific languages (not both)
    remove_stop_words             = false
    # remove_stop_words_languages = ["en"]        # OR list specific languages (not both)
    decompound_query              = true
    remove_words_if_no_results    = "none"  # "none", "lastWords", "firstWords", "allOptional"
    camel_case_attributes         = ["productName"]
    keep_diacritics_on_characters = ""
    attributes_to_transliterate   = ["name", "description"]
  }

  query_strategy {
    query_type                  = "prefixLast"  # "prefixLast", "prefixAll", "prefixNone"
    advanced_syntax             = false
    advanced_syntax_features    = ["exactPhrase", "excludeWords"]
    optional_words              = ["the", "a"]
    disable_prefix_on_attributes = ["sku"]
    disable_exact_on_attributes = ["description"]
    exact_on_single_word_query  = "attribute"  # "attribute", "none", "word"
    alternatives_as_exact       = ["ignorePlurals", "singleWordSynonym"]
  }

  performance {
    numeric_attributes_for_filtering  = ["price", "quantity"]
    allow_compression_of_integer_array = false
  }

  advanced {
    distinct                                    = 1     # 0=off, 1=dedupe, 2+=group size
    min_proximity                               = 1
    replace_synonyms_in_highlight               = false
    separators_to_index                         = "+#"
    response_fields                             = ["hits", "nbHits"]
    enable_rules                                = true
    enable_personalization                       = false
    replicas                                    = ["products_price_asc"]
    enable_re_ranking                           = false
    mode                                        = "keywordSearch"  # or "neuralSearch"
    attribute_criteria_computed_by_min_proximity = false
    # JSON-encoded fields:
    user_data               = jsonencode({ version = "2.0" })
    semantic_search          = jsonencode({ eventSources = ["order"] })
    re_ranking_apply_filter = jsonencode("brand:Apple")
  }
}
