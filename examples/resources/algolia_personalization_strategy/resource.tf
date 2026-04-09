provider "algolia" {
  personalization_region = "us"
}

resource "algolia_personalization_strategy" "example" {
  personalization_impact = 80

  events_scoring {
    event_name = "Product Clicked"
    event_type = "click"
    score      = 50
  }

  facets_scoring {
    facet_name = "category"
    score      = 30
  }
}
