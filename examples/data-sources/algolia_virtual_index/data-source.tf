data "algolia_virtual_index" "example" {
  name = "products_price_asc"
}

output "virtual_index_primary" {
  value = data.algolia_virtual_index.example.primary_index_name
}
