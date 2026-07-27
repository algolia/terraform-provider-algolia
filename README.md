<div align="center">

# Terraform Provider for Algolia

Manage your [Algolia](https://www.algolia.com/) search infrastructure as code.

[![Status](https://img.shields.io/badge/status-internal%20beta-orange.svg)](https://github.com/algolia/terraform-provider-algolia/releases)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL_2.0-blue.svg)](LICENSE)

</div>

## Overview

The Algolia Terraform provider lets you configure and manage Algolia resources declaratively: index settings, API keys, query suggestions, personalization strategies, rules, synonyms, Agent Studio agents, and the full Ingestion pipeline (authentications, sources, destinations, transformations, and tasks).

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.0
- [Go](https://go.dev/doc/install) >= 1.25 (for building from source)

## Installation (internal)

> **This provider is distributed internally only.** It is not published to the public Terraform Registry. Every internal developer installs it from the signed release artifacts on [GitHub Releases](https://github.com/algolia/terraform-provider-algolia/releases) using a local [filesystem mirror](https://developer.hashicorp.com/terraform/cli/config/config-file#filesystem_mirror). No private registry is required.

**1. Download the release archive for your platform.** Grab the `terraform-provider-algolia_<version>_<os>_<arch>.zip` for the version you want from the [releases page](https://github.com/algolia/terraform-provider-algolia/releases) (`darwin_arm64`, `darwin_amd64`, `linux_amd64`, `linux_arm64`, `windows_amd64`, or `windows_arm64`).

**2. Drop it into your filesystem mirror** (do **not** unzip it; the mirror uses the packed layout):

```bash
# macOS / Linux
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/algolia/algolia
mv ~/Downloads/terraform-provider-algolia_0.1.0-beta.1_darwin_arm64.zip \
   ~/.terraform.d/plugins/registry.terraform.io/algolia/algolia/
```

On Windows the mirror directory is `%APPDATA%\terraform.d\plugins\registry.terraform.io\algolia\algolia\`.

**3. Tell Terraform to resolve this provider from the mirror** by adding a `provider_installation` block to your CLI config (`~/.terraformrc` on macOS/Linux, `%APPDATA%\terraform.rc` on Windows). This keeps every other provider coming from the public registry while pulling `algolia/algolia` from disk:

```hcl
provider_installation {
  filesystem_mirror {
    path    = "/Users/<you>/.terraform.d/plugins"
    include = ["registry.terraform.io/algolia/algolia"]
  }
  direct {
    exclude = ["registry.terraform.io/algolia/algolia"]
  }
}
```

**4. (Optional) Verify the download.** Each release ships a GPG-signed `SHA256SUMS` (`SHA256SUMS.sig`). Filesystem-mirror installs are not signature-checked by Terraform, so verify manually if you need the assurance:

```bash
shasum -a 256 -c terraform-provider-algolia_0.1.0-beta.1_SHA256SUMS 2>&1 | grep OK
```

Then run `terraform init`, and Terraform will pick up the mirrored provider without contacting the network.

## While still internally available

Until the provider is published to a registry, the fastest way to try it (or test an unreleased build) is to install it straight from GitHub with `go install` and point Terraform at the resulting binary with a [`dev_overrides`](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers) block. No release download, no filesystem mirror, and no `terraform init`.

**Prerequisites:** Go >= 1.25 and Git access to this (internal) repository. Because the repo is private, `go` must be able to authenticate to GitHub. If you clone Algolia repos over SSH, tell `go` to use SSH for `github.com` too (one time):

```zsh
git config --global url."git@github.com:".insteadOf "https://github.com/"
```

(Alternatively, `gh auth setup-git` configures Git to authenticate over HTTPS via the GitHub CLI.)

**1. Install straight from GitHub.** Use a tag, `@main`, or `@latest`:

```zsh
export GOPRIVATE=github.com/algolia/*   # skip the public proxy/checksum DB for the internal repo
go install github.com/algolia/terraform-provider-algolia@v0.1.0-beta.1
```

The binary lands in `$(go env GOPATH)/bin` (typically `~/go/bin`).

**2. Point Terraform at it** with a `dev_overrides` block in your CLI config (`~/.terraformrc` on macOS/Linux, `%APPDATA%\terraform.rc` on Windows). Use the absolute path to your `GOPATH/bin` directory (the CLI config does not expand `~`):

```hcl
provider_installation {
  dev_overrides {
    "algolia/algolia" = "/Users/you/go/bin"
  }
  # everything else installs from the public registry as normal
  direct {}
}
```

**3. Plan or apply directly (no `terraform init` needed):**

```hcl
terraform {
  required_providers {
    algolia = { source = "algolia/algolia" }   # no version constraint under dev_overrides
  }
}

provider "algolia" {
  app_id  = var.algolia_app_id
  api_key = var.algolia_api_key
}
```

```zsh
terraform plan
```

Terraform prints a `Provider development overrides are in effect` warning on every command. That is expected, and it confirms Terraform is using your local binary instead of a registry.

> **Working on the provider itself?** Swap step 1 for `make build` in a local checkout and set the `dev_overrides` path to the repo root. Same flow, no GitHub fetch.

## Getting Started

The provider is currently a pre-release (`0.1.0-beta.1`). Terraform excludes pre-release versions from range constraints (e.g. `~> 0.1` will **not** match a `-beta` build), so pin the exact version:

```hcl
terraform {
  required_providers {
    algolia = {
      source  = "algolia/algolia"
      version = "0.1.0-beta.1"
    }
  }
}

provider "algolia" {
  app_id  = var.algolia_app_id   # or ALGOLIA_APP_ID env var
  api_key = var.algolia_api_key  # or ALGOLIA_API_KEY env var
}

variable "algolia_app_id" {
  type = string
}

variable "algolia_api_key" {
  type      = string
  sensitive = true
}

resource "algolia_index" "products" {
  name                = "products"
  deletion_protection = true

  attributes {
    searchable_attributes  = ["name", "description", "categories"]
    attributes_to_retrieve = ["name", "price", "image_url"]
    attribute_for_distinct  = "product_id"
  }

  ranking {
    custom_ranking = ["desc(popularity)", "asc(price)"]
  }

  faceting {
    attributes_for_faceting = ["searchable(categories)", "brand", "price"]
  }

  typos {
    typo_tolerance           = "true"
    min_word_size_for_1_typo = 4
  }

  advanced {
    distinct = 1
  }
}
```

## Authentication

| Attribute          | Environment Variable       | Description                                                     |
| ------------------ | -------------------------- | --------------------------------------------------------------- |
| `app_id`           | `ALGOLIA_APP_ID`           | Algolia Application ID                                          |
| `api_key`          | `ALGOLIA_API_KEY`          | Algolia Admin API Key                                           |
| `analytics_region` | `ALGOLIA_ANALYTICS_REGION` | Region for Query Suggestions and Personalization (`us` or `eu`) |

## Resources

| Resource                                                                         | Description                                                                   |
| -------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| [`algolia_index`](docs/resources/index.md)                                       | Manage index settings (searchable attributes, ranking, faceting, typos, etc.) |
| [`algolia_virtual_index`](docs/resources/virtual_index.md)                       | Manage virtual replica index settings                                         |
| [`algolia_api_key`](docs/resources/api_key.md)                                   | Create scoped API keys with specific permissions                              |
| [`algolia_rule`](docs/resources/rule.md)                                         | Configure index query rules                                                   |
| [`algolia_synonym`](docs/resources/synonym.md)                                   | Manage synonym entries on an index                                            |
| [`algolia_dictionary_entry`](docs/resources/dictionary_entry.md)                 | Manage custom dictionary entries (stopwords, plurals, compounds)              |
| [`algolia_dictionary_settings`](docs/resources/dictionary_settings.md)           | Manage which built-in standard dictionary entries are disabled, per language  |
| [`algolia_allowed_sources`](docs/resources/allowed_sources.md)                   | Manage the application's allowed source IP addresses/CIDR ranges (allowlist)  |
| [`algolia_query_suggestions`](docs/resources/query_suggestions.md)               | Configure Query Suggestions                                                   |
| [`algolia_personalization_strategy`](docs/resources/personalization_strategy.md) | Set the app-level Personalization strategy                                    |
| [`algolia_agent_provider`](docs/resources/agent_provider.md)                     | Register LLM providers for Agent Studio                                       |
| [`algolia_agent`](docs/resources/agent.md)                                       | Create and configure Agent Studio agents                                      |
| [`algolia_ingestion_authentication`](docs/resources/ingestion_authentication.md) | Manage Ingestion authentication credentials for sources/destinations          |
| [`algolia_ingestion_source`](docs/resources/ingestion_source.md)                 | Manage Ingestion sources (where a task reads or receives records from)        |
| [`algolia_ingestion_destination`](docs/resources/ingestion_destination.md)       | Manage Ingestion destinations (the index or event stream a task writes to)    |
| [`algolia_ingestion_transformation`](docs/resources/ingestion_transformation.md) | Manage Ingestion transformations (code or no-code logic applied to records)   |
| [`algolia_ingestion_task`](docs/resources/ingestion_task.md)                     | Manage Ingestion tasks (schedule a source → transform → destination pipeline) |
| [`algolia_recommend_rule`](docs/resources/recommend_rule.md)                     | Manage Recommend rules (condition/consequence pairs that customize a Recommend model's results) |
| [`algolia_composition`](docs/resources/composition.md)                           | Manage a Composition (combine source indices into one search experience) |
| [`algolia_composition_rule`](docs/resources/composition_rule.md)                 | Manage a Composition rule (condition/consequence pairs on a composition) |

## Data Sources

| Data Source                                                                         | Description                                        |
| ----------------------------------------------------------------------------------- | -------------------------------------------------- |
| [`algolia_index`](docs/data-sources/index.md)                                       | Read existing index settings                       |
| [`algolia_virtual_index`](docs/data-sources/virtual_index.md)                       | Read virtual replica settings                      |
| [`algolia_indices`](docs/data-sources/indices.md)                                   | List every index in the application                |
| [`algolia_api_key`](docs/data-sources/api_key.md)                                   | Read an API key's metadata by its key value         |
| [`algolia_api_keys`](docs/data-sources/api_keys.md)                                 | List every API key configured for the application  |
| [`algolia_rule`](docs/data-sources/rule.md)                                         | Read a rule by index and object ID                 |
| [`algolia_synonym`](docs/data-sources/synonym.md)                                   | Read a synonym by index and object ID              |
| [`algolia_dictionary_entry`](docs/data-sources/dictionary_entry.md)                 | Read a dictionary entry by dictionary and object ID |
| [`algolia_dictionary_settings`](docs/data-sources/dictionary_settings.md)           | Read which built-in standard dictionary entries are disabled |
| [`algolia_allowed_sources`](docs/data-sources/allowed_sources.md)                   | Read the application's allowed source IP addresses/CIDR ranges (allowlist) |
| [`algolia_query_suggestions`](docs/data-sources/query_suggestions.md)               | Read a Query Suggestions configuration             |
| [`algolia_personalization_strategy`](docs/data-sources/personalization_strategy.md) | Read the Personalization strategy                  |
| [`algolia_agent_provider`](docs/data-sources/agent_provider.md)                     | Read Agent Studio provider details                 |
| [`algolia_agent_provider_models`](docs/data-sources/agent_provider_models.md)       | List models available for an Agent Studio provider |
| [`algolia_agent`](docs/data-sources/agent.md)                                       | Read Agent Studio agent details                    |
| [`algolia_ingestion_authentication`](docs/data-sources/ingestion_authentication.md) | Read Ingestion authentication metadata (credentials are never exposed) |
| [`algolia_ingestion_source`](docs/data-sources/ingestion_source.md)                 | Read an Ingestion source's configuration, including `input`        |
| [`algolia_ingestion_destination`](docs/data-sources/ingestion_destination.md)       | Read an Ingestion destination's configuration, including `input`   |
| [`algolia_ingestion_transformation`](docs/data-sources/ingestion_transformation.md) | Read an Ingestion transformation's configuration, including `code`/`input` |
| [`algolia_ingestion_task`](docs/data-sources/ingestion_task.md)                     | Read an Ingestion task's configuration, including `input`/`notifications`/`policies`/`cursor` |
| [`algolia_recommend_rule`](docs/data-sources/recommend_rule.md)                     | Read a Recommend rule by index, model, and object ID |
| [`algolia_composition`](docs/data-sources/composition.md)                           | Read a Composition by object ID |
| [`algolia_composition_rule`](docs/data-sources/composition_rule.md)                 | Read a Composition rule by composition and object ID |
| [`algolia_clusters`](docs/data-sources/clusters.md)                                 | List every cluster in a multi-cluster (MCM) application |
| [`algolia_user_ids`](docs/data-sources/user_ids.md)                                 | List every user ID mapped to a cluster in a multi-cluster (MCM) application |

## API Coverage

How far the provider reaches across Algolia's API surface. The three columns are a dependency chain — each stage depends on the one before it:

- **Public API** — does Algolia expose a public REST API for this feature? Some features are dashboard-only (e.g. Collections) and have no API at all, so nothing downstream is possible. These can't be discovered from the OpenAPI specs, so they're tracked here manually.
- **Go v4 client** — does the official [`algoliasearch-client-go/v4`](https://github.com/algolia/algoliasearch-client-go) client wrap it? Where it doesn't, the provider must ship a hand-rolled HTTP client (as it already does for Agent Studio).
- **Terraform** — does this provider support it?

| API surface / feature | Public API | Go v4 client | Terraform |
| --- | :---: | :---: | --- |
| **Search** — index settings | ✅ | ✅ | ✅ `algolia_index` |
| **Search** — virtual (replica) index | ✅ | ✅ | ✅ `algolia_virtual_index` |
| **Search** — index listing | ✅ | ✅ | ✅ `algolia_indices` (data source) |
| **Search** — API keys | ✅ | ✅ | ✅ `algolia_api_key` (+ `algolia_api_keys` data source) |
| **Search** — rules | ✅ | ✅ | ✅ `algolia_rule` |
| **Search** — synonyms | ✅ | ✅ | ✅ `algolia_synonym` |
| **Search** — dictionaries (custom entries) | ✅ | ✅ | ✅ `algolia_dictionary_entry` |
| **Search** — dictionaries (settings) | ✅ | ✅ | ✅ `algolia_dictionary_settings` |
| **Search** — allowed sources (IP allowlist) | ✅ | ✅ | ✅ `algolia_allowed_sources` |
| **Search** — MCM clusters & user IDs | ✅ | ✅ | ✅ `algolia_clusters`, `algolia_user_ids` (data sources) |
| **Search** — records · search · browse | ✅ | ✅ | ⛔ Out of scope (data-plane) |
| **Query Suggestions** | ✅ | ✅ | ✅ `algolia_query_suggestions` |
| **Personalization** — strategy | ✅ | ✅ | ✅ `algolia_personalization_strategy` |
| **Advanced Personalization** | ✅ | ❌ not in v4 | 🟡 Planned · P3 (investigate) |
| **Agent Studio** — agents & providers | ✅ | ❌ custom client | ✅ `algolia_agent`, `algolia_agent_provider` |
| **Ingestion** — authentications | ✅ | ✅ | ✅ `algolia_ingestion_authentication` |
| **Ingestion** — sources | ✅ | ✅ | ✅ `algolia_ingestion_source` |
| **Ingestion** — destinations | ✅ | ✅ | ✅ `algolia_ingestion_destination` |
| **Ingestion** — transformations | ✅ | ✅ | ✅ `algolia_ingestion_transformation` |
| **Ingestion** — tasks | ✅ | ✅ | ✅ `algolia_ingestion_task` |
| **A/B Testing** | ✅ | ✅ | ✅ `algolia_ab_test` |
| **Recommend** — rules | ✅ | ✅ | ✅ `algolia_recommend_rule` |
| **Composition** | ✅ | ✅ | ✅ `algolia_composition`, `algolia_composition_rule` |
| **Crawler** | ✅ | ❌ not in v4 | 🟡 Planned · P3 (needs custom client) |
| **Analytics** | ✅ | ✅ | 🟡 Planned · P4 (read-only) |
| **Monitoring** | ✅ | ✅ | 🟡 Planned · P4 (read-only) |
| **Insights** (events) | ✅ | ✅ | ⛔ Out of scope (runtime) |
| **Collections** _(example: dashboard-only)_ | ❌ none | ❌ | ❌ No public API |

**Legend:** ✅ available · 🟡 planned (phase) · ⛔ out of scope (runtime / data-plane) · ❌ not available

See [ROADMAP.md](ROADMAP.md) for the phased plan and scope rationale behind these statuses.

## Importing Existing Resources

Bring existing Algolia resources under Terraform management:

```bash
terraform import algolia_index.products products
terraform import algolia_api_key.search AB1CD2EF3G
terraform import algolia_rule.promo products/my-rule-id
terraform import algolia_synonym.brand products/my-synonym-id
terraform import algolia_dictionary_entry.stopword stopwords/my-entry-id
```

## Contributing

Interested in contributing? See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, build instructions, and testing guidelines.

## License

[Mozilla Public License 2.0](LICENSE)
