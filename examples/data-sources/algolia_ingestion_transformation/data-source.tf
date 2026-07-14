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

# Reads an existing transformation's configuration, including `code` and
# `input`: like algolia_ingestion_source and algolia_ingestion_destination,
# the Ingestion API does not redact a transformation's configuration.
data "algolia_ingestion_transformation" "example" {
  transformation_id = "6c02aed6-9d1f-4fef-b2f6-9a5a3ea1e2f2"
}

output "transformation_type" {
  value = data.algolia_ingestion_transformation.example.type
}

output "transformation_code" {
  value = data.algolia_ingestion_transformation.example.code
}
