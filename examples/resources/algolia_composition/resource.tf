resource "algolia_composition" "example" {
  object_id   = "featured-products"
  name        = "Featured products"
  description = "Composition that surfaces products from a single source index"

  # JSON-encoded composition behavior (injection or multifeed).
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

  sorting_strategy = jsonencode({
    relevancy = "asc"
  })
}
