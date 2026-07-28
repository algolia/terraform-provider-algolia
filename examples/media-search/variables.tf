variable "algolia_app_id" {
  description = "Algolia application ID."
  type        = string
}

variable "algolia_api_key" {
  description = "Admin API key Terraform uses to manage resources. Keep it out of source control (use TF_VAR_algolia_api_key or a secrets manager)."
  type        = string
  sensitive   = true
}
