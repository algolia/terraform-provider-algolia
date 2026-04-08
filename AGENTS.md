# AGENTS.md

This file provides guidance to AI coding agents working with code in this repository.

## Build & Test Commands

```bash
make build                    # compile the provider
make test                     # run unit tests only (no Algolia credentials needed)
make testacc                  # run acceptance tests (requires ALGOLIA_APP_ID & ALGOLIA_API_KEY; some resources also require ALGOLIA_ANALYTICS_REGION)
make lint                     # run golangci-lint
make generate                 # regenerate docs via tfplugindocs

# run a single test
go test ./internal/services/index/ -run TestExpandTypoTolerance -v

# run all unit tests (skip acceptance)
go test ./... -v -timeout 120m
```

Acceptance tests require `TF_ACC=1` and valid `ALGOLIA_APP_ID`/`ALGOLIA_API_KEY` environment variables. Region-routed services such as Query Suggestions and Personalization also require `ALGOLIA_ANALYTICS_REGION`. Personalization acceptance tests are additionally gated behind `ALGOLIA_RUN_PERSONALIZATION_ACC=1` because the API enforces a daily strategy-save quota. Without the required environment variables, tests are skipped automatically.

## Architecture

This is a Terraform provider built on the **Terraform Plugin Framework** (not the older SDK v2). It uses the **Algolia Go client v4** (`algoliasearch-client-go/v4`).

**Registry address:** `registry.terraform.io/algolia/algolia`

### Package Layout

- `main.go` — Entry point, serves the provider via `providerserver.Serve`
- `internal/provider/` — Provider definition (schema, configure, resource/datasource registration)
- `internal/types/` — Shared `ProviderData` struct (extracted to break import cycle between provider and services)
- `internal/analyticsregion/` — Shared helpers for region-routed Algolia APIs such as Query Suggestions and Personalization
- `internal/services/` — Service packages by API surface, including `index`, `agent`, `agentprovider`, `apikey`, `personalization`, `querysuggestions`, `rule`, and `synonym`

### Index Service Files (internal/services/index/)

The index package follows a clear separation:

- **model.go** — Terraform model structs (`IndexResourceModel` + 10 block models like `AttributesModel`, `RankingModel`, etc.) with `tfsdk` tags
- **schema.go** — Resource schema definition with 10 `SingleNestedBlock`s and validators
- **data_source_schema.go** — Data source schema (mirrors resource but all attributes are Computed-only)
- **model_expand.go** — Terraform model → Algolia `*search.IndexSettings` (the "expand" direction)
- **model_flatten.go** — Algolia `*search.SettingsResponse` → Terraform model (the "flatten" direction)
- **resource.go** — CRUD operations (Create/Read/Update/Delete/Import)
- **data_source.go** — Read-only data source

### Key Design Patterns

**Union types:** Algolia's API has union types (`TypoTolerance`, `Distinct`, `IgnorePlurals`, `RemoveStopWords`, `OptionalWords`, `ReRankingApplyFilter`). These are mapped to simpler Terraform types:

- `TypoTolerance` (bool|enum) → `types.String` with values `"true"/"false"/"min"/"strict"`
- `Distinct` (bool|int32) → `types.Int64` where 0=false, 1=true, 2+=group sizes
- `IgnorePlurals`/`RemoveStopWords` (bool|[]language) → split into two fields: a `types.Bool` and a `types.List` for languages

**JSON-encoded fields:** Complex nested types (`decompounded_attributes`, `custom_normalization`, `user_data`, `semantic_search`, `re_ranking_apply_filter`) are stored as JSON-encoded `types.String` in Terraform state.

**Deletion protection:** Indexes default to `deletion_protection = true`. The Delete operation checks this before calling the API.

**Settings-as-index:** Algolia auto-creates an index on first `SetSettings` call—there's no separate "create index" API. The resource's Create simply calls `SetSettings` + `WaitForTask`.

### Adding a New Resource

1. Create a new package under `internal/services/`
2. Implement model, schema, expand/flatten, resource, and data source files
3. Register in `internal/provider/provider.go` (Resources/DataSources methods)
4. Use `*providertypes.ProviderData` from `internal/types/` to access the Algolia client
5. If the API is region-routed, use `internal/analyticsregion` plus `ProviderData.AnalyticsRegion` instead of embedding per-resource region config
