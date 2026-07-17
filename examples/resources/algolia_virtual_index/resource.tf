resource "algolia_index" "primary" {
  name                = "products"
  deletion_protection = false
}

# A virtual replica shares the primary index's records but applies its own
# custom ranking - ideal for offering an alternative sort order (here, cheapest
# first) without duplicating data.
resource "algolia_virtual_index" "example" {
  name                = "products_price_asc"
  primary_index_name  = algolia_index.primary.name
  deletion_protection = false

  ranking {
    custom_ranking = ["asc(price)"]
  }
}
