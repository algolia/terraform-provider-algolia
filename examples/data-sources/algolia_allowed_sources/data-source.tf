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
