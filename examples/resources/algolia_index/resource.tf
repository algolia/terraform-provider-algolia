resource "algolia_index" "example" {
  name                = "example_index"
  deletion_protection = true

  attributes {
    searchable_attributes    = ["name", "description", "categories"]
    attributes_to_retrieve   = ["name", "description", "price", "image_url"]
    unretrievable_attributes = ["internal_id"]
    attribute_for_distinct   = "product_id"
  }

  ranking {
    ranking        = ["typo", "geo", "words", "filters", "proximity", "attribute", "exact", "custom"]
    custom_ranking = ["desc(popularity)", "asc(price)"]
  }

  faceting {
    attributes_for_faceting = ["searchable(categories)", "brand", "price"]
    max_values_per_facet    = 100
    sort_facet_values_by    = "count"
  }

  highlighting {
    highlight_pre_tag  = "<em>"
    highlight_post_tag = "</em>"
  }

  pagination {
    hits_per_page = 20
  }

  typos {
    typo_tolerance            = "true"
    min_word_size_for_1_typo  = 4
    min_word_size_for_2_typos = 8
  }

  languages {
    query_languages = ["en"]
    ignore_plurals  = true
  }

  query_strategy {
    query_type = "prefixLast"
  }

  advanced {
    distinct = 1
  }
}
