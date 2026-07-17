output "products_index" {
  description = "Name of the managed products index."
  value       = algolia_index.products.name
}

output "frontend_search_api_key" {
  description = "Search-only API key for storefront clients. Only known after apply."
  value       = algolia_api_key.frontend_search.id
  sensitive   = true
}
