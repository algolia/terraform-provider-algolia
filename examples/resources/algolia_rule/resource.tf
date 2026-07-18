resource "algolia_index" "example" {
  name                = "products"
  deletion_protection = false
}

# A merchandising rule: when a query contains a brand facet, boost results
# matching the "featured" query.
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
