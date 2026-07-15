resource "algolia_composition" "example" {
  object_id = "featured-products"
  name      = "Featured products"

  behavior = jsonencode({
    injection = {
      main = {
        source = {
          search = {
            index = "products"
          }
        }
      }
    }
  })
}

resource "algolia_composition_rule" "example" {
  composition_id = algolia_composition.example.object_id
  object_id      = "boost-featured-on-mobile"
  description    = "Swap in the featured index for mobile shoppers"
  enabled        = true

  # JSON-encoded list of conditions that trigger the rule.
  conditions = jsonencode([
    {
      filters = "brand:apple"
      context = "mobile"
    }
  ])

  # JSON-encoded consequence applied when the conditions match.
  consequence = jsonencode({
    behavior = {
      injection = {
        main = {
          source = {
            search = {
              index = "products_featured"
            }
          }
        }
      }
    }
  })
}
