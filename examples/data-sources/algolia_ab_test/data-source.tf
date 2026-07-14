terraform {
  required_providers {
    algolia = {
      source = "registry.terraform.io/algolia/algolia"
    }
  }
}

provider "algolia" {
  analytics_region = "us"
}

# Reads an A/B test's current, enriched state - including runtime results
# (per-variant metric values, significance, estimated sample size). Unlike
# the algolia_ab_test resource, which never refreshes variants/metrics/
# configuration to avoid corrupting a write-once configuration with runtime
# data, this data source always reflects the latest GetABTest response.
data "algolia_ab_test" "example" {
  ab_test_id = 4242
}

output "ab_test_status" {
  value = data.algolia_ab_test.example.status
}

output "ab_test_variants" {
  # JSON-encoded array of variants, including per-variant results.
  value = jsondecode(data.algolia_ab_test.example.variants)
}
