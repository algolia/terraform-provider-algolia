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

# A task ties a source and a destination together: it reads from the
# source, optionally transforms records, and writes them to the
# destination using `action`.
resource "algolia_ingestion_source" "shopify" {
  name = "terraform-example-shopify-source"
  type = "shopify"

  input = jsonencode({
    shop = "my-shop"
  })
}

resource "algolia_ingestion_destination" "products" {
  name = "terraform-example-products-destination"
  type = "search"

  input = jsonencode({
    indexName = "products"
  })
}

# `cron` schedules the task to run automatically; omit it for an on-demand
# task that only runs when triggered manually. `action` and `source_id`
# force replacement if changed: the Ingestion API's task update endpoint
# has no way to change either after creation.
resource "algolia_ingestion_task" "shopify_to_products" {
  source_id      = algolia_ingestion_source.shopify.source_id
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
    criticalThreshold = 50
  })
}

# A "push" source needs no `input`, and a task on top of it needs no
# `input` either: records are pushed to it directly rather than pulled on
# a schedule, so this task is on-demand (no `cron`).
resource "algolia_ingestion_source" "push" {
  name = "terraform-example-push-source"
  type = "push"
}

resource "algolia_ingestion_task" "push_to_products" {
  source_id      = algolia_ingestion_source.push.source_id
  destination_id = algolia_ingestion_destination.products.destination_id
  action         = "save"
  enabled        = false
}
