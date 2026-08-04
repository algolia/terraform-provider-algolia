## Unreleased

BUG FIXES:

- **`algolia_ab_test` silently dropped `customSearchParameters` from its variants.** The
  generated union type in the Algolia client populates both of its arms when a variant carries
  `customSearchParameters`, and serialises the one without them, so the parameters never
  reached Algolia. Where the variants also differed by index the API accepted the request and
  ran a test that exercised none of the requested parameters; variants sharing an index failed
  instead with `An A/B test variant must have a unique index or custom search parameters`. The
  provider now selects the union arm itself. A test pins the serialised request, since asserting
  the decoded value passed while the wire format was wrong.

- **Importing a plain `algolia_ab_test` no longer adds empty custom search parameters.** The
  live API returns `customSearchParameters: {}` even when a variant did not configure them;
  import now canonicalises that empty object to an absent field so the imported state matches
  the configuration that created the test.

NOTES:

- The `algolia_ab_test` example no longer sets `minimumDetectableEffect`, which the API rejects
  unless sample sizes accompany it, and takes its `end_at` as a variable: a test may not run for
  more than 90 days, so a fixed date in an example expires. Its two experiments also target
  different indices, because Algolia refuses a second test on an index that already has one.

## 0.1.1 (August 3, 2026)

NOTES:

- **Installation now has its own guide, `INSTALL.md`.** It covers the no-checkout installer,
  doing it by hand on macOS, Linux and Windows, and what to check when `terraform init`
  cannot find the provider. The README links to it rather than carrying the detail.

- **An agent skill ships in `skills/algolia-terraform-provider`.** It teaches a coding agent
  to install the provider and to get the attribute shapes right, and installs into any
  supported agent with `npx skills add algolia/terraform-provider-algolia`.

- **`scripts/install.sh` no longer resolves a draft release.** Releases are created as drafts
  and GitHub lists them to anyone with write access, so running the installer between a
  release build and its publish could install an unpublished version. It now passes
  `--exclude-drafts`. Its `--help` also no longer describes `--dev-overrides` as installing a
  locally built binary, which it never did; it installs the released one.

## 0.1.0 (August 3, 2026)

First release. There is no earlier version to upgrade from, so this
entry describes what the provider does and how it behaves rather than what changed. The
development history, including the earlier `v0.1.0-beta.1` pre-release, is in the git log
and in the pull requests.

FEATURES:

21 resources and 26 data sources across fifteen Algolia API surfaces. Every resource
supports `terraform import`; see `docs/` for each one's arguments and attributes.

- **Search:** `algolia_index`, `algolia_virtual_index`, `algolia_rule`, `algolia_synonym`,
  `algolia_api_key`, `algolia_dictionary_entry`, `algolia_dictionary_settings`
- **Ingestion:** `algolia_ingestion_source`, `algolia_ingestion_destination`,
  `algolia_ingestion_task`, `algolia_ingestion_transformation`,
  `algolia_ingestion_authentication`
- **Agent Studio:** `algolia_agent`, `algolia_agent_provider`
- **Relevance and personalization:** `algolia_query_suggestions`, `algolia_recommend_rule`,
  `algolia_personalization_strategy`, `algolia_ab_test`
- **Compositions:** `algolia_composition`, `algolia_composition_rule`
- **Security:** `algolia_allowed_sources`
- **Read-only data sources** additionally cover listing and multi-cluster inspection:
  `algolia_indices`, `algolia_api_keys`, `algolia_agent_provider_models`,
  `algolia_clusters`, `algolia_user_ids`

Built on the Terraform Plugin Framework, using the Algolia Go client v4.

NOTES:

Behaviour worth knowing before writing a configuration. Most of it exists because Algolia's
API does something a Terraform user would not predict.

- **`deletion_protection` defaults to `true` on nine resources**, and `terraform destroy`
  refuses until it is set to `false` and applied. It covers `algolia_index`,
  `algolia_virtual_index`, `algolia_agent`, `algolia_api_key` and the five
  `algolia_ingestion_*` resources: those whose deletion a re-apply cannot undo. An API key's
  id *is* the credential, so a replacement is a different secret and everything holding the
  old one breaks at once; an ingestion task is a running pipeline. Resources the
  configuration fully describes, such as rules and synonyms, are deliberately unguarded,
  since re-applying restores them. Algolia does not store the flag, so a value missing from
  state reads as protected rather than unprotected.

- **A primary index's replicas are owned by two resources, split by kind.**
  `algolia_index`'s `advanced.replicas` owns the standard replicas and rejects a
  `virtual(...)` entry at plan time; each virtual replica is declared by its own
  `algolia_virtual_index`, which maintains its own entry in the same Algolia field. Writing
  the list preserves the virtual entries Algolia reports, and reading it back reports only
  the standard ones.

- **A standard replica is not accepted as a virtual one.** Algolia reports a primary index
  for both kinds, but it copies the records into a standard replica while a virtual one is
  only a view. Adopting one as the other would put a record-bearing index under a resource
  documented as managing a view, with `deletion_protection` guarding what its owner had
  reason to believe was empty. Import and apply refuse; a refresh warns, and the next plan
  proposes the repair.

- **Reference a replica resource rather than repeating its name.** Where an index is managed
  by its own `algolia_index` and also named in another index's `advanced.replicas`, write
  `replicas = [algolia_index.example.name]`. Both writes create the same index, and with no
  dependency between them Terraform runs them concurrently, which Algolia handles by
  restarting that index's task queue. The provider recovers by re-sending the write, costing
  about half a minute; the reference avoids the collision and also orders the destroy
  correctly.

- **Destroying an index that is, or recently was, part of an A/B test can fail on the first
  attempt.** Algolia accepts the delete and publishes the task while the index survives, so
  the provider re-checks and raises `Index still exists after deletion` rather than
  reporting a success that did not happen. Destroying it again succeeds once nothing
  references it.

- **Several complex settings are JSON-encoded strings** rather than nested blocks:
  `decompounded_attributes`, `custom_normalization`, `user_data`, `semantic_search` and
  `re_ranking_apply_filter`. Algolia's union types are flattened as well: `typo_tolerance`
  is a string taking `"true"`, `"false"`, `"min"` or `"strict"`, and `distinct` is an
  integer where 0 disables it, 1 means one result per value, and 2 or more sets a group
  size.

- **Attributes that can carry credentials are marked sensitive.** `input` on
  `algolia_ingestion_source` is sensitive on the resource and on the data source, because
  the API returns it unredacted and several source types carry secrets in it;
  `algolia_ingestion_authentication`'s `input` is sensitive on the resource, and its data
  source does not expose the attribute at all. `algolia_api_key`'s `id` is the key value
  itself and is likewise sensitive.

- **`crawler_user_id` and `crawler_api_key` on the provider are deprecated and configure
  nothing.** No crawler resource or data source exists and none is planned; they remain so
  that a configuration already setting them keeps planning.

- **Schemas are all at version 0 and no resource implements `UpgradeState`.** Removing,
  renaming or retyping an attribute in a later version will need a schema version bump and
  an upgrader. `TestSchemaSnapshot` fails on any schema change, so that moment is visible
  rather than discovered after a release.
