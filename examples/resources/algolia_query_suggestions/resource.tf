# Query Suggestions is region-routed. Set `analytics_region` on the provider
# (it must match the region your Algolia application is hosted in).
# provider "algolia" {
#   analytics_region = "us"
# }

resource "algolia_index" "source" {
  name                = "products"
  deletion_protection = false
}

# Builds a "products_query_suggestions" index from the search analytics of the
# source index, so you can power an as-you-type suggestions UI.
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
