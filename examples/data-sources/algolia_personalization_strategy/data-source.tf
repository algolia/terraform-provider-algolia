provider "algolia" {
  analytics_region = "us"
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

data "algolia_personalization_strategy" "example" {
  depends_on = [algolia_personalization_strategy.example]
}

# The strategy Algolia is applying, useful for confirming a write landed.
output "personalization_impact" {
  value = data.algolia_personalization_strategy.example.personalization_impact
}
