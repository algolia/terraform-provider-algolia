data "algolia_index" "example" {
  name = "existing_index"
}

output "index_entries" {
  value = data.algolia_index.example.entries
}

output "searchable_attributes" {
  value = data.algolia_index.example.attributes.searchable_attributes
}
