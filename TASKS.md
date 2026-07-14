# Implementation Tasks

Agent-ready breakdown of [ROADMAP.md](ROADMAP.md). Each task is scoped to be picked up independently by a coding agent. Work top-to-bottom within a phase; tasks marked **(depends: …)** must wait for their prerequisite.

## How to use this file

- Pick the first unchecked task whose dependencies are met.
- Follow the **Definition of Done** below — it applies to every resource/data-source task.
- Confirm exact client signatures against the version pinned in `go.mod` (currently `algoliasearch-client-go/v4 v4.44.0`); method-name hints here are from that version.
- Reference implementation to mirror: `internal/services/index/` (settings/keys/rules/synonyms) and `internal/services/agent/` (for APIs needing a custom HTTP client).

## Definition of Done (every resource/data-source task)

Follow the pattern in `AGENTS.md` → "Adding a New Resource":

- [ ] New package under `internal/services/<name>/` with `model.go`, `schema.go`, `model_expand.go`, `model_flatten.go`, `resource.go`, and `data_source.go` (+ `data_source_schema.go`).
- [ ] Access the Algolia client via `*providertypes.ProviderData` from `internal/types/`.
- [ ] If the API is region-routed, use `internal/analyticsregion` + `ProviderData.AnalyticsRegion` — do **not** embed per-resource region config.
- [ ] Full CRUD + `ImportState` implemented; `Read` handles remote-deleted resources (removes from state, no error).
- [ ] Register the resource/data source in `internal/provider/provider.go` (`Resources` / `DataSources`).
- [ ] Unit tests for expand/flatten (`*_test.go`, no credentials needed — must pass under `make test`).
- [ ] Acceptance tests (`TF_ACC=1`) gated so they skip without credentials; document any extra env gate in `AGENTS.md` (mirror `ALGOLIA_ANALYTICS_REGION` / `ALGOLIA_RUN_PERSONALIZATION_ACC`).
- [ ] Docs + example: add `examples/resources/<name>/resource.tf` (and `examples/data-sources/...`), run `make generate`, update the coverage tables in `README.md` (flip 🟡 → ✅) and `ROADMAP.md`.
- [ ] `make build`, `make test`, and `make lint` all pass.

---

## Phase 0 — Foundation

- [x] **P0.1 — Bump the Go client.** Update `algoliasearch-client-go/v4` to the latest `v4.x` in `go.mod`, run `go mod tidy`, and confirm `make build`/`make test` pass. Do this first so later phases build against current API models.
- [ ] **P0.2 — (Spike) Evaluate scaffolding generation.** Given 10+ new services incoming, assess whether the `model`/`expand`/`flatten` boilerplate can be generated from the OpenAPI schema. Output a short recommendation (generate vs. hand-write) before starting Phase 2. Non-blocking.

## Phase 1 — Complete the Search API

Uses the existing `search` client — no new client wiring.

- [x] **P1.1 — `algolia_dictionary_entry` (resource + data source).** Custom stopword/plural/compound entries.
  - Client: `SetDictionaryEntries`/`BatchDictionaryEntries` (upsert), `SearchDictionaryEntries` (read), delete via batch with `deleteEntry` action.
  - Notes: entry ID + dictionary type (`stopwords`|`plurals`|`compounds`) form the composite import ID (`<type>/<objectID>`); language is per-entry.
  - Follow the Definition of Done.
- [x] **P1.2 — `algolia_dictionary_settings` (resource).** App-level `disableStandardEntries` per language.
  - Client: `SetDictionarySettings` / `GetDictionarySettings`.
  - Notes: singleton per app — import ID can be the app ID; `Delete` resets to defaults.
- [x] **P1.3 — `algolia_allowed_sources` (resource).** API-access IP allowlist.
  - Client: `GetSources` (read), `ReplaceSources` (set), `AppendSource`/`DeleteSource` (incremental).
  - Notes: decide singleton-full-list vs. per-source resource; recommend full-list resource managing the complete set via `ReplaceSources`.
  - Shipped as a full-list singleton (resource + data source). Note: `ReplaceSources` rejects an empty `source` slice client-side, so `Delete` clears the allowlist via per-entry `DeleteSource` calls instead of `ReplaceSources([])`.
- [x] **P1.4 — Search lookup data sources.** Add `algolia_api_key` (`GetApiKey`), `algolia_indices` (`ListIndices`), `algolia_api_keys` (`ListApiKeys`). Data-source-only; no CRUD.
  - Shipped: `algolia_api_key`/`algolia_api_keys` in `internal/services/apikey/`, `algolia_indices` in `internal/services/index/`. `ListIndices` is paginated (`nbPages`); the data source pages through `WithPage` until every page is fetched.

## Phase 2 — Net-new configuration APIs

All clients ship in v4 already. **Do Ingestion first** — it is highest-value and sets the multi-resource template.

- [x] **P2.0 — Ingestion service scaffolding (depends: P0.1).** Create `internal/services/ingestion/`, wire the `ingestion` client into `ProviderData` (region-routed — reuse `analyticsregion`), and add a shared `client.go` helper. Blocks P2.1a–e.
  - Shipped: `analyticsregion.NewIngestionClient` (mirrors `NewQuerySuggestionsClient`/`NewPersonalizationClient`) plus `internal/services/ingestion/client.go`, a `base` struct (appID/apiKey/analyticsRegion + `configure`/`client` helpers) the five upcoming resources/data sources embed, and a `configuredClient(*providertypes.ProviderData)` convenience helper. No client stored on `ProviderData` — built on demand, per the existing region-routed convention. No resource/data source yet.
- [ ] **P2.1a — `algolia_ingestion_authentication` (depends: P2.0).** `Create/Get/Update/DeleteAuthentication`. Handle secret/credential fields as `Sensitive`.
- [ ] **P2.1b — `algolia_ingestion_source` (depends: P2.0).** `Create/Get/Update/DeleteSource`.
- [ ] **P2.1c — `algolia_ingestion_destination` (depends: P2.0).** `Create/Get/Update/DeleteDestination`.
- [ ] **P2.1d — `algolia_ingestion_transformation` (depends: P2.0).** `Create/Get/Update/DeleteTransformation`.
- [ ] **P2.1e — `algolia_ingestion_task` (depends: P2.1b, P2.1c).** `Create/Get/Update/DeleteTask` + `EnableTask`/`DisableTask` (map to an `enabled` attribute). References a source and destination by ID. `runs`/`events` are runtime — **out of scope**.
- [ ] **P2.2 — `algolia_ab_test` (resource).** Build on `abtesting-v3` (region-routed).
  - Client: `AddABTests` (create), `GetABTest` (read), `StopABTest` (stop), `DeleteABTest` (delete). No update → mark mutable fields `RequiresReplace`.
  - Gate acceptance tests (test creation has cost/quota); document the env flag in `AGENTS.md`.
- [ ] **P2.3 — `algolia_recommend_rule` (resource + data source).**
  - Client: `BatchRecommendRules` (upsert), `GetRecommendRule` (read), `DeleteRecommendRule`, `SearchRecommendRules`. `GetRecommendations` is data-plane — out of scope.
  - Notes: composite ID `<indexName>/<model>/<objectID>` (model = `related-products`|`bought-together`|…).
- [ ] **P2.4 — `algolia_composition` + `algolia_composition_rule` (resources + data sources).**
  - Client: `PutComposition`/`GetComposition`/`DeleteComposition`/`ListCompositions`; rules via `PutCompositionRule`/`GetRule`/`SaveRules`/`SearchCompositionRules`/`DeleteCompositionRule`. Use `GetTask` for task-wait.

## Phase 3 — APIs needing extra client work

- [ ] **P3.1 — Crawler support.** The Crawler is **not** in the Go v4 client.
  - [ ] **P3.1a — Custom client.** Add `internal/services/crawler/client.go` — a lightweight `net/http` client mirroring `internal/services/agent/client.go` (crawler API host + auth).
  - [ ] **P3.1b — `algolia_crawler` (resource + data source) (depends: P3.1a).** Manage crawler config (create/update/delete + read). Store the crawler config JSON as a JSON-encoded `types.String` if the schema is large/dynamic (mirror the index JSON-field pattern).
- [ ] **P3.2 — (Spike) Advanced Personalization.** Not in v4 client. Investigate how `advanced-personalization` relates to the existing `algolia_personalization_strategy`. Output: new resource vs. evolve existing vs. blocked-on-client. Non-blocking; decide before implementing.

## Phase 4 — Read-only observability (data sources only)

Low priority / demand-driven. Data-source-only — no CRUD, no import.

- [ ] **P4.1 — Analytics data sources.** e.g. `algolia_analytics_top_searches`, `_no_results`, `_click_through_rate`, etc. (`analytics` client, region-routed). Start with the 2–3 highest-demand metrics rather than all 20 endpoints.
- [ ] **P4.2 — Monitoring data sources.** e.g. `algolia_monitoring_status`, `_incidents`, `_latency` (`monitoring` client). Note: monitoring may require a separate monitoring API key — surface that in the provider/config.

## Explicitly NOT tasks (out of scope)

Do not create resources for these — they are runtime/data-plane operations with no reconcilable state (see ROADMAP.md § Explicitly out of scope):

- Search / browse / facet queries, and object (record) writes.
- Insights `PushEvents` / `DeleteUserToken`.
- Recommend `GetRecommendations`.
- Secured API key generation (`GenerateSecuredApiKey`).
- MCM user-ID assignment (cluster *listing* as a data source is fine — that's P1.4 territory).
