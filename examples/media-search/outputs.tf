output "titles_index" {
  description = "Name of the managed titles index."
  value       = algolia_index.titles.name
}

output "app_search_api_key" {
  description = "Search-only API key for streaming app clients. Only known after apply."
  value       = algolia_api_key.app_search.id
  sensitive   = true
}
