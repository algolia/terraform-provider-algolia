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

variable "destination_app_id" {
  description = "Algolia application ID that this authentication writes to."
  type        = string
}

variable "destination_api_key" {
  description = "Algolia API key with addObject/deleteIndex/editSettings ACLs on the destination application."
  type        = string
  sensitive   = true
}

# An "algolia" authentication is typically used to let an ingestion task
# write into another Algolia application (e.g. push connectors, or
# source/destination pairs that live in different apps).
resource "algolia_ingestion_authentication" "algolia_destination" {
  # Destroying this is not a recoverable step: removing it breaks every source and task that authenticates with it.
  deletion_protection = true

  name = "terraform-example-algolia-auth"
  type = "algolia"

  # input is JSON-encoded credentials matching `type`; see the schema docs
  # for the shape expected by each authentication type. input is
  # write-only: Terraform never reads it back (the API redacts secrets), so
  # it will not show drift if the credential is rotated out-of-band.
  input = jsonencode({
    appID  = var.destination_app_id
    apiKey = var.destination_api_key
  })
}

variable "shopify_api_key" {
  description = "API key used to authenticate with a Shopify store."
  type        = string
  sensitive   = true
}

# Platform-scoped authentications (bigcommerce, commercetools, shopify) are
# used by sources/destinations that connect to that ecommerce platform.
# platform is set at creation time only: the Ingestion API's update endpoint
# does not support changing it, so changing it in configuration forces
# replacement.
resource "algolia_ingestion_authentication" "shopify_source" {
  # Destroying this is not a recoverable step: removing it breaks every source and task that authenticates with it.
  deletion_protection = true

  name     = "terraform-example-shopify-auth"
  type     = "apiKey"
  platform = "shopify"

  input = jsonencode({
    key = var.shopify_api_key
  })
}
