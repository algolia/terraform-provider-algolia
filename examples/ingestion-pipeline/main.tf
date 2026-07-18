provider "algolia" {
  app_id  = var.algolia_app_id
  api_key = var.algolia_api_key

  # The Ingestion API is region-routed: analytics_region must match the region
  # your application is hosted in (or set ALGOLIA_ANALYTICS_REGION).
  analytics_region = var.analytics_region
}

# 1. Source - where records come from. Here, a CSV feed pulled over HTTPS.
resource "algolia_ingestion_source" "products_csv" {
  name = "products-csv-feed"
  type = "csv"

  input = jsonencode({
    url            = var.products_csv_url
    uniqueIDColumn = "id"
  })
}

# 2. Transformation - reshape each record before it is indexed. This one
#    stamps an ingestion timestamp onto every record.
resource "algolia_ingestion_transformation" "stamp_indexed_at" {
  name = "stamp-indexed-at"
  type = "code"

  code = <<-EOT
    function transform(record) {
      record.indexedAt = new Date().toISOString();
      return record;
    }
  EOT
}

# 3. Destination - the Algolia index records are written to, with the
#    transformation applied on the way in.
resource "algolia_ingestion_destination" "products_index" {
  name = "products-search-destination"
  type = "search"

  input = jsonencode({
    indexName = "products"
  })

  transformation_ids = [algolia_ingestion_transformation.stamp_indexed_at.transformation_id]
}

# 4. Task - ties source and destination together on a schedule. This one
#    fully replaces the index contents every night at 02:00.
resource "algolia_ingestion_task" "nightly_sync" {
  source_id      = algolia_ingestion_source.products_csv.source_id
  destination_id = algolia_ingestion_destination.products_index.destination_id
  action         = "replace"
  cron           = "0 2 * * *"
  enabled        = true
}
