# E-commerce search setup

An end-to-end example wiring together the resources a typical storefront needs,
showing how they reference each other:

| Resource | Role |
|---|---|
| `algolia_index.products` | The catalog index: searchable attributes, custom ranking, facets, pagination. |
| `algolia_rule.boost_on_sale` | Merchandising rule that boosts on-sale products for sale-intent queries. |
| `algolia_synonym.tv` | A two-way synonym so "tv", "television", and "telly" all match. |
| `algolia_api_key.frontend_search` | A search-only key, scoped to the index, safe to expose to browser/mobile clients. |

The rule, synonym, and key all reference `algolia_index.products.name`, so the
dependency graph guarantees the index is configured first.

## Usage

```bash
export TF_VAR_algolia_app_id="YourApplicationID"
export TF_VAR_algolia_api_key="your-admin-api-key"   # needs settings + editSettings + admin ACLs

terraform init
terraform apply
```

Retrieve the generated search-only key for your front end:

```bash
terraform output -raw frontend_search_api_key
```

## Notes

- The index only manages **settings**. Populate records separately (Algolia
  auto-creates the index on the first settings write). See the
  [`ingestion-pipeline`](../ingestion-pipeline) example for one way to load
  data declaratively.
- `deletion_protection` is `true` on the index; set it to `false` and apply
  before running `terraform destroy`.
- The `on_sale` attribute is declared `filterOnly(...)` so the merchandising
  rule's `optionalFilters` can reference it without exposing it as a UI facet.
