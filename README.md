<div align="center">

# Terraform Provider for Algolia

Manage your [Algolia](https://www.algolia.com/) search infrastructure as code.

[![Terraform Registry](https://img.shields.io/badge/Terraform-Registry-purple.svg)](https://registry.terraform.io/providers/algolia/algolia/latest)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL_2.0-blue.svg)](LICENSE)

</div>

## Overview

The Algolia Terraform provider lets you configure and manage Algolia resources declaratively: index settings, API keys, query suggestions, personalization strategies, rules, synonyms, Agent Studio agents, and Ingestion authentication credentials and sources.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.0
- [Go](https://go.dev/doc/install) >= 1.25 (for building from source)

## Getting Started

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
| **Search** — MCM clusters & user IDs | ✅ | ✅ | 🟡 Planned · P1 (read-only) |
| **Search** — records · search · browse | ✅ | ✅ | ⛔ Out of scope (data-plane) |
| **Query Suggestions** | ✅ | ✅ | ✅ `algolia_query_suggestions` |
| **Personalization** — strategy | ✅ | ✅ | ✅ `algolia_personalization_strategy` |
| **Advanced Personalization** | ✅ | ❌ not in v4 | 🟡 Planned · P3 (investigate) |
| **Agent Studio** — agents & providers | ✅ | ❌ custom client | ✅ `algolia_agent`, `algolia_agent_provider` |
| **Ingestion** — authentications | ✅ | ✅ | ✅ `algolia_ingestion_authentication` |
| **Ingestion** — sources | ✅ | ✅ | ✅ `algolia_ingestion_source` |
| **Ingestion** — destinations, tasks, transformations | ✅ | ✅ | 🟡 Planned · P2 |
| **A/B Testing** | ✅ | ✅ | 🟡 Planned · P2 |
| **Recommend** — rules | ✅ | ✅ | 🟡 Planned · P2 |
| **Composition** | ✅ | ✅ | 🟡 Planned · P2 |
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
