resource "algolia_query_suggestions_config" "example" {
  index_name = "my_suggestions"
  region     = "us"

  source_index {
    index_name     = "products"
    replicas       = true
    min_hits       = 5
    min_letters    = 4
    analytics_tags = ["production"]

    facets {
      attribute = "brand"
      amount    = 1
    }

    generate = jsonencode([["brand"], ["category", "brand"]])
    external = ["external_index"]
  }

  languages                = ["en", "fr"]
  exclude                  = ["stop_word"]
  enable_personalization   = false
  allow_special_characters = false
  deletion_protection      = true
}
