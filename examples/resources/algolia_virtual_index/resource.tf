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

# If you manage the primary's replicas list yourself, list every virtual replica
# in it too, in the virtual(<name>) form. Both resources write that one Algolia
# setting, and algolia_index writes it as a complete list: a list omitting a
# virtual replica unlinks it, which empties it, since a virtual replica is a view
# over the primary's records rather than a copy. The provider warns while applying
# such a write, so check apply output rather than relying on the plan to flag it.
resource "algolia_index" "catalog" {
  name                = "catalog"
  deletion_protection = false

  advanced {
    replicas = ["virtual(catalog_price_asc)"]
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
