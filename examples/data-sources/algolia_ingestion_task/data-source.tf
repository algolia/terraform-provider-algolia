terraform {
  required_providers {
    algolia = {
      source = "registry.terraform.io/algolia/algolia"
    }
  }
}

provider "algolia" {
  analytics_region = "us"
}

# Reads an existing task's configuration, including `input`, `notifications`,
# `policies`, and `cursor` in full: like the other Ingestion data sources
# (except algolia_ingestion_authentication), the API does not redact a
# task's configuration.
data "algolia_ingestion_task" "example" {
  task_id = "6c02aed6-9d1f-4fef-b2f6-9a5a3ea1e2f2"
}

output "task_action" {
  value = data.algolia_ingestion_task.example.action
}

output "task_enabled" {
  value = data.algolia_ingestion_task.example.enabled
}

output "task_next_run" {
  value = data.algolia_ingestion_task.example.next_run
}
