provider "algolia" {
  app_id  = var.algolia_app_id
  api_key = var.algolia_api_key

  # The Ingestion API is region-routed: analytics_region must match the region
  # your application is hosted in (or set ALGOLIA_ANALYTICS_REGION).
  analytics_region = var.analytics_region
}

# 1. Source - where records come from. Here, a CSV feed pulled over HTTPS.
resource "algolia_ingestion_source" "products_csv" {
  # Destroying this is not a recoverable step: removing it stops whatever tasks read from it.
  # This example is meant to be torn down again, so the guard is off.
  deletion_protection = false

  name = "products-csv-feed"
  type = "csv"

  input = jsonencode({
    url            = var.products_csv_url
    uniqueIDColumn = "id"
  })
}

# 2. Transformation - reshape each record before it is indexed. This one
#    stamps an ingestion timestamp onto every record. The logic goes in
#    `input`: the API requires it whenever `type` is set.
resource "algolia_ingestion_transformation" "stamp_indexed_at" {
  # Destroying this is not a recoverable step: removing it changes what every task using it writes.
  # This example is meant to be torn down again, so the guard is off.
  deletion_protection = false

  name = "stamp-indexed-at"
  type = "code"

  input = jsonencode({
    code = <<-EOT
      function transform({ record }) {
        record.indexedAt = new Date().toISOString();
        return record;
      }
    EOT
  })
}

# 3. Authentication - a destination needs one, and its type is fixed by the
#    destination's type: a "search" destination requires "algolia", and the
#    API rejects an authentication pointing at a different application.
resource "algolia_ingestion_authentication" "destination" {
  # Destroying this is not a recoverable step: removing it breaks every source and task that authenticates with it.
  # This example is meant to be torn down again, so the guard is off.
  deletion_protection = false

  name = "products-destination-auth"
  type = "algolia"

  input = jsonencode({
    appID  = var.algolia_app_id
    apiKey = var.algolia_api_key
  })
}

# 4. Destination - the Algolia index records are written to, with the
#    transformation applied on the way in.
resource "algolia_ingestion_destination" "products_index" {
  # Destroying this is not a recoverable step: removing it stops whatever tasks write to it.
  # This example is meant to be torn down again, so the guard is off.
  deletion_protection = false

  name              = "products-search-destination"
  type              = "search"
  authentication_id = algolia_ingestion_authentication.destination.authentication_id

  input = jsonencode({
    indexName = "products"
  })

  transformation_ids = [algolia_ingestion_transformation.stamp_indexed_at.transformation_id]
}

# 5. Task - ties source and destination together on a schedule. This one
#    fully replaces the index contents every night at 02:00. Removing `cron`
#    later forces the task to be replaced, since the API cannot clear a
#    schedule; to pause it instead, set `enabled = false`.
resource "algolia_ingestion_task" "nightly_sync" {
  # Destroying this is not a recoverable step: removing it stops a running pipeline.
  # This example is meant to be torn down again, so the guard is off.
  deletion_protection = false

  source_id      = algolia_ingestion_source.products_csv.source_id
  destination_id = algolia_ingestion_destination.products_index.destination_id
  action         = "replace"
  cron           = "0 2 * * *"
  enabled        = true
}
