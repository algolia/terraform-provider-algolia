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
  app_id  = "YOUR_APP_ID"   # or ALGOLIA_APP_ID env var
  api_key = "YOUR_API_KEY"  # or ALGOLIA_API_KEY env var
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

| Name                          | Description                      |
|-------------------------------|----------------------------------|
| `algolia_index` (resource)    | Create and manage index settings |
| `algolia_index` (data source) | Read existing index settings     |

## ⚙️ Configuration

| Attribute | Env Variable      | Description            |
|-----------|-------------------|------------------------|
| `app_id`  | `ALGOLIA_APP_ID`  | Algolia Application ID |
| `api_key` | `ALGOLIA_API_KEY` | Algolia Admin API Key  |

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
make testacc    # acceptance tests (requires ALGOLIA_APP_ID & ALGOLIA_API_KEY)
make lint       # golangci-lint
make generate   # regenerate docs
```
