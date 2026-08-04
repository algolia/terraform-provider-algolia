data "algolia_api_keys" "example" {}

# Keys are identified by their description, which is the only non-secret handle they
# have. `value` is the key itself and is marked sensitive: an output derived from it
# must set `sensitive = true`, or the plan fails with "Output refers to sensitive
# values", and the key would land in state and in CI logs.
output "api_key_descriptions" {
  value = [for key in data.algolia_api_keys.example.keys : key.description]
}
