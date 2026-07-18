resource "algolia_index" "example" {
  name                = "products"
  deletion_protection = false
}

# A two-way synonym: searches for any listed term also match the others.
resource "algolia_synonym" "example" {
  index_name = algolia_index.example.name
  object_id  = "phone-synonym"
  type       = "synonym"
  synonyms   = ["iphone", "ios phone"]
}
