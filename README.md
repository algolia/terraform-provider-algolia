<div align="center">

# Terraform Provider for Algolia

**Terraform provider for managing [Algolia](https://www.algolia.com/) search infrastructure as code.**

[![License: MPL-2.0](https://img.shields.io/badge/License-MPL_2.0-blue.svg)](LICENSE)

</div>

## 🚀 Quick Start

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
  app_id           = "YOUR_APP_ID"     # or ALGOLIA_APP_ID env var
  api_key          = "YOUR_API_KEY"    # or ALGOLIA_API_KEY env var
  analytics_region = "us"              # or ALGOLIA_ANALYTICS_REGION env var
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

## 📚 Resources & Data Sources

| Name                                   | Description                                         |
|----------------------------------------|-----------------------------------------------------|
| `algolia_index` (resource)             | Create and manage index settings                    |
| `algolia_virtual_index` (resource)     | Manage virtual replica settings                     |
| `algolia_api_key` (resource)           | Manage scoped Algolia API keys                      |
| `algolia_rule` (resource)              | Manage index rules                                  |
| `algolia_synonym` (resource)           | Manage individual synonym objects                   |
| `algolia_query_suggestions` (resource) | Manage Query Suggestions configurations             |
| `algolia_personalization_strategy` (resource) | Manage the app-level Personalization strategy |
| `algolia_agent_provider` (resource)    | Manage Agent Studio LLM providers                   |
| `algolia_agent` (resource)             | Manage Agent Studio agents                          |
| `algolia_index` (data source)          | Read existing index settings                        |
| `algolia_virtual_index` (data source)  | Read virtual replica settings                       |
| `algolia_rule` (data source)           | Read a rule by index and object ID                  |
| `algolia_synonym` (data source)        | Read a synonym object by index and object ID        |
| `algolia_query_suggestions` (data source) | Read a Query Suggestions configuration           |
| `algolia_personalization_strategy` (data source) | Read the app-level Personalization strategy |
| `algolia_agent_provider` (data source) | Read Agent Studio provider metadata and endpoints   |
| `algolia_agent_provider_models` (data source) | List available models for an Agent Studio provider |
| `algolia_agent` (data source)          | Read Agent Studio agents                            |

## ⚙️ Configuration

| Attribute | Env Variable               | Description |
|-----------|----------------------------|-------------|
| `app_id`  | `ALGOLIA_APP_ID`           | Algolia Application ID |
| `api_key` | `ALGOLIA_API_KEY`          | Algolia Admin API Key |
| `analytics_region` | `ALGOLIA_ANALYTICS_REGION` | Analytics region for Query Suggestions and Personalization APIs (`us` or `eu`) |

## 🧩 Index Settings Blocks

Settings are organized into logical blocks:

| Block            | Key Settings                                                                |
|------------------|-----------------------------------------------------------------------------|
| `attributes`     | `searchable_attributes`, `attributes_to_retrieve`, `attribute_for_distinct` |
| `ranking`        | `ranking`, `custom_ranking`, `relevancy_strictness`                         |
| `faceting`       | `attributes_for_faceting`, `max_values_per_facet`, `sort_facet_values_by`   |
| `highlighting`   | `highlight_pre_tag`, `highlight_post_tag`, `attributes_to_snippet`          |
| `pagination`     | `hits_per_page`, `pagination_limited_to`                                    |
| `typos`          | `typo_tolerance`, `min_word_size_for_1_typo`, `min_word_size_for_2_typos`   |
| `languages`      | `query_languages`, `ignore_plurals`, `remove_stop_words`                    |
| `query_strategy` | `query_type`, `advanced_syntax`, `optional_words`                           |
| `performance`    | `numeric_attributes_for_filtering`, `allow_compression_of_integer_array`    |
| `advanced`       | `distinct`, `replicas`, `mode`, `enable_re_ranking`                         |

## 📥 Import

```bash
terraform import algolia_index.products existing_index_name
```

## 🛠️ Development

```bash
make build      # compile
make test       # unit tests
make testacc    # acceptance tests (requires ALGOLIA_APP_ID & ALGOLIA_API_KEY; some resources also require ALGOLIA_ANALYTICS_REGION)
make lint       # golangci-lint
make generate   # regenerate docs
```

Query Suggestions and Personalization acceptance tests also require `ALGOLIA_ANALYTICS_REGION`. Personalization acceptance tests are opt-in via `ALGOLIA_RUN_PERSONALIZATION_ACC=1` because the API enforces a daily strategy-save quota.
