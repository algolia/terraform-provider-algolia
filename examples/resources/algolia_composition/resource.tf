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

  # Map of sorting labels to the index (or replica) implementing each sort.
  sorting_strategy = jsonencode({
    "Price (asc)"  = "products_price_asc"
    "Price (desc)" = "products_price_desc"
  })
}
