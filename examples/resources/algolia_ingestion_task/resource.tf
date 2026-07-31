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
  description = "Algolia application ID. A destination's authentication must point at the same application the provider is configured with."
  type        = string
}

variable "algolia_api_key" {
  description = "API key the ingestion pipeline writes with. Keep it out of source control (use TF_VAR_algolia_api_key or a secrets manager)."
  type        = string
  sensitive   = true
}

# A task ties a source and a destination together: it reads from the
# source, optionally transforms records, and writes them to the
# destination using `action`.
#
# Only a pull-based source can be scheduled - the API rejects a `cron` on a
# "push" source with "a source of type 'push' isn't able to schedule tasks".
resource "algolia_ingestion_source" "products_csv" {
  # Destroying this is not a recoverable step: removing it stops whatever tasks read from it.
  deletion_protection = true

  name = "terraform-example-csv-source"
  type = "csv"

  input = jsonencode({
    url            = "https://example.com/products.csv"
    uniqueIDColumn = "id"
  })
}

# A "search" destination requires an authentication of type "algolia".
resource "algolia_ingestion_authentication" "destination" {
  # Destroying this is not a recoverable step: removing it breaks every source and task that authenticates with it.
  deletion_protection = true

  name = "terraform-example-task-destination-auth"
  type = "algolia"

  input = jsonencode({
    appID  = var.algolia_app_id
    apiKey = var.algolia_api_key
  })
}

resource "algolia_ingestion_destination" "products" {
  # Destroying this is not a recoverable step: removing it stops whatever tasks write to it.
  deletion_protection = true

  name              = "terraform-example-products-destination"
  type              = "search"
  authentication_id = algolia_ingestion_authentication.destination.authentication_id

  input = jsonencode({
    indexName = "products"
  })
}

# `cron` schedules the task to run automatically; omit it for an on-demand
# task that only runs when triggered manually. `action` and `source_id`
# force replacement if changed: the Ingestion API's task update endpoint
# has no way to change either after creation. Removing `cron` later forces
# replacement too, because the API cannot clear a schedule - to pause a task
# instead, set `enabled = false`.
resource "algolia_ingestion_task" "csv_to_products" {
  # Destroying this is not a recoverable step: removing it stops a running pipeline.
  deletion_protection = true

  source_id      = algolia_ingestion_source.products_csv.source_id
  destination_id = algolia_ingestion_destination.products.destination_id
  action         = "replace"
  cron           = "0 0 * * *"
  enabled        = true

  # `notifications`/`policies` are JSON-encoded, like `input`.
  notifications = jsonencode({
    email = {
      enabled = true
    }
  })

  policies = jsonencode({
    criticalThreshold = 5
  })
}

# A "push" source needs no `input`, and a task on top of it needs no
# `input` either: records are pushed to it directly rather than pulled on
# a schedule, so this task is on-demand (no `cron`).
resource "algolia_ingestion_source" "push" {
  # Destroying this is not a recoverable step: removing it stops whatever tasks read from it.
  deletion_protection = true

  name = "terraform-example-push-source"
  type = "push"
}

resource "algolia_ingestion_task" "push_to_products" {
  # Destroying this is not a recoverable step: removing it stops a running pipeline.
  deletion_protection = true

  source_id      = algolia_ingestion_source.push.source_id
  destination_id = algolia_ingestion_destination.products.destination_id
  action         = "save"
  enabled        = false
}
