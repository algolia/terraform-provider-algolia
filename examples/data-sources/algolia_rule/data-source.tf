resource "algolia_index" "example" {
  name                = "products"
  deletion_protection = false
}

resource "algolia_rule" "example" {
  index_name  = algolia_index.example.name
  object_id   = "featured-brand"
  description = "Boost featured brand searches"

  conditions {
    pattern   = "{facet:brand}"
    anchoring = "contains"
  }

  consequence {
    params_json = jsonencode({
      query = "featured"
    })
  }
}

data "algolia_rule" "example" {
  index_name = algolia_rule.example.index_name
  object_id  = algolia_rule.example.object_id
}
