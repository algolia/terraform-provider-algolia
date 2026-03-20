---
name: terraform-algolia-provider
description: Use when writing or modifying Terraform configurations that use the algolia provider to manage Algolia search indices, importing existing indices, or troubleshooting plan diffs, union type mappings, or JSON-encoded field issues.
---

# Terraform Algolia Provider

The `algolia/algolia` Terraform provider manages Algolia index settings declaratively. Its single resource (`algolia_index`) maps Algolia's `SetSettings`/`GetSettings` API to HCL blocks — every block is optional and only manages the settings you include.

## When to Use

- Write or modify `algolia_index` resources or data sources
- Import an existing Algolia index into Terraform state
- Troubleshoot plan diffs, union type mappings, or JSON-encoded fields

**Not for:** Algolia dashboard/REST API outside Terraform, managing records/objects, synonyms or query rules (not yet in v0.1).

## Quick Reference

| Block | Purpose |
|-------|---------|
| `attributes` | Searchable, retrievable, unretrievable attributes; distinct key |
| `ranking` | Ranking formula, custom ranking, relevancy strictness |
| `faceting` | Facet attributes, max hits/values, sort order |
| `highlighting` | Pre/post tags, snippet config, highlight attributes |
| `pagination` | Hits per page, pagination limit |
| `typos` | Typo tolerance mode, min word sizes, disabled attributes/words |
| `languages` | Index/query languages, plurals, stop words, transliteration, decomposition |
| `query_strategy` | Query type, optional words, exact matching, advanced syntax |
| `performance` | Numeric filtering attributes, integer array compression |
| `advanced` | Distinct, replicas, separators, response fields, neural/keyword mode, JSON fields |

## Provider Setup

Credentials via HCL args (`app_id`, `api_key`) or env vars (`ALGOLIA_APP_ID`, `ALGOLIA_API_KEY`). The API key **must** have write permissions (Admin key or `editSettings` ACL).

## Resource Structure

The resource has **10 nested blocks**, all optional. Only include blocks for settings you want to manage — omitted blocks are ignored (not reset to defaults). See `full-example.tf` in this directory for every attribute.

```hcl
resource "algolia_index" "products" {
  name                = "products"
  deletion_protection = false

  attributes {
    searchable_attributes = ["name", "description", "brand"]
  }
}
```

## Import

```bash
terraform import algolia_index.products my-index-name
```

After import, only `name` and `deletion_protection` are in state — all setting blocks are null. Blocks are populated on the next `terraform apply` when the config is present. `deletion_protection` defaults to `true`.

## Union Type Mappings

| API Field | Terraform Type | Mapping |
|-----------|---------------|---------|
| `typoTolerance` (bool\|enum) | `types.String` | `"true"`, `"false"`, `"min"`, `"strict"` |
| `distinct` (bool\|int) | `types.Int64` | `0`=off, `1`=dedupe, `2`-`4`=group size |
| `ignorePlurals` (bool\|[]lang) | Split: `ignore_plurals` (Bool) + `ignore_plurals_languages` (List) |
| `removeStopWords` (bool\|[]lang) | Split: `remove_stop_words` (Bool) + `remove_stop_words_languages` (List) |

For `ignorePlurals`/`removeStopWords`: use **either** the bool **or** the languages list, not both. Languages take precedence during expand.

## JSON-Encoded Fields

Use `jsonencode()` in HCL — raw strings cause persistent diffs:

- `decompounded_attributes` — e.g., `jsonencode({ de = ["attr1", "attr2"] })`
- `custom_normalization` — e.g., `jsonencode({ default = { a = "b" } })`
- `user_data` — arbitrary JSON
- `semantic_search` — e.g., `jsonencode({ eventSources = ["order"] })`
- `re_ranking_apply_filter` — string or nested array filter

## Common Mistakes

- **`deletion_protection = true` (default) + `terraform destroy`** — destroy fails. Set to `false` and apply first.
- **Changing `name`** — destroys and recreates (data loss). Create new index, migrate, remove old.
- **Search-only API key** — 403 on writes. Use Admin key or `editSettings` ACL.
- **Both `ignore_plurals` and `ignore_plurals_languages`** — use one form only.
- **Raw JSON strings** — use `jsonencode()` to avoid formatting diffs.
- **snake_case enum values** — Algolia uses camelCase: `"prefixLast"`, `"lastWords"`, `"neuralSearch"`.
- **Assuming omitted blocks reset to defaults** — omitted blocks are unmanaged. Include block with default values to reset.

## Computed Metadata

Read-only after apply: `primary`, `entries`, `data_size`, `created_at`, `updated_at`.
