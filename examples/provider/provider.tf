terraform {
  required_providers {
    algolia = {
      source  = "algolia/algolia"
      version = "0.1.0"
    }
  }
}

# Configure the Algolia provider.
# Credentials can also be set via ALGOLIA_APP_ID and ALGOLIA_API_KEY environment variables.
provider "algolia" {
  app_id  = var.algolia_app_id
  api_key = var.algolia_api_key
}

variable "algolia_app_id" {
  description = "Algolia Application ID"
  type        = string
  sensitive   = false
}

variable "algolia_api_key" {
  description = "Algolia Admin API Key"
  type        = string
  sensitive   = true
}
