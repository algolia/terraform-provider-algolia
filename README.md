<div align="center">

# Terraform Provider for Algolia

Manage your [Algolia](https://www.algolia.com/) search infrastructure as code.

[![Status](https://img.shields.io/badge/status-beta-orange.svg)](https://github.com/algolia/terraform-provider-algolia/releases)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL_2.0-blue.svg)](LICENSE)

</div>

## What it does

Configure Algolia declaratively instead of through the dashboard: index settings and
replicas, API keys, rules, synonyms, dictionaries, Query Suggestions, Personalization,
A/B tests, Recommend rules, Compositions, Agent Studio agents, and the whole Ingestion
pipeline.

**21 resources and 26 data sources.** Every resource supports `terraform import`. Reference
documentation for each is in [`docs/`](docs/): [resources](docs/resources/),
[data sources](docs/data-sources/).

| Area | Resources |
| --- | --- |
| Search | `algolia_index`, `algolia_virtual_index`, `algolia_rule`, `algolia_synonym`, `algolia_api_key`, `algolia_dictionary_entry`, `algolia_dictionary_settings` |
| Ingestion | `algolia_ingestion_source`, `algolia_ingestion_destination`, `algolia_ingestion_task`, `algolia_ingestion_transformation`, `algolia_ingestion_authentication` |
| Agent Studio | `algolia_agent`, `algolia_agent_provider` |
| Relevance | `algolia_query_suggestions`, `algolia_recommend_rule`, `algolia_personalization_strategy`, `algolia_ab_test` |
| Compositions | `algolia_composition`, `algolia_composition_rule` |
| Security | `algolia_allowed_sources` |

Which Algolia APIs are covered, and which are deliberately out of scope, is tracked in
[ROADMAP.md](ROADMAP.md#coverage-matrix).

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.0
- [Go](https://go.dev/doc/install) >= 1.25, only to build from source

## Install

The provider is not on the Terraform Registry, so it installs from a release archive. With
[`gh`](https://cli.github.com/) authenticated:

```bash
gh api -H "Accept: application/vnd.github.raw" \
  repos/algolia/terraform-provider-algolia/contents/scripts/install.sh | bash
```

Then pin the version it prints. See [INSTALL.md](INSTALL.md) for the other routes, doing it
by hand, and what to check when `terraform init` cannot find the provider.

## Use it

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

resource "algolia_index" "products" {
  name = "products"

  attributes {
    searchable_attributes = ["name", "description"]
  }

  ranking {
    custom_ranking = ["desc(popularity)"]
  }
}
```

`terraform import algolia_index.products products` brings an existing index under
management. The import ID differs per resource: rules and synonyms take
`<index>/<object_id>`, Recommend rules add the model (`<index>/<model>/<object_id>`), and
agents and Ingestion resources take a UUID. Each resource's docs page has a worked example.

| Provider argument | Environment variable | Notes |
| --- | --- | --- |
| `app_id` | `ALGOLIA_APP_ID` | |
| `api_key` | `ALGOLIA_API_KEY` | Admin API key |
| `analytics_region` | `ALGOLIA_ANALYTICS_REGION` | `us` or `eu`. Required for Query Suggestions, Personalization, A/B testing and Ingestion |

Two behaviours worth knowing before your first apply: `deletion_protection` defaults to
`true` on the resources whose deletion cannot be undone, so `terraform destroy` refuses
until you set it to `false` and apply; and a primary index's replicas are split across two
resources, standard ones in `algolia_index`'s `advanced.replicas` and each virtual one in
its own `algolia_virtual_index`. The [CHANGELOG](CHANGELOG.md) notes cover both.

## Using it with a coding agent

[`skills/algolia-terraform-provider`](skills/algolia-terraform-provider/) is an agent skill
covering installation and the attribute shapes that make a plausible-looking configuration
fail. Install it into whichever agent you use:

```bash
npx skills add algolia/terraform-provider-algolia
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and testing, and
[AGENTS.md](AGENTS.md) if you are working here with a coding agent.

## License

[Mozilla Public License 2.0](LICENSE)
