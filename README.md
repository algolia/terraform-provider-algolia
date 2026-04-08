<div align="center">

# Terraform Provider for Algolia

Manage your [Algolia](https://www.algolia.com/) search infrastructure as code.

[![Terraform Registry](https://img.shields.io/badge/Terraform-Registry-purple.svg)](https://registry.terraform.io/providers/algolia/algolia/latest)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL_2.0-blue.svg)](LICENSE)

</div>

## Overview

The Algolia Terraform provider lets you configure and manage Algolia resources declaratively: index settings, API keys, query suggestions, personalization strategies, rules, synonyms, and Agent Studio agents.

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
| [`algolia_query_suggestions`](docs/resources/query_suggestions.md)               | Configure Query Suggestions                                                   |
| [`algolia_personalization_strategy`](docs/resources/personalization_strategy.md) | Set the app-level Personalization strategy                                    |
| [`algolia_agent_provider`](docs/resources/agent_provider.md)                     | Register LLM providers for Agent Studio                                       |
| [`algolia_agent`](docs/resources/agent.md)                                       | Create and configure Agent Studio agents                                      |

## Data Sources

| Data Source                                                                         | Description                                        |
| ----------------------------------------------------------------------------------- | -------------------------------------------------- |
| [`algolia_index`](docs/data-sources/index.md)                                       | Read existing index settings                       |
| [`algolia_virtual_index`](docs/data-sources/virtual_index.md)                       | Read virtual replica settings                      |
| [`algolia_rule`](docs/data-sources/rule.md)                                         | Read a rule by index and object ID                 |
| [`algolia_synonym`](docs/data-sources/synonym.md)                                   | Read a synonym by index and object ID              |
| [`algolia_query_suggestions`](docs/data-sources/query_suggestions.md)               | Read a Query Suggestions configuration             |
| [`algolia_personalization_strategy`](docs/data-sources/personalization_strategy.md) | Read the Personalization strategy                  |
| [`algolia_agent_provider`](docs/data-sources/agent_provider.md)                     | Read Agent Studio provider details                 |
| [`algolia_agent_provider_models`](docs/data-sources/agent_provider_models.md)       | List models available for an Agent Studio provider |
| [`algolia_agent`](docs/data-sources/agent.md)                                       | Read Agent Studio agent details                    |

## Importing Existing Resources

Bring existing Algolia resources under Terraform management:

```bash
terraform import algolia_index.products products
terraform import algolia_api_key.search AB1CD2EF3G
terraform import algolia_rule.promo products/my-rule-id
terraform import algolia_synonym.brand products/my-synonym-id
```

## Contributing

Interested in contributing? See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, build instructions, and testing guidelines.

## License

[Mozilla Public License 2.0](LICENSE)
