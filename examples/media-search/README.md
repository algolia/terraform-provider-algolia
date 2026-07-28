# Media search setup

The same building blocks as the [`ecommerce-search`](../ecommerce-search)
example, pointed at a different vertical: a streaming service's title catalog.
Swap the attributes and the storefront becomes a media catalog - the four
resources and how they reference each other are identical.

| Resource | Role |
|---|---|
| `algolia_index.titles` | The title catalog: searchable attributes, custom ranking, facets, pagination. |
| `algolia_rule.boost_originals` | Merchandising rule that surfaces platform originals for browse-intent queries. |
| `algolia_synonym.scifi` | A two-way synonym so "scifi", "sci-fi", and "science fiction" all match. |
| `algolia_api_key.app_search` | A search-only key, scoped to the index, safe to embed in web/TV app clients. |

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
- `attribute_for_distinct = "series_id"` collapses a show's episodes into a
  single hit, so a series shows up once rather than once per episode.
- `deletion_protection` is `true` on the index; set it to `false` and apply
  before running `terraform destroy`.
- The `is_original` attribute is declared `filterOnly(...)` so the
  merchandising rule's `optionalFilters` can reference it without exposing it
  as a UI facet.
