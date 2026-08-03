---
name: algolia-terraform-provider
description: Use when installing the Algolia Terraform provider or writing Terraform configuration that manages Algolia resources (indices, replicas, API keys, rules, synonyms, Ingestion tasks, Agent Studio agents). Covers installation, the provider block, and the behaviours that make a plausible-looking configuration fail.
---

# Algolia Terraform provider

For writing Terraform that *uses* this provider. If you are changing the provider's own Go
code, read `AGENTS.md` in the provider repository instead.

The authoritative reference for every argument and attribute is the provider's `docs/`
directory: `docs/resources/<name>.md` and `docs/data-sources/<name>.md`. Read the page for a
resource before writing a configuration for it rather than guessing attribute names, because
several attributes do not match the Algolia API's own field names.

## Install

Not on the Terraform Registry, so Terraform will not fetch it. With `gh` authenticated:

```bash
gh api -H "Accept: application/vnd.github.raw" \
  repos/algolia/terraform-provider-algolia/contents/scripts/install.sh | bash
```

This puts the release archive in a filesystem mirror and writes a `provider_installation`
block to `~/.terraformrc`. It prints the version to pin. `INSTALL.md` in the repository
covers doing it by hand.

Then pin that exact version. A mirror holds only what has been put in it, so a constraint
matching nothing fails instead of downloading:

```hcl
terraform {
  required_providers {
    algolia = {
      source  = "algolia/algolia"
      version = "0.1.0"
    }
  }
}

provider "algolia" {} # reads ALGOLIA_APP_ID and ALGOLIA_API_KEY
```

`ALGOLIA_API_KEY` must be an admin key. Set `ALGOLIA_ANALYTICS_REGION` to `us` or `eu` as
well when using Query Suggestions, Personalization, A/B testing or Ingestion; those APIs are
region-routed and fail without it.

## Rules that stop a configuration working

These are the ones that make correct-looking Terraform fail.

**`deletion_protection` defaults to `true`** on `algolia_index`, `algolia_virtual_index`,
`algolia_agent`, `algolia_api_key` and the five `algolia_ingestion_*` resources. A
`terraform destroy` refuses until the attribute is set to `false` **and applied** in a
separate step. When writing a configuration meant to be torn down, such as an example or a
test fixture, set `deletion_protection = false` from the start.

**A primary index's replicas are owned by two resources.** Standard replicas go in
`algolia_index`'s `advanced.replicas`; each virtual replica is its own
`algolia_virtual_index`. Putting a `virtual(...)` entry in `advanced.replicas` is rejected at
plan time. Do not try to manage both kinds from one place.

**Reference a replica resource, do not repeat its name.** If an index is managed by its own
`algolia_index` and also listed in another index's `advanced.replicas`:

```hcl
resource "algolia_index" "primary" {
  name = "products"
  advanced {
    replicas = [algolia_index.price_asc.name] # not "products_price_asc"
  }
}

resource "algolia_index" "price_asc" {
  name = "products_price_asc"
}
```

The reference makes Terraform order the two writes. A literal string leaves them concurrent,
which costs about thirty seconds on create and can fail a destroy.

**Some complex settings are JSON strings written with `jsonencode()`, and others are nested
blocks. The same attribute name is not the same shape on every resource**, so check the
resource's own docs page rather than carrying a shape over from a neighbouring resource.
`validity` is the clearest trap: a block on `algolia_rule`, a JSON string on
`algolia_recommend_rule` and `algolia_composition_rule`, and a number of seconds on
`algolia_api_key`.

Blocks, not JSON: `conditions`, `consequence` and `validity` on `algolia_rule`.

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

JSON strings, by resource:

| Resource | JSON-encoded attributes |
| --- | --- |
| `algolia_index`, `algolia_virtual_index` | `custom_normalization`, `decompounded_attributes`, `semantic_search`, `re_ranking_apply_filter`, `user_data` |
| `algolia_rule` | `consequence.params_json`, `consequence.user_data` |
| `algolia_recommend_rule` | `condition`, `consequence`, `validity` |
| `algolia_composition_rule` | `conditions`, `consequence`, `validity` |
| `algolia_ab_test` | `variants`, `metrics`, `configuration` |
| `algolia_ingestion_task` | `input`, `notifications`, `policies` |
| `algolia_ingestion_source`, `_destination`, `_authentication`, `_transformation` | `input` |
| `algolia_agent` | `config`, `input_schema`, `search_parameters` |

`input` on `algolia_synonym` is *not* JSON: it is the base word of a one-way synonym.

**Algolia's union-typed settings are flattened**, so they do not take the shape the API
reference shows. `typo_tolerance` is a string taking `"true"`, `"false"`, `"min"` or
`"strict"`. `distinct` is an integer where 0 disables it, 1 means one result per value and 2
or more sets a group size. The two settings that accept either a bool or a language list are
each split in half: `ignore_plurals` plus `ignore_plurals_languages`, and `remove_stop_words`
plus `remove_stop_words_languages`.

**Index settings sit inside one of ten blocks, not at the top level**, and which block is not
guessable from the setting's name, so look it up in `docs/resources/index.md` instead of
assuming. The blocks are `advanced`, `attributes`, `faceting`, `highlighting`, `languages`,
`pagination`, `performance`, `query_strategy`, `ranking` and `typos`. The ones named above
land in three different blocks:

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

**Import IDs differ per resource, and there is no rule covering all of them.** Every
resource has an `## Import` section in its docs page with a worked example; read it rather
than guessing. The shapes in use:

```bash
terraform import algolia_index.products products                  # the object's own name
terraform import algolia_rule.promo products/my-rule              # <index>/<object_id>
terraform import algolia_recommend_rule.hide products/related-products/hide  # <index>/<model>/<object_id>
terraform import algolia_agent.support 01234567-89ab-cdef-0123-456789abcdef  # a UUID
terraform import algolia_ab_test.pricing 42                       # a number
terraform import algolia_allowed_sources.main YourApplicationID   # the application id
```

The UUID form covers `algolia_agent`, `algolia_agent_provider` and all five
`algolia_ingestion_*` resources; those ids are only discoverable from the Algolia
dashboard or API, not from the object's name. `algolia_api_key`'s import id is the key
value itself.

## What an error means

| Message | Cause |
| --- | --- |
| `no available releases match the given constraints` | The pinned version is not in the mirror. Check what `install.sh` installed. |
| `Deletion Protection Enabled` | Set `deletion_protection = false` and apply before destroying. |
| `Virtual replica declared on the wrong resource` | A `virtual(...)` entry is in `advanced.replicas`. Use `algolia_virtual_index`. |
| `Index is a standard replica` | Algolia holds the index as a standard replica, which copies records, so `algolia_virtual_index` will not adopt it. |
| `Index still exists after deletion` | Algolia accepted the delete but the index survives, usually because an A/B test references it. Destroy again once nothing does. |
| `401 The log processing region does not match` | `ALGOLIA_ANALYTICS_REGION` is wrong for the application. |
| `cannot apply the deleteIndex operation on a replica index` | The index is still listed as a replica of a primary that is not going away. |

## Checking your work

Run `terraform plan` again after an apply. It must report no changes: this provider reads
every setting back from Algolia, so a second plan that still proposes changes means the
configuration and Algolia disagree, and applying repeatedly will not settle it. That is the
cheapest signal that something is wrong, and it catches more than `terraform validate` does.
