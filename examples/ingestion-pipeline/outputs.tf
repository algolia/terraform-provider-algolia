output "source_id" {
  description = "ID of the CSV ingestion source."
  value       = algolia_ingestion_source.products_csv.source_id
}

output "destination_id" {
  description = "ID of the search destination."
  value       = algolia_ingestion_destination.products_index.destination_id
}

output "task_id" {
  description = "ID of the scheduled sync task. Trigger a run on demand from the Algolia dashboard or the Ingestion API."
  value       = algolia_ingestion_task.nightly_sync.task_id
}
