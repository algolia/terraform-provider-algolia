terraform {
  required_providers {
    algolia = {
      source = "registry.terraform.io/algolia/algolia"
    }
  }
}

provider "algolia" {}

resource "algolia_index" "products" {
  name                = "products"
  deletion_protection = false
}

# A manually curated collection — the provider diffs `records` against prior
# state on each apply and sends the add/remove deltas to the API.
resource "algolia_collection" "summer_deals" {
  name       = "Summer Deals"
  index_name = algolia_index.products.name

  records = [
    "sku-1001",
    "sku-1002",
    "sku-1003",
  ]

  deletion_protection = false
}

# A rule-based collection defined by facet and numeric filters.
#
# Each `facet_filter` / `numeric_filter` block is AND-ed with its siblings.
# Filters inside a block's `filters` list are OR-ed with each other.
#
# The example below selects: Apple products (brand AND) in Phone or Tablet
# categories (OR) priced under $100 or over $1000 (OR).
resource "algolia_collection" "discounted_apple" {
  name        = "Discounted Apple Gear"
  index_name  = algolia_index.products.name
  description = "Apple products currently on sale."

  conditions {
    facet_filter {
      filters = ["brand:Apple"]
    }
    facet_filter {
      filters = ["category:Phone", "category:Tablet"]
    }

    numeric_filter {
      filters = ["price<100", "price>1000"]
    }
  }

  deletion_protection = false
}

# A hybrid collection: rule-based inclusion via `conditions`, plus a few
# hand-picked SKUs pinned in via `records`. The two fields are independent —
# records are always included; conditions bring in anything matching.
resource "algolia_collection" "holiday_picks" {
  name        = "Holiday Picks"
  index_name  = algolia_index.products.name
  description = "Editor-curated highlights plus anything on sale in featured categories."

  # Always include these, regardless of filters:
  records = [
    "sku-9001",
    "sku-9002",
  ]

  # Plus anything matching these conditions:
  conditions {
    facet_filter {
      filters = ["category:Gift", "category:Seasonal"]
    }
    numeric_filter {
      filters = ["discount>0"]
    }
  }

  deletion_protection = false
}
