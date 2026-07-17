variable "algolia_app_id" {
  description = "Algolia application ID."
  type        = string
}

variable "algolia_api_key" {
  description = "Admin API key Terraform uses to manage resources. Keep it out of source control (use TF_VAR_algolia_api_key or a secrets manager)."
  type        = string
  sensitive   = true
}

variable "analytics_region" {
  description = "Region your Algolia application is hosted in. The Ingestion API is region-routed."
  type        = string
  default     = "us"
}

variable "products_csv_url" {
  description = "Public HTTPS URL of the product CSV feed to ingest."
  type        = string
  default     = "https://example.com/products.csv"
}
