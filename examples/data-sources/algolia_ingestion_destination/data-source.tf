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

# Reads an existing destination's configuration, including `input` and
# `transformation_ids`: like algolia_ingestion_source, the Ingestion API
# does not redact a destination's configuration.
data "algolia_ingestion_destination" "example" {
  destination_id = "6c02aed6-9d1f-4fef-b2f6-9a5a3ea1e2f2"
}

output "destination_type" {
  value = data.algolia_ingestion_destination.example.type
}

output "destination_input" {
  value = data.algolia_ingestion_destination.example.input
}
