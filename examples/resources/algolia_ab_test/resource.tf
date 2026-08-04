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
  name = "homepage-ranking-experiment"

  # An A/B test cannot run for more than 90 days, so this has to be within 90 days
  # of the apply. A date further out is rejected with "ABTest cannot run for more
  # than 90 days (due to retention)".
  end_at = var.ab_test_end_at

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

  # `minimumDetectableEffect` is deliberately not set here: the API rejects it
  # unless sample sizes are supplied alongside it ("Sample sizes are required when
  # a minimum detectable effect is provided").
  configuration = jsonencode({
    errorCorrection = "bonferroni"
  })
}

# A/B test that compares custom search parameters on one index, instead of two
# different indices. Note it targets a different index from the test above:
# Algolia does not run two tests against the same index at once, so declaring
# both against `products_prod` would fail on apply.
resource "algolia_ab_test" "search_params_experiment" {
  name   = "typo-tolerance-experiment"
  end_at = var.ab_test_end_at

  variants = jsonencode([
    {
      index             = "articles_prod"
      trafficPercentage = 50
      description       = "default typo tolerance"
    },
    {
      index                  = "articles_prod"
      trafficPercentage      = 50
      description            = "typo tolerance disabled"
      customSearchParameters = { typoTolerance = false }
    },
  ])

  metrics = jsonencode([
    { name = "clickThroughRate" },
  ])
}

variable "ab_test_end_at" {
  description = "When both tests stop, RFC3339. Must be within 90 days of the apply: the API rejects a longer run because A/B test data is only retained that long."
  type        = string
}
