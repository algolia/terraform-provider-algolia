terraform {
  required_providers {
    algolia = {
      source = "registry.terraform.io/algolia/algolia"
    }
  }
}

provider "algolia" {
  # The Ingestion API is region-routed: analytics_region (or the
  # ALGOLIA_ANALYTICS_REGION environment variable) must be set.
  analytics_region = "us"
}

# A "search" destination writes records to an Algolia index. Unlike
# algolia_ingestion_source, `input` is required: every destination needs an
# `indexName` to write to.
resource "algolia_ingestion_destination" "search" {
  name = "terraform-example-search-destination"
  type = "search"

  input = jsonencode({
    indexName = "products"
  })
}

# An "insights" destination records data as user events in the Insights
# API instead of writing to a search index.
resource "algolia_ingestion_destination" "insights" {
  name = "terraform-example-insights-destination"
  type = "insights"

  input = jsonencode({
    indexName = "products"
  })
}

resource "algolia_ingestion_authentication" "shopify" {
  name     = "terraform-example-shopify-auth"
  type     = "apiKey"
  platform = "shopify"

  input = jsonencode({
    key = "shpat_example_api_key"
  })
}

# `transformation_ids` orders the algolia_ingestion_transformation
# resources applied to records before they reach this destination.
resource "algolia_ingestion_destination" "with_authentication" {
  name              = "terraform-example-destination-with-auth"
  type              = "search"
  authentication_id = algolia_ingestion_authentication.shopify.authentication_id

  input = jsonencode({
    indexName           = "shopify_products"
    attributesToExclude = ["internalNotes"]
  })

  transformation_ids = ["6c02aed6-9d1f-4fef-b2f6-9a5a3ea1e2f2"]
}
