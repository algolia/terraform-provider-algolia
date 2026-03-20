---
name: terraform-algolia-provider
description: Use when writing or modifying Terraform configurations that use the algolia provider to manage Algolia search indices, importing existing indices, or troubleshooting algolia_index resource issues.
---

# Terraform Algolia Provider

The `algolia/algolia` Terraform provider manages Algolia index settings declaratively. Its single resource (`algolia_index`) maps Algolia's `SetSettings`/`GetSettings` API to HCL blocks — every block is optional and only manages the settings you include.

## When to Use

Use this skill when you need to:

- Write a new `algolia_index` resource or data source
- Add or modify settings blocks on an existing index configuration
- Import an existing Algolia index into Terraform state
- Troubleshoot plan diffs, union type mappings, or JSON-encoded fields
- Understand which attributes go in which block

Do **not** use this skill for:

- Algolia dashboard or REST API usage outside Terraform
- Managing Algolia records/objects (the provider only manages index **settings**)
- Synonym or query rule management (not yet supported in v0.1)

## Quick Reference

| Block | Purpose |
|-------|---------|
| `attributes` | Searchable, retrievable, and unretrievable attributes; distinct key |
| `ranking` | Ranking formula, custom ranking, relevancy strictness |
| `faceting` | Facet attributes, max hits/values, sort order |
| `highlighting` | Pre/post tags, snippet config, highlight attributes |
| `pagination` | Hits per page, pagination limit |
| `typos` | Typo tolerance mode, min word sizes, disabled attributes/words |
| `languages` | Index/query languages, plurals, stop words, transliteration, decomposition |
| `query_strategy` | Query type, optional words, exact matching, advanced syntax |
| `performance` | Numeric filtering attributes, integer array compression |
| `advanced` | Distinct, replicas, separators, response fields, neural/keyword mode, JSON fields |

## Provider Configuration

```hcl
terraform {
  required_providers {
    algolia = {
      source  = "algolia/algolia"
      version = "~> 0.1"
    }
  }
}

provider "algolia" {
  app_id  = var.algolia_app_id   # or env: ALGOLIA_APP_ID
  api_key = var.algolia_api_key  # or env: ALGOLIA_API_KEY (must be Admin API key)
}
```

Both `app_id` and `api_key` fall back to environment variables. The API key **must** have write permissions (Admin key) — search-only keys will fail with a 403.

## The `algolia_index` Resource

This is the only resource in v0.1. It manages an Algolia index and its settings. There is no separate "create index" API — Algolia auto-creates an index on the first `SetSettings` call.

### Minimal Example

```hcl
resource "algolia_index" "products" {
  name                = "products"
  deletion_protection = false

  attributes {
    searchable_attributes = ["name", "description", "brand"]
  }
}
```

### Full Structure

The resource has **10 nested blocks**, all optional. Only include blocks for settings you want to manage — omitted blocks are ignored (not reset to defaults).

```hcl
resource "algolia_index" "example" {
  name                = "my-index"       # Required. Changing this destroys and recreates.
  deletion_protection = true             # Defaults to true. Must set false before destroy.

  attributes {
    searchable_attributes    = ["name", "description"]
    attributes_to_retrieve   = ["name", "description", "price"]
    unretrievable_attributes = ["internal_id"]
    attribute_for_distinct   = "product_id"
  }

  ranking {
    ranking              = ["typo", "geo", "words", "filters", "proximity", "attribute", "exact", "custom"]
    custom_ranking       = ["desc(popularity)", "asc(price)"]
    relevancy_strictness = 90
  }

  faceting {
    attributes_for_faceting = ["searchable(category)", "brand", "price"]
    max_facet_hits          = 10
    max_values_per_facet    = 100
    sort_facet_values_by    = "count"  # "count" or "alpha"
  }

  highlighting {
    highlight_pre_tag                     = "<em>"
    highlight_post_tag                    = "</em>"
    attributes_to_highlight               = ["name", "description"]
    attributes_to_snippet                 = ["description:20"]
    snippet_ellipsis_text                 = "..."
    restrict_highlight_and_snippet_arrays = false
  }

  pagination {
    hits_per_page         = 20
    pagination_limited_to = 1000
  }

  typos {
    typo_tolerance                       = "true"  # "true", "false", "min", or "strict"
    min_word_size_for_1_typo             = 4
    min_word_size_for_2_typos            = 8
    allow_typos_on_numeric_tokens        = true
    disable_typo_tolerance_on_attributes = ["sku"]
    disable_typo_tolerance_on_words      = ["iphone"]
  }

  languages {
    index_languages               = ["en", "fr"]
    query_languages               = ["en"]
    ignore_plurals                = true          # bool — applies to all languages
    # ignore_plurals_languages    = ["en", "fr"]  # OR list specific languages (not both)
    remove_stop_words             = false
    # remove_stop_words_languages = ["en"]        # OR list specific languages (not both)
    decompound_query              = true
    remove_words_if_no_results    = "none"  # "none", "lastWords", "firstWords", "allOptional"
    camel_case_attributes         = ["productName"]
    keep_diacritics_on_characters = ""
    attributes_to_transliterate   = ["name", "description"]
  }

  query_strategy {
    query_type                  = "prefixLast"  # "prefixLast", "prefixAll", "prefixNone"
    advanced_syntax             = false
    advanced_syntax_features    = ["exactPhrase", "excludeWords"]
    optional_words              = ["the", "a"]
    disable_prefix_on_attributes = ["sku"]
    disable_exact_on_attributes = ["description"]
    exact_on_single_word_query  = "attribute"  # "attribute", "none", "word"
    alternatives_as_exact       = ["ignorePlurals", "singleWordSynonym"]
  }

  performance {
    numeric_attributes_for_filtering  = ["price", "quantity"]
    allow_compression_of_integer_array = false
  }

  advanced {
    distinct                                = 1     # 0=off, 1=dedupe, 2+=group size
    min_proximity                           = 1
    replace_synonyms_in_highlight           = false
    separators_to_index                     = "+#"
    response_fields                         = ["hits", "nbHits"]
    enable_rules                            = true
    enable_personalization                   = false
    replicas                                = ["products_price_asc"]
    enable_re_ranking                       = false
    mode                                    = "keywordSearch"  # or "neuralSearch"
    attribute_criteria_computed_by_min_proximity = false
    # JSON-encoded fields:
    user_data         = jsonencode({ version = "2.0" })
    semantic_search   = jsonencode({ eventSources = ["order"] })
    re_ranking_apply_filter = jsonencode("brand:Apple")
  }
}
```

### Data Source

Read-only. Same blocks, all computed:

```hcl
data "algolia_index" "existing" {
  name = "my-existing-index"
}

output "hits_per_page" {
  value = data.algolia_index.existing.pagination[0].hits_per_page
}
```

### Import

```bash
terraform import algolia_index.products my-index-name
```

The index name is the import ID. All settings are populated on import. `deletion_protection` defaults to `true` after import.

## Union Type Mappings

The Algolia API has union types that are mapped to simpler Terraform types:

| API Field | Terraform Type | Mapping |
|-----------|---------------|---------|
| `typoTolerance` (bool\|enum) | `types.String` | `"true"`, `"false"`, `"min"`, `"strict"` |
| `distinct` (bool\|int) | `types.Int64` | `0`=off, `1`=dedupe, `2`-`4`=group size |
| `ignorePlurals` (bool\|[]lang) | Split: `ignore_plurals` (Bool) + `ignore_plurals_languages` (List) |
| `removeStopWords` (bool\|[]lang) | Split: `remove_stop_words` (Bool) + `remove_stop_words_languages` (List) |

For `ignorePlurals` and `removeStopWords`: use **either** the bool field **or** the languages list, not both. Languages take precedence during expand.

## JSON-Encoded Fields

These complex nested types are stored as JSON strings in Terraform state. Use `jsonencode()` in HCL:

- `decompounded_attributes` — e.g., `jsonencode({ de = ["attr1", "attr2"] })`
- `custom_normalization` — e.g., `jsonencode({ default = { a = "b" } })`
- `user_data` — arbitrary JSON
- `semantic_search` — e.g., `jsonencode({ eventSources = ["order"] })`
- `re_ranking_apply_filter` — string or nested array filter

## Common Mistakes

**Setting `deletion_protection` to `true` (the default) then running `terraform destroy`.**
The destroy will fail. Fix: set `deletion_protection = false` and run `terraform apply` before destroying.

**Renaming an index by changing `name`.**
This destroys and recreates the index, causing data loss. Fix: create a new index, migrate data, then remove the old one.

**Using a search-only API key.**
Write operations return 403. Fix: use an Admin API key or one with `editSettings` ACL.

**Setting both `ignore_plurals` and `ignore_plurals_languages` (same for `remove_stop_words`).**
Only one form is supported at a time. Fix: use the bool to apply to all languages, or the list to target specific ones — never both.

**Passing raw JSON strings instead of `jsonencode()`.**
Raw strings cause persistent diffs because formatting may differ from what Algolia returns. Fix: always use `jsonencode()`.

**Expecting `relevancy_strictness` or `allow_compression_of_integer_array` to appear after import.**
These are write-only API fields — the API accepts them but `GetSettings` doesn't return them. Fix: add them to your config manually after import; the provider preserves them in state.

**Using snake_case for enum values.**
Algolia enum values are camelCase: `"prefixLast"`, `"lastWords"`, `"neuralSearch"`, `"allOptional"`. Fix: use the exact camelCase strings.

**Assuming omitted blocks reset to defaults.**
Omitted blocks are not managed — Terraform won't touch those settings. Fix: explicitly include a block with default values if you want to reset settings.

## Computed Metadata

These read-only attributes are available after apply:

- `primary` — name of the primary index (for replicas)
- `entries` — number of records
- `data_size` — index size in bytes
- `created_at` / `updated_at` — timestamps
