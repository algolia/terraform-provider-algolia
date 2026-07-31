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

# A "csv" source pulls records from a file over HTTP(S). `input` is
# JSON-encoded configuration matching `type`; see the schema docs for the
# shape expected by each source type.
resource "algolia_ingestion_source" "csv" {
  # Destroying this is not a recoverable step: removing it stops whatever tasks read from it.
  deletion_protection = true

  name = "terraform-example-csv-source"
  type = "csv"

  input = jsonencode({
    url            = "https://example.com/products.csv"
    uniqueIDColumn = "id"
  })
}

# A "push" source accepts records pushed directly to it (e.g. via the
# Ingestion API's push endpoint) and needs no configuration at all, so
# `input` is omitted.
resource "algolia_ingestion_source" "push" {
  # Destroying this is not a recoverable step: removing it stops whatever tasks read from it.
  deletion_protection = true

  name = "terraform-example-push-source"
  type = "push"
}

variable "shopify_shop_url" {
  description = "URL of the Shopify store to pull products from."
  type        = string
}

# A platform-scoped source (shopify, bigcommerce, commercetools) typically
# references an algolia_ingestion_authentication resource created with a
# matching `platform`.
resource "algolia_ingestion_authentication" "shopify" {
  # Destroying this is not a recoverable step: removing it breaks every source and task that authenticates with it.
  deletion_protection = true

  name     = "terraform-example-shopify-auth"
  type     = "apiKey"
  platform = "shopify"

  input = jsonencode({
    key = "shpat_example_api_key"
  })
}

resource "algolia_ingestion_source" "shopify" {
  # Destroying this is not a recoverable step: removing it stops whatever tasks read from it.
  deletion_protection = true

  name              = "terraform-example-shopify-source"
  type              = "shopify"
  authentication_id = algolia_ingestion_authentication.shopify.authentication_id

  input = jsonencode({
    shopURL = var.shopify_shop_url
  })
}
