# Terraform Provider for Algolia — API Coverage Roadmap

This document tracks how the provider will grow to cover Algolia's full API surface.

## Source of truth

The provider does not vendor an OpenAPI spec. The authoritative definitions are:

- **Algolia OpenAPI specs:** [`algolia/api-clients-automation`](https://github.com/algolia/api-clients-automation/tree/main/specs) — one folder per API surface.
- **Algolia Go client v4** (`github.com/algolia/algoliasearch-client-go/v4`, pinned in `go.mod`) — code-generated from those specs. This is what the provider actually calls, so the client's packages are our practical, offline mirror of the spec surface.

The specs repo exposes 14 real API surfaces (plus `bundled`/`common`, which are meta): `search`, `personalization`, `advanced-personalization`, `query-suggestions`, `agent-studio`, `abtesting`, `abtesting-v3`, `composition`, `recommend`, `ingestion`, `crawler`, `analytics`, `monitoring`, `insights`.

## Scope principle

A Terraform provider manages **declarative, persistent configuration** with a create/read/update/delete lifecycle. It is **not** a runtime/data-plane client. That principle decides what belongs here:

- **Manage (resource):** persistent config an operator wants versioned and reconciled — index settings, API keys, rules, A/B tests, ingestion connectors, crawler configs.
- **Read (data source):** read-only lookups and reporting.
- **Out of scope (runtime/data-plane):** search/browse queries, event ingestion, recommendation retrieval, record (object) writes, secured-key generation. These are per-request operations with no stored, reconcilable state and do not belong in Terraform.

## Coverage matrix

| API surface | Terraform verdict | Status |
|---|---|---|
| **search** — index settings | resource + data source | ✅ `algolia_index`, `algolia_virtual_index` |
| **search** — index listing | data source | ✅ `algolia_indices` |
| **search** — API keys | resource + data source | ✅ `algolia_api_key`, `algolia_api_keys` |
| **search** — rules | resource + data source | ✅ `algolia_rule` |
| **search** — synonyms | resource + data source | ✅ `algolia_synonym` |
| **search** — dictionaries — custom entries | resource + data source | ✅ `algolia_dictionary_entry` |
| **search** — dictionaries — settings (`disableStandardEntries`) | resource + data source | ✅ `algolia_dictionary_settings` |
| **search** — allowed sources (IP allowlist) | resource | ✅ `algolia_allowed_sources` |
| **search** — MCM clusters / user IDs | data source (clusters); user-id assignment is operational | ❌ gap, low priority |
| **search** — records/objects, search, browse, secured keys | out of scope (data-plane) | — |
| **query-suggestions** | resource + data source | ✅ `algolia_query_suggestions` |
| **personalization** — strategy | resource + data source | ✅ `algolia_personalization_strategy` |
| **advanced-personalization** | evaluate vs. existing strategy | ❌ investigate |
| **agent-studio** | resource + data source | ✅ `algolia_agent`, `algolia_agent_provider` |
| **ingestion** — sources, destinations, authentications, tasks, transformations | resource + data source | ❌ gap (highest IaC value) |
| **ingestion** — runs, events, push | out of scope (runtime) | — |
| **abtesting** / **abtesting-v3** | resource | ❌ gap |
| **recommend** — recommend rules | resource + data source | ❌ gap |
| **recommend** — get recommendations | out of scope (data-plane) | — |
| **composition** — compositions + composition rules | resource + data source | ❌ gap |
| **crawler** | resource + data source | ❌ gap (needs custom client — see below) |
| **analytics** | data source only | ❌ optional, low priority |
| **monitoring** | data source only | ❌ optional, low priority |
| **insights** — push events / delete user token | out of scope (runtime) | — |

## Current state (baseline)

12 resources and 15 data sources across `search`, `query-suggestions`, `personalization`, and `agent-studio`. Registered in `internal/provider/provider.go`; each lives under `internal/services/<name>/` following the model / schema / expand / flatten / resource / data_source layout described in `AGENTS.md`.

## Phased roadmap

Phases are ordered by **value ÷ effort**, front-loading work where the Go client is already wired in.

### Phase 1 — Complete the Search API

The `search` client is already a provider dependency, so these are low-risk additions that close obvious gaps.

- ✅ `algolia_dictionary_entry` — custom stopwords / plurals / compounds (`BatchDictionaryEntries`, `SearchDictionaryEntries`). Shipped (resource + data source).
- ✅ `algolia_dictionary_settings` — `disableStandardEntries` (`SetDictionarySettings` / `GetDictionarySettings`). Shipped (resource + data source).
- ✅ `algolia_allowed_sources` — API access IP allowlist (`GetSources` / `ReplaceSources` / `DeleteSource`). Shipped (resource + data source).
- ✅ Search lookup data sources — `algolia_api_key` (`GetApiKey`), `algolia_api_keys` (`ListApiKeys`), `algolia_indices` (`ListIndices`, paginated). Shipped (data-source-only, no CRUD/import).

**Effort:** S–M each. **Risk:** low.

### Phase 2 — Net-new configuration APIs (client already in v4)

`ingestion`, `abtesting`, `abtesting-v3`, `recommend`, and `composition` all ship in the Go client — no new transport work, just new service packages.

- **Ingestion** (highest priority — the canonical infrastructure-as-code use case):
  `algolia_ingestion_source`, `algolia_ingestion_destination`, `algolia_ingestion_authentication`, `algolia_ingestion_task`, `algolia_ingestion_transformation`, plus read-only data sources. Region-routed — reuse `internal/analyticsregion` patterns.
  Scaffolding shipped (`internal/analyticsregion.NewIngestionClient` + `internal/services/ingestion/client.go`); the five resources/data sources themselves are still to come.
- **A/B Testing:** `algolia_ab_test` (build on `abtesting-v3`; region-routed).
- **Recommend:** `algolia_recommend_rule` (`GetRecommendRule` / `BatchRecommendRules` / `DeleteRecommendRule`) + data source.
- **Composition:** `algolia_composition`, `algolia_composition_rule` + data sources.

**Effort:** Ingestion L (5 resources); others M each. **Risk:** low–medium (region routing, task/run lifecycle semantics).

### Phase 3 — APIs needing extra client work

- **Crawler** — `algolia_crawler` config (+ data source). The Crawler has its own API host and is **not part of the Go v4 client**, so this needs a small dedicated HTTP client under `internal/services/crawler/` (mirror the pattern used for the Agent Studio client). High customer value; the added client is the main cost.
- **Advanced Personalization** — investigate how `advanced-personalization` relates to the existing `personalization_strategy` resource; decide whether it's a new resource, an evolution of the current one, or client-gated.

**Effort:** Crawler M–L (custom client). **Risk:** medium.

### Phase 4 — Read-only observability (data sources only)

Optional, demand-driven. Low Terraform value because the data is runtime reporting, not config.

- **Analytics** data sources (top searches, no-results, click/conversion metrics).
- **Monitoring** data sources (status, incidents, latency, reachability).

**Effort:** S each, repetitive. **Risk:** low.

## Explicitly out of scope

Runtime / data-plane operations with no reconcilable state, and why:

- **Search / browse / facet queries** (`search` API) — per-request reads.
- **Records / objects** (`SaveObject`, `Batch`, `PartialUpdate`) — indexed *data*, not infrastructure; churns constantly and doesn't belong in Terraform state.
- **Insights** (`PushEvents`, `DeleteUserToken`) — event stream.
- **Recommend** `GetRecommendations` — per-request reads.
- **Secured API keys** (`GenerateSecuredApiKey`) — a client-side signing helper, not a stored resource.
- **MCM user-ID assignment** — operational cluster balancing, not declarative config (cluster *listing* may still ship as a data source).

## Cross-cutting workstreams

These run alongside every phase:

1. **Keep the Go client current.** Pinned at `v4.37.1`; newer releases add spec coverage and fixes. Bump deliberately per phase.
2. **Consider spec-driven scaffolding.** With 5+ new services incoming (especially Ingestion), evaluate generating the model / expand / flatten boilerplate from the OpenAPI schema instead of hand-writing each — even a one-off generator pays off across Ingestion + Composition + Recommend.
3. **Acceptance-test gating.** New region-routed services need `ALGOLIA_ANALYTICS_REGION`; quota-limited or destructive APIs need an opt-in env flag, mirroring `ALGOLIA_RUN_PERSONALIZATION_ACC`. Document each in `AGENTS.md`.
4. **Docs & examples.** Every resource ships with a `tfplugindocs`-generated page (`make generate`) and an `examples/` entry, matching the current layout.

## Suggested sequencing

```
Phase 1  (search gaps)      ──▶  Phase 2  (ingestion, ab-test, recommend, composition)
                                     │
                                     ▼
                                 Phase 3  (crawler, advanced-personalization)
                                     │
                                     ▼
                                 Phase 4  (analytics / monitoring data sources — as demanded)
```

Within Phase 2, **Ingestion first** — it delivers the most infrastructure-as-code value and its multi-resource shape sets the template for the rest.
