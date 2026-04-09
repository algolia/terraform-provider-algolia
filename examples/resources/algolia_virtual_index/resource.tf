resource "algolia_index" "primary" {
  name                = "products"
  deletion_protection = false
}

resource "algolia_virtual_index" "example" {
  name                = "products_price_desc"
  primary_index_name  = algolia_index.primary.name
  deletion_protection = false

  ranking {
    relevancy_strictness = 80
    custom_ranking       = ["desc(price)"]
  }
}
