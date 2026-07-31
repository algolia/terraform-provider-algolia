terraform {
  required_providers {
    algolia = {
      source = "registry.terraform.io/algolia/algolia"
    }
  }
}

provider "algolia" {
  # The Ingestion API is region-routed: analytics_region (or the
  # ALGOLIA_ANALYTICS_REGION environment variable) must be set.
  analytics_region = "us"
}

# A "code" transformation runs custom JavaScript against each record. Supply it
# through `input`, which is what the API expects whenever `type` is set - a
# payload carrying `type` without `input` is rejected with "'input' is required
# if 'Type' is present".
resource "algolia_ingestion_transformation" "code" {
  name = "terraform-example-code-transformation"
  type = "code"

  input = jsonencode({
    code = <<-EOT
      function transform({ record }) {
        record.indexedAt = new Date().toISOString();
        return record;
      }
    EOT
  })
}

# The deprecated `code` attribute still works on its own, without `type`. It
# conflicts with `input`, so set one or the other.
resource "algolia_ingestion_transformation" "legacy_code" {
  name = "terraform-example-legacy-code-transformation"

  code = <<-EOT
    function transform({ record }) {
      record.indexedAt = new Date().toISOString();
      return record;
    }
  EOT
}

# A "noCode" transformation is built from a series of steps instead of raw
# code. Its `input` is JSON-encoded, like algolia_ingestion_source and
# algolia_ingestion_destination.
#
# Every step needs an `id`, a `name` and a `configuration.action`, and the
# action's `kind` is one of addAttribute, removeAttribute or filterRecords.
# `formula` is an expression, so a literal string value is quoted inside it.
# Note `enabled` defaults to false: a step is authored but inert until you set
# it, which is easy to miss.
resource "algolia_ingestion_transformation" "no_code" {
  name        = "terraform-example-no-code-transformation"
  type        = "noCode"
  description = "Adds a static attribute to every record"

  input = jsonencode({
    steps = [
      {
        id      = "add-source"
        name    = "Add a source attribute"
        enabled = true

        configuration = {
          action = {
            kind          = "addAttribute"
            attributeName = "source"
            formula       = "\"terraform\""
          }
        }
      }
    ]
  })
}

# A transformation's authentications must be of type "secrets" - the API
# rejects any other type with "Invalid authentications for transformation".
# Each key becomes available to the transformation code at run time, which is
# how it reaches an external API without the credential appearing in the code.
resource "algolia_ingestion_authentication" "enrichment_api" {
  name = "terraform-example-enrichment-secrets"
  type = "secrets"

  input = jsonencode({
    enrichmentApiKey = "example-secret-value"
  })
}

# `authentication_ids` associates the algolia_ingestion_authentication
# resources this transformation needs.
resource "algolia_ingestion_transformation" "with_authentication" {
  name = "terraform-example-transformation-with-auth"
  type = "code"

  input = jsonencode({
    code = "function transform({ record }) { return record; }"
  })

  authentication_ids = [algolia_ingestion_authentication.enrichment_api.authentication_id]
}
