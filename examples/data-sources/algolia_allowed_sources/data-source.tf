resource "algolia_allowed_sources" "example" {
  source = [
    {
      source      = "203.0.113.10/32"
      description = "Terraform CI runner - required, do not remove"
    },
  ]
}

data "algolia_allowed_sources" "example" {
  depends_on = [algolia_allowed_sources.example]
}

# The allowlist Algolia currently holds, including any entry added outside Terraform.
output "allowed_sources" {
  value = data.algolia_allowed_sources.example.source
}
