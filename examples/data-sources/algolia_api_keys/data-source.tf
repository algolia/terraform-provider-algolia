data "algolia_api_keys" "example" {}

output "api_key_values" {
  value = [for key in data.algolia_api_keys.example.keys : key.value]
}
