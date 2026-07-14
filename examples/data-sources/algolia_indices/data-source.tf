data "algolia_indices" "example" {}

output "index_names" {
  value = [for index in data.algolia_indices.example.indices : index.name]
}
