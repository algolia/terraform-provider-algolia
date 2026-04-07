resource "algolia_index" "example" {
  name                = "products"
  deletion_protection = false
}

resource "algolia_synonym" "example" {
  index_name = algolia_index.example.name
  object_id  = "phone-synonym"
  type       = "synonym"
  synonyms   = ["iphone", "ios phone"]
}

data "algolia_synonym" "example" {
  index_name = algolia_synonym.example.index_name
  object_id  = algolia_synonym.example.object_id
}
