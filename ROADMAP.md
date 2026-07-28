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
| **search** — MCM clusters / user IDs | data source (clusters, user IDs); user-id assignment is operational | ✅ `algolia_clusters`, `algolia_user_ids` |
| **search** — records/objects, search, browse, secured keys | out of scope (data-plane) | — |
| **query-suggestions** | resource + data source | ✅ `algolia_query_suggestions` |
| **personalization** — strategy | resource + data source | ✅ `algolia_personalization_strategy` |
| **advanced-personalization** | new resource (singleton `/2/config`); needs a custom region-routed client | ⏸️ Deferred (spike done 2026-07-18): separate product from classic Personalization, not in the v4 client |
| **agent-studio** — agents, providers | resource + data source | ✅ `algolia_agent`, `algolia_agent_provider` |
| **agent-studio** — allowed domains, secret keys | resource + data source | ❌ Not started. The v4 client has full CRUD for both (`CreateAgentAllowedDomain`, `CreateSecretKey`, ...), but neither has a provider surface. A published agent is therefore not reachable from any origin without an out-of-band step, and the keys behind secured user tokens cannot be provisioned or rotated as code. |
| **ingestion** — authentications | resource + data source | ✅ `algolia_ingestion_authentication` |
| **ingestion** — sources | resource + data source | ✅ `algolia_ingestion_source` |
| **ingestion** — destinations | resource + data source | ✅ `algolia_ingestion_destination` |
| **ingestion** — transformations | resource + data source | ✅ `algolia_ingestion_transformation` |
| **ingestion** — tasks | resource + data source | ✅ `algolia_ingestion_task` |
| **ingestion** — runs, events, push | out of scope (runtime) | — |
| **abtesting** / **abtesting-v3** | resource | ✅ `algolia_ab_test` |
| **recommend** — recommend rules | resource + data source | ✅ `algolia_recommend_rule` |
| **recommend** — get recommendations | out of scope (data-plane) | — |
| **composition** — compositions + composition rules | resource + data source | ✅ `algolia_composition`, `algolia_composition_rule` |
| **crawler** | ~~resource + data source~~ | 🚫 Won't do (descoped 2026-07-18); P3.1a client shipped, resource dropped |
| **analytics** | ~~data source only~~ | 🚫 Won't do (descoped 2026-07-18); runtime reporting, poor IaC fit |
| **monitoring** | ~~data source only~~ | 🚫 Won't do (descoped 2026-07-18); runtime reporting, poor IaC fit |
| **insights** — push events / delete user token | out of scope (runtime) | — |

## Current state (baseline)

21 resources and 26 data sources across `search`, `query-suggestions`, `personalization`, `agent-studio`, `abtesting-v3`, `composition`, `recommend`, `ingestion`, and `monitoring` (MCM, data sources only). Registered in `internal/provider/provider.go`; each lives under `internal/services/<name>/` following the model / schema / expand / flatten / resource / data_source layout described in `AGENTS.md`.

## Phased roadmap

Phases are ordered by **value ÷ effort**, front-loading work where the Go client is already wired in.

### Phase 1 — Complete the Search API

The `search` client is already a provider dependency, so these are low-risk additions that close obvious gaps.

- ✅ `algolia_dictionary_entry` — custom stopwords / plurals / compounds (`BatchDictionaryEntries`, `SearchDictionaryEntries`). Shipped (resource + data source).
- ✅ `algolia_dictionary_settings` — `disableStandardEntries` (`SetDictionarySettings` / `GetDictionarySettings`). Shipped (resource + data source).
- ✅ `algolia_allowed_sources` — API access IP allowlist (`GetSources` / `ReplaceSources` / `DeleteSource`). Shipped (resource + data source).
- ✅ Search lookup data sources — `algolia_api_key` (`GetApiKey`), `algolia_api_keys` (`ListApiKeys`), `algolia_indices` (`ListIndices`, paginated). Shipped (data-source-only, no CRUD/import).
- ✅ **MCM read-only data sources** — `algolia_clusters` (`ListClusters`) + `algolia_user_ids` (`ListUserIds`, paged). Shipped in `internal/services/mcm/` (data-source-only; both endpoints are deprecated in Algolia's own API but still functional). `ListClustersResponse` only carries cluster *names* (its `topUsers` field is a `[]string`, despite the name) — no per-cluster record/user counts or data size, unlike the richer `Cluster` schema documented elsewhere in Algolia's spec, so `algolia_clusters` only exposes `cluster_name`. `ListUserIdsResponse` carries no paging metadata (no `nbPages`/`nbUsers`, unlike `ListIndicesResponse`), so `algolia_user_ids` pages by requesting a fixed `hitsPerPage` and stopping once a page returns fewer items than that. Acceptance tests are additionally gated behind `ALGOLIA_RUN_MCM_ACC=1`, since MCM endpoints error out on applications that aren't on a multi-cluster plan.

**Effort:** S–M each. **Risk:** low.

### Phase 2 — Net-new configuration APIs (client already in v4)

`ingestion`, `abtesting`, `abtesting-v3`, `recommend`, and `composition` all ship in the Go client — no new transport work, just new service packages.

- **Ingestion** (highest priority — the canonical infrastructure-as-code use case):
  `algolia_ingestion_source`, `algolia_ingestion_destination`, `algolia_ingestion_authentication`, `algolia_ingestion_task`, `algolia_ingestion_transformation`, plus read-only data sources. Region-routed — reuse `internal/analyticsregion` patterns.
  **Shipped in full** — all five resources (+ matching data sources). Scaffolding: `internal/analyticsregion.NewIngestionClient` + `internal/services/ingestion/client.go`. Authentication's `input` credentials are modeled as a JSON-encoded, write-only `Sensitive` string (the API redacts secrets on read, so it's never refreshed) — see `internal/services/ingestion/authentication_schema.go`. Source's `input` is JSON-encoded too, but *not* Sensitive and *is* refreshed on read (`GetSource` returns it in full, unredacted) — refreshing naively would cause a perpetual diff from harmless JSON reordering, so `flattenSource` only adopts the API's encoding when it isn't semantically equal (ignoring key/array order) to the configured value; see `internal/services/ingestion/source_flatten.go` and the shared `jsonSemanticallyEqual`/`normalizeJSON` helpers in `internal/services/ingestion/json.go`. Source's create (`SourceCreate.Input *SourceInput`) and update (`SourceUpdate.Input *SourceUpdateInput`) endpoints use two different union types for the same JSON, handled in `source_expand.go`. Destination follows the same `input` refresh pattern (`destination_flatten.go`), but unlike Source, `input` is *required* (every destination writes to a specific `indexName`) and `DestinationInput`/`DestinationUpdateInput` are plain structs rather than tagged unions — no need to decode into a variant type per destination `type`. Destination also adds `transformation_ids` (a `List` of transformation UUIDs, converted to/from `[]string` the way `internal/services/dictionary` handles `words`/`decomposition`). Transformation's legacy `code` field is a plain string (not JSON), refreshed directly on read with no semantic-equality dance since there's no key/array-order ambiguity in a raw string; its `input` (JSON-encoded `TransformationInput` union, for no-code transformations) follows the same optional, semantic-equality-preserving pattern as Source's. Unlike Source/Destination, `UpdateTransformation` accepts the very same `*TransformationCreate` body as `CreateTransformation` (including `type`), so `type` is *not* `RequiresReplace` there and `expandTransformationCreate` is reused for both Create and Update. Task — the last piece, tying a source and destination together — has three JSON-encoded fields (`input`, `notifications`, `policies`) all following the same semantic-equality-preserving pattern (`task_flatten.go`); `source_id` and `action` are `RequiresReplace` since `TaskUpdate` has neither field; `enabled` is `Optional`+`Computed` with a `booldefault.StaticBool(true)` default, mirroring `deletion_protection`/`agent`'s `publish` field. `cursor` is the one attribute deliberately *never* refreshed from the API (unlike every other field here): `TaskUpdate` has no `cursor` field at all, and its true value advances automatically as the task runs in the background (out of scope — runs/events are runtime), so refreshing it naively risks Terraform's "provider produced inconsistent result" error once a long-lived cron task's cursor has moved between applies; `flattenTask` simply leaves it untouched, mirroring `algolia_ingestion_authentication`'s write-only `input` handling. `runs`/`events` remain explicitly out of scope (runtime, no reconcilable state).
- ✅ **A/B Testing:** `algolia_ab_test` (build on `abtesting-v3`; region-routed). Shipped (resource + data source) in `internal/services/abtest/`. Scaffolding: `internal/analyticsregion.NewABTestingClient` - the one region-routed client in this provider whose Region enum is *not* "eu"/"us": abtesting-v3 uses "de"/"us" (its EU cluster is hosted in Germany), so `NewABTestingClient` maps the provider's normalized "eu" region onto `abtesting.DE` to keep `analytics_region` consistent across every region-routed resource. `AddABTests` has no update endpoint, so `name`/`end_at`/`variants`/`metrics`/`configuration` are all `RequiresReplace`, and - unlike every other JSON-encoded field in this provider - Read never refreshes them from `GetABTest`: that response is enriched with runtime results (per-variant metrics, significance, status) whose shape diverges from what was submitted at creation, so refreshing naively would corrupt state and cause a perpetual diff (the same write-once rationale as `algolia_ingestion_authentication`'s `input`, applied to five fields instead of one). Only `status` and the identifiers are refreshed on read. The data source has no prior configuration to protect, so it surfaces `GetABTest`'s enriched response directly, including per-variant results. Import recovers `name`/`end_at`/`configuration` from the enriched response but not `metrics` (not returned in its original shape by `GetABTest`) and only approximately for `variants`; acceptance tests are additionally gated behind `ALGOLIA_RUN_ABTESTING_ACC=1` (creating a test has cost/quota implications), mirroring Personalization's `ALGOLIA_RUN_PERSONALIZATION_ACC`.
- ✅ **Recommend:** `algolia_recommend_rule` (`GetRecommendRule` / `BatchRecommendRules` / `DeleteRecommendRule`) + data source. Shipped in `internal/services/recommend/`. Not region-routed, unlike every other Phase 2 service - `recommend.NewClient(appID, apiKey)` takes no region, so `internal/services/recommend/client.go`'s `base` only carries `appID`/`apiKey` (built on demand, same on-demand-client convention as `abtest.base`, just without the `analyticsRegion` field). Composite ID `<index_name>/<model>/<object_id>` (`model` = `related-products`|`bought-together`|`trending-items`|`trending-facets`, `stringvalidator.OneOf` derived from `recommend.AllowedRecommendModelsEnumValues`); `object_id` is generated (UUIDv4) when omitted, mirroring `algolia_dictionary_entry`. `condition`/`consequence`/`validity` are JSON-encoded strings (unlike `algolia_rule`, which models the equivalent Search Rule concepts as nested blocks) refreshed on read using the same semantic-equality preserve-prior pattern as Ingestion's JSON attributes - the helper is replicated (not imported) into `internal/services/recommend/json.go`, per the existing precedent in `internal/services/index`/`internal/services/ingestion`. Deletes are asynchronous like every other Recommend write, waited out via `GetRecommendStatus` (a per-model/index task-status endpoint distinct from `search`'s `GetTask`). `GetRecommendations` remains out of scope (data-plane).
- **Composition:** ✅ shipped — `algolia_composition`, `algolia_composition_rule` + data sources. **Phase 2 complete.**

**Effort:** Ingestion L (5 resources); others M each. **Risk:** low–medium (region routing, task/run lifecycle semantics).

### Phase 3 — APIs needing extra client work

- **Crawler** (won't do, descoped 2026-07-18): `algolia_crawler` config (+ data source) is not planned. The dedicated HTTP client under `internal/services/crawler/` (P3.1a) shipped and stays in the codebase, but no resource/data source will be built on it unless this is revisited.
- **Advanced Personalization** (spike done 2026-07-18): decision is a NEW resource, not an evolution of `algolia_personalization_strategy`. "AI Personalization" is a separate product (host `ai-personalization.{region}.algolia.com`, region-routed, API `/2`) and is not in the Go v4 client, so it needs a hand-rolled client. Only the singleton `/2/config` (indices, facet attributes, events) is Terraform-suitable. Deferred until there is demand (custom-client work, like the descoped crawler).

**Effort:** Advanced Personalization deferred pending demand (custom client needed); crawler descoped. **Risk:** low.

### Phase 4 — Read-only observability (data sources only)

**Won't do** (descoped 2026-07-18). Runtime reporting, not declarative config: the data changes constantly and nothing downstream consumes it, so it is a poor fit for Terraform state. Read it in the dashboard or via a script instead.

- ~~**Analytics** data sources (top searches, no-results, click/conversion metrics).~~
- ~~**Monitoring** data sources (status, incidents, latency, reachability).~~

## Explicitly out of scope

Runtime / data-plane operations with no reconcilable state, and why:

- **Search / browse / facet queries** (`search` API) — per-request reads.
- **Records / objects** (`SaveObject`, `Batch`, `PartialUpdate`) — indexed *data*, not infrastructure; churns constantly and doesn't belong in Terraform state.
- **Insights** (`PushEvents`, `DeleteUserToken`) — event stream.
- **Recommend** `GetRecommendations` — per-request reads.
- **Secured API keys** (`GenerateSecuredApiKey`) — a client-side signing helper, not a stored resource.
- **MCM user-ID assignment** — operational cluster balancing, not declarative config (cluster/user-ID *listing* ships as `algolia_clusters`/`algolia_user_ids` data sources — see Phase 1).

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
                                 Phase 3  (advanced-personalization)
                                     │
                                     ▼
                                 Phase 4  (analytics / monitoring: won't do)
```

Within Phase 2, **Ingestion first** — it delivers the most infrastructure-as-code value and its multi-resource shape sets the template for the rest.
