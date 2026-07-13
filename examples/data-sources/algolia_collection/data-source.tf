resource "algolia_index" "example" {
  name                = "products"
  deletion_protection = false
}

resource "algolia_collection" "example" {
  name       = "Summer Deals"
  index_name = algolia_index.example.name

  records = ["sku-1001", "sku-1002"]

  deletion_protection = false
}

data "algolia_collection" "example" {
  id = algolia_collection.example.id
}
