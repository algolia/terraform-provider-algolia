resource "algolia_index" "example" {
  name                = "products"
  deletion_protection = false
}

resource "algolia_recommend_rule" "hide_discontinued" {
  index_name  = algolia_index.example.name
  model       = "related-products"
  object_id   = "hide-discontinued"
  description = "Hide discontinued items from related-products recommendations"

  condition = jsonencode({
    filters = "brand:apple"
  })

  consequence = jsonencode({
    hide = [
      { objectID = "42" },
    ]
  })

  validity = jsonencode([
    { from = 1893456000, until = 1893542400 },
  ])
}
