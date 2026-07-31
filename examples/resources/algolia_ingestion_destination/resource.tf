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

variable "algolia_app_id" {
  description = "Algolia application ID. Must be the same application the provider is configured with: the API rejects a destination whose authentication points elsewhere."
  type        = string
}

variable "algolia_api_key" {
  description = "API key the ingestion pipeline writes with. Keep it out of source control (use TF_VAR_algolia_api_key or a secrets manager)."
  type        = string
  sensitive   = true
}

# Every destination needs an authentication, and its type is dictated by the
# destination's type - the API rejects any other pairing. A "search"
# destination requires type "algolia".
resource "algolia_ingestion_authentication" "search" {
  # Destroying this is not a recoverable step: removing it breaks every source and task that authenticates with it.
  deletion_protection = true

  name = "terraform-example-search-auth"
  type = "algolia"

  input = jsonencode({
    appID  = var.algolia_app_id
    apiKey = var.algolia_api_key
  })
}

# A "search" destination writes records to an Algolia index. Unlike
# algolia_ingestion_source, `input` is required: every destination needs an
# `indexName` to write to.
resource "algolia_ingestion_destination" "search" {
  # Destroying this is not a recoverable step: removing it stops whatever tasks write to it.
  deletion_protection = true

  name              = "terraform-example-search-destination"
  type              = "search"
  authentication_id = algolia_ingestion_authentication.search.authentication_id

  input = jsonencode({
    indexName = "products"
  })
}

# An "insights" destination records data as user events in the Insights API
# instead of writing to a search index, and takes its own authentication type:
# "algoliaInsights", not "algolia".
resource "algolia_ingestion_authentication" "insights" {
  # Destroying this is not a recoverable step: removing it breaks every source and task that authenticates with it.
  deletion_protection = true

  name = "terraform-example-insights-auth"
  type = "algoliaInsights"

  input = jsonencode({
    appID  = var.algolia_app_id
    apiKey = var.algolia_api_key
  })
}

resource "algolia_ingestion_destination" "insights" {
  # Destroying this is not a recoverable step: removing it stops whatever tasks write to it.
  deletion_protection = true

  name              = "terraform-example-insights-destination"
  type              = "insights"
  authentication_id = algolia_ingestion_authentication.insights.authentication_id

  input = jsonencode({
    indexName = "products"
  })
}

resource "algolia_ingestion_transformation" "drop_internal_notes" {
  # Destroying this is not a recoverable step: removing it changes what every task using it writes.
  deletion_protection = true

  name = "terraform-example-drop-internal-notes"
  type = "code"

  input = jsonencode({
    code = <<-EOT
      function transform({ record }) {
        delete record.internalNotes;
        return record;
      }
    EOT
  })
}

# `transformation_ids` orders the algolia_ingestion_transformation resources
# applied to records on their way into this destination. Reference the
# resources rather than pasting IDs, so Terraform creates them first.
resource "algolia_ingestion_destination" "with_transformation" {
  # Destroying this is not a recoverable step: removing it stops whatever tasks write to it.
  deletion_protection = true

  name              = "terraform-example-destination-with-transformation"
  type              = "search"
  authentication_id = algolia_ingestion_authentication.search.authentication_id

  input = jsonencode({
    indexName = "products_transformed"
  })

  transformation_ids = [algolia_ingestion_transformation.drop_internal_notes.transformation_id]
}
