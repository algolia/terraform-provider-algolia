<div align="center">

# Terraform Provider for Algolia

Manage your [Algolia](https://www.algolia.com/) search infrastructure as code.

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

Terraform installs the provider from the public
[Terraform Registry](https://registry.terraform.io/providers/algolia/algolia/latest) when you
declare it in `required_providers` as shown below and initialize the configuration:

```bash
terraform init
```

See [INSTALL.md](INSTALL.md) for version upgrades, signed release installation as a fallback,
and what to check when `terraform init` cannot find the provider.

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

## Sensitive data

In addition to the Algolia Admin API key, resources and data sources can handle:

- Algolia API-key values managed by `algolia_api_key` or read by the
  `algolia_api_key` and `algolia_api_keys` data sources
- Agent Provider API keys managed by `algolia_agent_provider`
- MCP authorization headers managed or read by `algolia_agent`
- credentials in `algolia_ingestion_authentication.input`
- connector secrets and presigned URLs in `algolia_ingestion_source.input`

These attributes are marked sensitive, which suppresses their values in normal Terraform
output but does not omit or encrypt them in state. Sensitive values may also be present in
configuration, saved plans, outputs, logs, and CI artifacts. Treat state and plan files as
sensitive, restrict access to them, and use the encryption and access controls offered by
the selected backend and CI platform. Environment variables and sensitive Terraform
variables can keep credentials out of checked-in configuration and routine CLI output,
but values assigned to resource attributes can still be retained in state.

Configuration repositories, state backends, saved plans, outputs, and CI environments
are controlled by the Terraform operator and the systems they select.

## Downstream external integrations

The provider runs in the operator's environment and sends configuration to Algolia;
Algolia services may make downstream calls, like so:

```text
Operator -> Terraform Provider -> Algolia API -> Algolia-managed service -> External endpoint
```

Configuration that may cause downstream calls includes:

- Agent Studio: Agent Provider `base_url` fields and Agent MCP `url`, optionally with `headers`
- Ingestion: source and OAuth URLs; connector configuration and transformation code may
  specify additional destinations


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
