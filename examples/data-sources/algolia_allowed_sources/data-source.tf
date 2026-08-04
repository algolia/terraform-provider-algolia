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

# The allowlist Algolia currently holds. Note the resource above replaces the list
# wholesale, so this reads back exactly what was declared there; entries added out of
# band are removed by that write, not reported here.
output "allowed_sources" {
  value = data.algolia_allowed_sources.example.source
}
