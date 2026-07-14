terraform {
  required_providers {
    algolia = {
      source = "registry.terraform.io/algolia/algolia"
    }
  }
}

provider "algolia" {
  # The A/B Testing API is region-routed: analytics_region (or the
  # ALGOLIA_ANALYTICS_REGION environment variable) must be set.
  analytics_region = "us"
}

# The A/B Testing API has no update endpoint, so name/end_at/variants/
# metrics/configuration are all write-once: changing any of them forces
# this resource to be replaced (a new A/B test is created, the old one is
# deleted). Read never refreshes variants/metrics/configuration from the
# API - GetABTest's response is enriched with runtime results whose shape
# diverges from what was submitted here. Use the algolia_ab_test data
# source to inspect a test's live results.
resource "algolia_ab_test" "ranking_experiment" {
  name   = "homepage-ranking-experiment"
  end_at = "2026-12-31T23:59:59Z"

  # The first variant is conventionally the control (e.g. production);
  # the rest are indices with changed settings to test against it.
  variants = jsonencode([
    {
      index             = "products_prod"
      trafficPercentage = 50
      description       = "control"
    },
    {
      index             = "products_prod_ranking_variant"
      trafficPercentage = 50
      description       = "custom ranking variant"
    },
  ])

  # Only these metrics are considered when calculating results.
  metrics = jsonencode([
    { name = "addToCartRate" },
    { name = "revenue", dimension = "USD" },
  ])

  configuration = jsonencode({
    minimumDetectableEffect = {
      size   = 0.1
      metric = "conversionRate"
    }
    errorCorrection = "bonferroni"
  })
}

# A/B test that compares custom search parameters on the same index,
# instead of two different indices.
resource "algolia_ab_test" "search_params_experiment" {
  name   = "typo-tolerance-experiment"
  end_at = "2026-12-31T23:59:59Z"

  variants = jsonencode([
    {
      index             = "products_prod"
      trafficPercentage = 50
      description       = "default typo tolerance"
    },
    {
      index                  = "products_prod"
      trafficPercentage      = 50
      description            = "typo tolerance disabled"
      customSearchParameters = { typoTolerance = false }
    },
  ])

  metrics = jsonencode([
    { name = "clickThroughRate" },
  ])
}
