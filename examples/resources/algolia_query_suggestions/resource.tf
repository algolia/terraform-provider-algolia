provider "algolia" {
  query_suggestions_region = "us"
}

resource "algolia_index" "source" {
  name                = "products"
  deletion_protection = false
}

resource "algolia_query_suggestions" "example" {
  index_name = "products_query_suggestions"
  languages  = ["en"]

  source_indices {
    index_name     = algolia_index.source.name
    analytics_tags = ["mobile"]
    min_hits       = 10
    min_letters    = 3
  }
}
