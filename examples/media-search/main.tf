provider "algolia" {
  app_id  = var.algolia_app_id
  api_key = var.algolia_api_key
}

# The catalog index for a streaming service: movies and shows in one index.
# Records are ingested separately (via the API, an ingestion task, or a
# crawler) - Terraform owns the settings.
resource "algolia_index" "titles" {
  name                = "titles"
  deletion_protection = true

  attributes {
    searchable_attributes  = ["title", "cast", "genres", "synopsis"]
    attributes_to_retrieve = ["title", "release_year", "genres", "poster_url", "maturity_rating"]
    attribute_for_distinct = "series_id" # collapse a show's episodes into one hit
  }

  ranking {
    ranking        = ["typo", "geo", "words", "filters", "proximity", "attribute", "exact", "custom"]
    custom_ranking = ["desc(popularity)", "desc(release_year)"]
  }

  faceting {
    attributes_for_faceting = ["genres", "maturity_rating", "release_year", "searchable(cast)", "filterOnly(is_original)"]
    max_values_per_facet    = 100
    sort_facet_values_by    = "count"
  }

  pagination {
    hits_per_page = 24
  }

  typos {
    typo_tolerance = "true"
  }

  languages {
    query_languages = ["en"]
    ignore_plurals  = true
  }

  query_strategy {
    query_type = "prefixLast"
  }
}

# Merchandising: surface platform originals for browse-intent queries.
resource "algolia_rule" "boost_originals" {
  index_name  = algolia_index.titles.name
  object_id   = "boost-originals"
  description = "Prioritise platform originals for trending/browse queries"

  conditions {
    pattern   = "trending"
    anchoring = "contains"
  }

  consequence {
    params_json = jsonencode({
      optionalFilters = ["is_original:true"]
    })
  }
}

# Domain synonyms so viewers find titles regardless of how they spell the genre.
resource "algolia_synonym" "scifi" {
  index_name = algolia_index.titles.name
  object_id  = "scifi-science-fiction"
  type       = "synonym"
  synonyms   = ["scifi", "sci-fi", "science fiction"]
}

# A search-only key scoped to the titles index - safe to embed in web and TV
# apps (it cannot read or mutate anything else).
resource "algolia_api_key" "app_search" {
  description = "Public search-only key for streaming apps"
  acl         = ["search", "browse"]
  indexes     = [algolia_index.titles.name]

  max_hits_per_query          = 50
  max_queries_per_ip_per_hour = 100000
}
