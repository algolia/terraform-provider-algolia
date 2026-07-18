provider "algolia" {
  app_id  = var.algolia_app_id
  api_key = var.algolia_api_key
}

# The primary catalog index. Records are ingested separately (via the API, an
# ingestion task, or a crawler) - Terraform owns the settings.
resource "algolia_index" "products" {
  name                = "products"
  deletion_protection = true

  attributes {
    searchable_attributes  = ["name", "brand", "categories", "description"]
    attributes_to_retrieve = ["name", "brand", "categories", "price", "image_url"]
    attribute_for_distinct = "product_id"
  }

  ranking {
    ranking        = ["typo", "geo", "words", "filters", "proximity", "attribute", "exact", "custom"]
    custom_ranking = ["desc(popularity)", "asc(price)"]
  }

  faceting {
    attributes_for_faceting = ["searchable(brand)", "categories", "price", "filterOnly(on_sale)"]
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

# Merchandising: when a query expresses sale intent, surface on-sale products.
resource "algolia_rule" "boost_on_sale" {
  index_name  = algolia_index.products.name
  object_id   = "boost-on-sale"
  description = "Prioritise on-sale products for sale-intent queries"

  conditions {
    pattern   = "sale"
    anchoring = "contains"
  }

  consequence {
    params_json = jsonencode({
      optionalFilters = ["on_sale:true"]
    })
  }
}

# Domain synonyms so shoppers find products regardless of the term they use.
resource "algolia_synonym" "tv" {
  index_name = algolia_index.products.name
  object_id  = "tv-television"
  type       = "synonym"
  synonyms   = ["tv", "television", "telly"]
}

# A search-only key scoped to the products index - safe to embed in browser
# and mobile clients (it cannot read or mutate anything else).
resource "algolia_api_key" "frontend_search" {
  description = "Public search-only key for storefront clients"
  acl         = ["search", "browse"]
  indexes     = [algolia_index.products.name]

  max_hits_per_query          = 50
  max_queries_per_ip_per_hour = 100000
}
