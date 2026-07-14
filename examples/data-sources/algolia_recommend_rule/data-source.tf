resource "algolia_index" "example" {
  name                = "products"
  deletion_protection = false
}

resource "algolia_recommend_rule" "hide_discontinued" {
  index_name = algolia_index.example.name
  model      = "related-products"
  object_id  = "hide-discontinued"

  consequence = jsonencode({
    hide = [
      { objectID = "42" },
    ]
  })
}

data "algolia_recommend_rule" "example" {
  index_name = algolia_recommend_rule.hide_discontinued.index_name
  model      = algolia_recommend_rule.hide_discontinued.model
  object_id  = algolia_recommend_rule.hide_discontinued.object_id
}
