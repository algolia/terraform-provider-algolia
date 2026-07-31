resource "algolia_index" "primary" {
  name                = "products"
  deletion_protection = false
}

# A virtual replica shares the primary index's records but applies its own
# custom ranking - ideal for offering an alternative sort order (here, cheapest
# first) without duplicating data.
resource "algolia_virtual_index" "example" {
  name                = "products_price_asc"
  primary_index_name  = algolia_index.primary.name
  deletion_protection = false

  ranking {
    custom_ranking = ["asc(price)"]
  }
}

# Several virtual replicas can share one primary. Each adds itself to the
# primary's replicas list, and the provider serialises those writes, so they need
# no ordering or depends_on between them.
resource "algolia_virtual_index" "price_desc" {
  name                = "products_price_desc"
  primary_index_name  = algolia_index.primary.name
  deletion_protection = false

  ranking {
    custom_ranking = ["desc(price)"]
  }
}

# Standard and virtual replicas share one Algolia setting but are owned
# separately: advanced.replicas owns the standard ones, and each
# algolia_virtual_index owns its own virtual(<name>) entry. Declaring both is
# safe - neither overwrites the other - and naming a virtual replica in
# advanced.replicas is rejected, because that entry is not this list's to own.
resource "algolia_index" "catalog" {
  name                = "catalog"
  deletion_protection = false

  advanced {
    # A standard replica: its own copy of the records, with its own settings.
    replicas = ["catalog_by_name"]
  }
}

resource "algolia_index" "catalog_by_name" {
  name                = "catalog_by_name"
  deletion_protection = false

  ranking {
    custom_ranking = ["asc(name)"]
  }
}

resource "algolia_virtual_index" "catalog_price_asc" {
  name                = "catalog_price_asc"
  primary_index_name  = algolia_index.catalog.name
  deletion_protection = false

  ranking {
    custom_ranking = ["asc(price)"]
  }
}
