# Attribute shapes

The shapes that differ from what Algolia's own API reference shows. Verified against the
v0.1.0 schema; the [generated resource docs](https://github.com/algolia/terraform-provider-algolia/tree/main/docs/resources) are
authoritative.

## JSON strings and nested blocks

Some complex settings are `jsonencode()`d strings and others are nested blocks. **The same
attribute name is not the same shape on every resource**, so check the resource's own docs
page rather than carrying a shape over from a neighbouring resource.

`validity` is the clearest trap: a block on `algolia_rule`, a JSON string on
`algolia_recommend_rule` and `algolia_composition_rule`, and a number of seconds on
`algolia_api_key`.

Blocks rather than JSON: `conditions`, `consequence` and `validity` on `algolia_rule`.

```hcl
resource "algolia_rule" "promo" {
  index_name = algolia_index.products.name
  object_id  = "promo"

  conditions {
    pattern   = "apple"
    anchoring = "contains"
  }

  consequence {
    params_json = jsonencode({ filters = "brand:apple" }) # JSON, inside a block
  }
}
```

JSON-encoded attributes, by resource:

| Resource | JSON-encoded attributes |
| --- | --- |
| `algolia_index`, `algolia_virtual_index` | `custom_normalization`, `decompounded_attributes`, `semantic_search`, `re_ranking_apply_filter`, `user_data` |
| `algolia_rule` | `consequence.params_json`, `consequence.user_data` |
| `algolia_recommend_rule` | `condition`, `consequence`, `validity` |
| `algolia_composition_rule` | `conditions`, `consequence`, `validity` |
| `algolia_ab_test` | `variants`, `metrics`, `configuration` |
| `algolia_ingestion_task` | `input`, `notifications`, `policies` |
| `algolia_ingestion_source`, `algolia_ingestion_destination`, `algolia_ingestion_authentication`, `algolia_ingestion_transformation` | `input` |
| `algolia_agent` | `config`, `input_schema`, `search_parameters`, and `predefined_recommend_parameters` inside the `tool_algolia_recommend` block |
| `algolia_composition` | `behavior` (**required**), `sorting_strategy` |

`input` on `algolia_synonym` is *not* JSON: it is the base word of a one-way synonym.

## Flattened union types

Algolia's union-typed settings are flattened, so they do not take the shape the API
reference shows.

- `typo_tolerance` is a string taking `"true"`, `"false"`, `"min"` or `"strict"`.
- `distinct` is an integer: 0 disables it, 1 means one result per value, 2 or more sets a
  group size.
- The two settings accepting either a bool or a language list are each split in half:
  `ignore_plurals` plus `ignore_plurals_languages`, and `remove_stop_words` plus
  `remove_stop_words_languages`.

## Index settings live inside one of ten blocks

Index settings are not top-level attributes, and which block holds a given setting is not
guessable from its name, so look it up in the
[`algolia_index` reference](https://github.com/algolia/terraform-provider-algolia/blob/main/docs/resources/index.md) rather than assuming.
The blocks are `advanced`, `attributes`, `faceting`, `highlighting`, `languages`,
`pagination`, `performance`, `query_strategy`, `ranking` and `typos`.

The settings named above land in three different blocks:

```hcl
resource "algolia_index" "products" {
  name = "products"

  advanced {
    distinct = 1 # also: replicas, user_data, semantic_search, re_ranking_apply_filter
  }

  languages {
    ignore_plurals    = true # also: remove_stop_words, custom_normalization,
    remove_stop_words = true # decompounded_attributes, and both *_languages lists
  }

  typos {
    typo_tolerance = "min" # not in `advanced`, despite reading like an advanced setting
  }
}
```
