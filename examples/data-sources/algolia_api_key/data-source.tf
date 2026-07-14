data "algolia_api_key" "example" {
  key = "AB1CD2EF3G4HI5JK6LM7NOP"
}

output "api_key_acl" {
  value = data.algolia_api_key.example.acl
}
