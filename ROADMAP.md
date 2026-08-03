# API coverage

What this provider covers of Algolia's API surface, what it deliberately does not, and why.

## Source of truth

The provider does not vendor an OpenAPI spec. The authoritative definitions are:

- **Algolia OpenAPI specs:** [`algolia/api-clients-automation`](https://github.com/algolia/api-clients-automation/tree/main/specs), one folder per API surface.
- **Algolia Go client v4** (`github.com/algolia/algoliasearch-client-go/v4`, pinned in `go.mod`), code-generated from those specs. This is what the provider actually calls, so the client's packages are the practical, offline mirror of the spec surface.

The specs repo exposes 14 real API surfaces (plus `bundled`/`common`, which are meta): `search`, `personalization`, `advanced-personalization`, `query-suggestions`, `agent-studio`, `abtesting`, `abtesting-v3`, `composition`, `recommend`, `ingestion`, `crawler`, `analytics`, `monitoring`, `insights`.

## Scope principle

A Terraform provider manages **declarative, persistent configuration** with a create/read/update/delete lifecycle. It is **not** a runtime or data-plane client. That principle decides what belongs here:

- **Manage (resource):** persistent config an operator wants versioned and reconciled, such as index settings, API keys, rules, A/B tests and ingestion connectors.
- **Read (data source):** read-only lookups and reporting.
- **Out of scope (runtime, data-plane):** search and browse queries, event ingestion, recommendation retrieval, record writes, secured-key generation. These are per-request operations with no stored, reconcilable state.

## Coverage matrix

| API surface | Terraform verdict | Status |
|---|---|---|
| **search**: index settings | resource + data source | ✅ `algolia_index`, `algolia_virtual_index` |
| **search**: index listing | data source | ✅ `algolia_indices` |
| **search**: API keys | resource + data source | ✅ `algolia_api_key`, `algolia_api_keys` |
| **search**: rules | resource + data source | ✅ `algolia_rule` |
| **search**: synonyms | resource + data source | ✅ `algolia_synonym` |
| **search**: dictionaries, custom entries | resource + data source | ✅ `algolia_dictionary_entry` |
| **search**: dictionaries, settings (`disableStandardEntries`) | resource + data source | ✅ `algolia_dictionary_settings` |
| **search**: allowed sources (IP allowlist) | resource | ✅ `algolia_allowed_sources` |
| **search**: MCM clusters / user IDs | data source (clusters, user IDs); user-id assignment is operational | ✅ `algolia_clusters`, `algolia_user_ids` |
| **search**: records/objects, search, browse, secured keys | out of scope (data-plane) | not planned |
| **query-suggestions** | resource + data source | ✅ `algolia_query_suggestions` |
| **personalization**: strategy | resource + data source | ✅ `algolia_personalization_strategy` |
| **advanced-personalization** | new resource (singleton `/2/config`); needs a custom region-routed client | ⏸️ Deferred (decided 2026-07-18): a separate product from classic Personalization, and not in the v4 client, so it needs a hand-rolled client. Deferred until there is demand. |
| **agent-studio**: agents, providers | resource + data source | ✅ `algolia_agent`, `algolia_agent_provider` |
| **agent-studio**: allowed domains, secret keys | resource + data source | ❌ Not started. The v4 client has full CRUD for both (`CreateAgentAllowedDomain`, `CreateSecretKey`), but neither has a provider surface. A published agent is therefore not reachable from any origin without an out-of-band step, and the keys behind secured user tokens cannot be provisioned or rotated as code. |
| **ingestion**: authentications | resource + data source | ✅ `algolia_ingestion_authentication` |
| **ingestion**: sources | resource + data source | ✅ `algolia_ingestion_source` |
| **ingestion**: destinations | resource + data source | ✅ `algolia_ingestion_destination` |
| **ingestion**: transformations | resource + data source | ✅ `algolia_ingestion_transformation` |
| **ingestion**: tasks | resource + data source | ✅ `algolia_ingestion_task` |
| **ingestion**: runs, events, push | out of scope (runtime) | not planned |
| **abtesting** / **abtesting-v3** | resource | ✅ `algolia_ab_test` |
| **recommend**: recommend rules | resource + data source | ✅ `algolia_recommend_rule` |
| **recommend**: get recommendations | out of scope (data-plane) | not planned |
| **composition**: compositions + composition rules | resource + data source | ✅ `algolia_composition`, `algolia_composition_rule` |
| **crawler** | ~~resource + data source~~ | 🚫 Won't do (descoped 2026-07-18). An HTTP client for it exists under `internal/services/crawler/`, but no resource is built on it and none is planned. |
| **analytics** | ~~data source only~~ | 🚫 Won't do (descoped 2026-07-18): runtime reporting, a poor fit for Terraform state. |
| **monitoring** | ~~data source only~~ | 🚫 Won't do (descoped 2026-07-18): runtime reporting, a poor fit for Terraform state. |
| **insights**: push events, delete user token | out of scope (runtime) | not planned |

## Explicitly out of scope

Runtime and data-plane operations with no reconcilable state, and why:

- **Search, browse and facet queries** (`search` API): per-request reads.
- **Records and objects** (`SaveObject`, `Batch`, `PartialUpdate`): indexed *data* rather than infrastructure. It churns constantly and does not belong in Terraform state.
- **Insights** (`PushEvents`, `DeleteUserToken`): an event stream.
- **Recommend** `GetRecommendations`: per-request reads.
- **Secured API keys** (`GenerateSecuredApiKey`): a client-side signing helper, not a stored resource.
- **MCM user-ID assignment**: operational cluster balancing rather than declarative config. Cluster and user-ID *listing* does ship, as the `algolia_clusters` and `algolia_user_ids` data sources.
