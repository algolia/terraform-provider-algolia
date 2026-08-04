---
name: algolia-terraform-provider
description: Use when installing the Algolia Terraform provider or writing Terraform configuration that manages Algolia resources (indices, replicas, API keys, rules, synonyms, Ingestion tasks, Agent Studio agents). Covers installation, the provider block, and the behaviours that make a plausible-looking configuration fail.
---

# Algolia Terraform provider

For writing Terraform that *uses* this provider.

The authoritative reference for every argument and attribute is the provider's generated
documentation. It is not on the Terraform Registry, so read it in the repository:

- [Resources](https://github.com/algolia/terraform-provider-algolia/tree/main/docs/resources)
- [Data sources](https://github.com/algolia/terraform-provider-algolia/tree/main/docs/data-sources)

Read the page for a resource before writing a configuration for it, because several
attributes do not match the Algolia API's own field names. From a checkout of the provider
the same pages are under `docs/`.

Reference files in this skill, to read when the topic comes up:

| File | When |
| --- | --- |
| `references/attribute-shapes.md` | Writing any non-trivial resource: which settings are JSON strings, which are blocks, and which of the ten index blocks a setting belongs to |
| `references/import-ids.md` | Bringing an existing Algolia object under management |
| `references/errors.md` | An apply, plan or `terraform init` failed |

## Install

Not on the Terraform Registry, so Terraform will not fetch it. With `gh` authenticated:

```bash
gh api -H "Accept: application/vnd.github.raw" \
  repos/algolia/terraform-provider-algolia/contents/scripts/install.sh | bash
```

This puts the release archive in a filesystem mirror, writes a `provider_installation` block
to `~/.terraformrc` unless the file already has one, and prints the version to pin. If it
finds an existing block it changes nothing and prints what to merge in by hand.
[INSTALL.md](https://github.com/algolia/terraform-provider-algolia/blob/main/INSTALL.md)
covers doing it by hand and what to check when `terraform init` cannot find the provider.

Then pin that exact version. A mirror holds only what has been put in it, so a constraint
matching nothing fails rather than downloading:

```hcl
terraform {
  required_providers {
    algolia = {
      source  = "algolia/algolia"
      version = "0.1.2"
    }
  }
}

provider "algolia" {} # reads ALGOLIA_APP_ID and ALGOLIA_API_KEY
```

`ALGOLIA_API_KEY` must be an admin key. Also set `ALGOLIA_ANALYTICS_REGION` to `us` or `eu`
when using Query Suggestions, Personalization, A/B testing or Ingestion; those APIs are
region-routed and fail without it.

## Rules that stop a configuration working

**`deletion_protection` defaults to `true`** on `algolia_index`, `algolia_virtual_index`,
`algolia_agent`, `algolia_api_key` and the five `algolia_ingestion_*` resources. A
`terraform destroy` refuses until the attribute is set to `false` **and applied** in a
separate step. When writing a configuration meant to be torn down, such as an example or a
test fixture, set `deletion_protection = false` from the start.

**A primary index's replicas are owned by two resources.** Standard replicas go in
`algolia_index`'s `advanced.replicas`; each virtual replica is its own
`algolia_virtual_index`. A `virtual(...)` entry in `advanced.replicas` is rejected at plan
time. Do not manage both kinds from one place.

**Reference a replica resource rather than repeating its name**, where an index is managed by
its own `algolia_index` and also listed in another index's `advanced.replicas`:

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

**Attribute shapes are the other common cause of a failed apply**, and they are not
guessable: see `references/attribute-shapes.md` before writing rules, A/B tests, Ingestion
tasks, agents or index settings.

## Checking your work

Run `terraform plan` again after an apply. It must report no changes: this provider reads most
settings back from Algolia, so a second plan that still proposes changes usually means the
configuration and Algolia disagree, and applying repeatedly will not settle it. That is the
cheapest signal that something is wrong, and it catches more than `terraform validate` does.
Some attributes are deliberately not refreshed, because Algolia redacts, enriches or cannot
return them, so drift in those is not reported at all. `references/errors.md` covers the main
ones, and what the common failures mean.
