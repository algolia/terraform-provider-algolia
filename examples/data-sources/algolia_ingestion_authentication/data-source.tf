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

# Reads an existing authentication's metadata (name, type, platform,
# timestamps). The `input` credentials are never exposed by this data
# source: the Ingestion API redacts secret values, so there is nothing
# meaningful to return.
data "algolia_ingestion_authentication" "example" {
  authentication_id = "6c02aed6-9d1f-4fef-b2f6-9a5a3ea1e2f2"
}

output "authentication_name" {
  value = data.algolia_ingestion_authentication.example.name
}
