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

# Reads an existing source's configuration, including `input`: unlike
# algolia_ingestion_authentication, the Ingestion API does not redact a
# source's configuration.
data "algolia_ingestion_source" "example" {
  source_id = "6c02aed6-9d1f-4fef-b2f6-9a5a3ea1e2f2"
}

output "source_type" {
  value = data.algolia_ingestion_source.example.type
}

output "source_input" {
  value = data.algolia_ingestion_source.example.input
}
