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

# A "code" transformation runs custom JavaScript against each record. The
# deprecated `code` attribute is the simplest way to specify it directly.
resource "algolia_ingestion_transformation" "code" {
  name = "terraform-example-code-transformation"
  type = "code"

  code = <<-EOT
    function transform(record) {
      record.indexedAt = new Date().toISOString();
      return record;
    }
  EOT
}

# A "noCode" transformation is built from a series of steps instead of raw
# code. Its `input` is JSON-encoded, like algolia_ingestion_source and
# algolia_ingestion_destination.
resource "algolia_ingestion_transformation" "no_code" {
  name        = "terraform-example-no-code-transformation"
  type        = "noCode"
  description = "Adds a static field to every record"

  input = jsonencode({
    steps = [
      {
        action = "addField"
        field  = "source"
        value  = "terraform"
      }
    ]
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

# `authentication_ids` associates the algolia_ingestion_authentication
# resources this transformation needs (e.g. to call an external API as part
# of its logic).
resource "algolia_ingestion_transformation" "with_authentication" {
  name = "terraform-example-transformation-with-auth"
  type = "code"
  code = "function transform(record) { return record; }"

  authentication_ids = [algolia_ingestion_authentication.shopify.authentication_id]
}
