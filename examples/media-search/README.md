# Media / streaming search setup

An end-to-end example wiring together the resources a typical streaming catalog
needs, showing how they reference each other. It mirrors the
[`ecommerce-search`](../ecommerce-search) example on a different vertical: the
same building blocks (index, rule, synonym, scoped key) power a movie/show
catalog just as they power a storefront.

| Resource | Role |
|---|---|
| `algolia_index.titles` | The catalog index: searchable attributes, custom ranking, facets, pagination. `attribute_for_distinct` plus `advanced.distinct = 1` collapses a series' episodes into one hit. |
| `algolia_rule.boost_originals` | Merchandising rule that boosts platform originals for browse/trending queries. |
| `algolia_synonym.scifi` | A two-way synonym so "scifi", "sci-fi", and "science fiction" all match. |
| `algolia_api_key.app_search` | A search-only key, scoped to the index, safe to ship in web/TV app clients. |

The rule, synonym, and key all reference `algolia_index.titles.name`, so the
dependency graph guarantees the index is configured first.

## Usage

```bash
export TF_VAR_algolia_app_id="YourApplicationID"
export TF_VAR_algolia_api_key="your-admin-api-key"   # needs settings + editSettings + admin ACLs

terraform init
terraform apply
```

Retrieve the generated search-only key for your app:

```bash
terraform output -raw app_search_api_key
```

## Notes

- The index only manages **settings**. Populate records separately (Algolia
  auto-creates the index on the first settings write). See the
  [`ingestion-pipeline`](../ingestion-pipeline) example for one way to load
  data declaratively.
- `deletion_protection` is `true` on the index; set it to `false` and apply
  before running `terraform destroy`.
- The `is_original` attribute is declared `filterOnly(...)` so the merchandising
  rule's `optionalFilters` can reference it without exposing it as a UI facet.
