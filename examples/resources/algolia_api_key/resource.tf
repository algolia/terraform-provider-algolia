# A search-only API key scoped to the "products" index. Safe to expose to
# front-end clients: it can search and browse, but cannot mutate data.
resource "algolia_api_key" "example" {
  description = "Search-only key for the products index"
  acl         = ["search", "browse"]
  indexes     = ["products"]

  max_hits_per_query          = 100
  max_queries_per_ip_per_hour = 10000

  # Optional RFC3339 expiry. Omit for a key that never expires.
  # expires_at = "2030-01-01T00:00:00Z"
}

# The generated key value is sensitive and only known after apply.
output "search_api_key" {
  value     = algolia_api_key.example.id
  sensitive = true
}
