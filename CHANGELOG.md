## 0.1.0 (Unreleased)

FEATURES:

- **New Resource:** `algolia_index` - Manage Algolia index settings
- **New Data Source:** `algolia_index` - Read Algolia index settings

BREAKING CHANGES:

- `algolia_index`, `algolia_virtual_index`: `terraform destroy` is now refused when
  `deletion_protection` is absent from state, instead of proceeding as if it were `false`. State
  written by an earlier version's `terraform import` has no value for it, so destroying such a
  resource now requires setting `deletion_protection = false` and running one `terraform apply`
  first. This is deliberate: the previous behaviour deleted indexes the configuration had marked as
  protected.
- `algolia_ingestion_authentication`, `algolia_ingestion_source`: the `input` attribute is now
  decoded strictly against the credential shape implied by `type`. Keys that do not belong to that
  shape are rejected instead of being silently discarded. A configuration relying on the previous
  behaviour was already sending incomplete credentials to Algolia, so this surfaces a pre-existing
  fault rather than introducing one.

- `algolia_ingestion_source`: `input` is now marked sensitive, on both the resource and the data
  source, because several source types carry credentials in it - a `docker` source's `configuration`
  is an arbitrary map holding the connector's own secrets, and `csv`/`json` take a `url` that is
  commonly presigned - and the Ingestion API returns `input` unredacted. A configuration that passes
  the value onward, for example `output "x" { value = algolia_ingestion_source.s.input }`, now fails
  until the consumer is also marked `sensitive`. The value is still stored in plaintext in state;
  marking it sensitive only stops it rendering in plan output and logs.

- data source `algolia_index`: the `deletion_protection` attribute is removed. It is a provider-side
  guard on `terraform destroy` with no representation in the Algolia API, so a read-only data source
  had nothing to populate it from and it always returned null. A configuration referencing
  `data.algolia_index.x.deletion_protection` now fails to plan instead of silently reading null.

BUG FIXES:

- `algolia_virtual_index`: **an unlinked virtual replica no longer wedges Terraform.** When an index
  still existed but had stopped being a virtual replica - which is what a wholesale write of the
  primary's `replicas` list causes - `Read` raised an error. That error surfaced on every operation
  that refreshes, so plan, apply and `terraform destroy` all failed together and `terraform state rm`
  was the only way out. The resource is now dropped from state with a warning naming the likely
  cause, so the next apply re-links the replica in place. Import still fails loudly on an index that
  is not a virtual replica, since that is a mistaken import command rather than drift.
- `algolia_virtual_index`: **several virtual replicas on one primary no longer lose each other's
  links within one apply.** Each of them adds its own `virtual(...)` entry to the primary's single
  `replicas` setting, and Terraform applies resources concurrently, so the writes interleaved and the
  later ones dropped the earlier ones' entries. These read-modify-write cycles are now serialised per
  primary index, within one provider process - which covers a single `terraform apply`. Concurrent
  applies against the same primary from separate processes remain racy, as does the residual window
  left by Algolia's writes being asynchronous and its reads eventually consistent.
- `algolia_index`: **an `advanced.replicas` list that configuration never declared is no longer written
  back.** The attribute is Optional+Computed, so when configuration omits it Terraform fills the plan
  value from the last refresh - and expanding that plan sent the remembered list to Algolia on every
  update. Any virtual replica linked since that refresh was silently unlinked, and unlinking empties
  it. The list is now written only when configuration declares one, which is what Optional+Computed is
  meant to express; otherwise the field is omitted and Algolia keeps what the index already has.
- `algolia_index`: **a write to `advanced.replicas` no longer races virtual replica linking.** The
  write is now taken under the same per-primary lock that serialises `algolia_virtual_index`, so a
  replica link cannot land between the check below and the write that would drop it.
- `algolia_index`: **a write to `advanced.replicas` that unlinks a virtual replica now warns.**
  `advanced.replicas` and `algolia_virtual_index` write the same Algolia setting, so a list omitting
  a `virtual(...)` entry silently unlinked it - which empties that replica, since it is a view over
  the primary's records rather than a copy. The removal is still honoured, because an explicit list
  is a complete declaration and merging omitted entries back in would make removal impossible to
  express; it is no longer silent. Both schemas document the ownership rule.
- `algolia_ab_test`: **fixed `terraform destroy` failing and leaving indexes behind.** Every write on
  the A/B Testing API returns a task ID and only queues the work, which the client's own model spells
  out: "A successful API response means that a task was added to a queue. It might not run
  immediately." The provider ignored that task entirely, so destroying a test together with the
  indexes it referenced raced: the test was already gone from the API while Algolia still rejected
  deleting its indexes with `403 cannot delete with an index under AB testing index as destination`,
  failing the destroy and leaving the indexes for the operator to clean up. Reproduced, then
  confirmed by hand: the very same deletes succeeded once the queued task had run. Create and delete
  now wait for it.
- `algolia_ab_test`: **`terraform import` no longer proposes destroying a running experiment.**
  Import used to store the enriched read response, which matches no reasonable configuration, and
  left `metrics` null - and since `variants`, `metrics` and `configuration` are all Required or
  `RequiresReplace`, the first plan after importing proposed a replace, discarding the statistics the
  test had gathered. Import now reconstructs all three in the shape the create endpoint accepts:
  `variants` keeps only the keys that endpoint reads and drops the runtime enrichment, and `metrics`
  is rebuilt from the per-variant results, which carry each metric's `name` and `dimension`. A
  difference in formatting or key order alone no longer forces a replace either.
  One limit, verified against the API rather than assumed: metric results only exist once a test has
  gathered data, so importing a test created moments ago still cannot recover `metrics`. Importing one
  that has been running does.
- `algolia_ab_test`: `configuration` is now `Optional+Computed`, because Algolia substitutes its own
  when a test is created without one. State previously claimed a test had no configuration while the
  API had applied an `errorCorrection` and an `isOutlier` filter, so an imported test could never
  match one created from a configuration that omitted the attribute. A configured document is still
  kept verbatim; only an absent one is filled from the response.
- `algolia_ab_test`: `created_at`, `updated_at` and `stopped_at` are now exposed on the resource, not
  just the data source.
- `algolia_query_suggestions`: added `all_languages`, so the boolean arm of the API's `languages`
  union is expressible. `languages` accepts either a list of language codes or `true` meaning every
  supported language, and only the list was reachable. The two arms are mutually exclusive and
  validated at plan time.
- `algolia_api_key`: **fixed a `terraform apply` silently removing a key's tenant restriction.**
  `queryParameters` - the field that scopes a search key to specific filters or sources - had no
  attribute, and because `UpdateApiKey` resets what it is not sent, changing any unrelated field
  stripped it. Reproduced end to end: a key restricted to `filters=tenant%3Aacme` was imported, only
  its `description` was edited, the plan showed just that edit, and after the apply the restriction
  was gone. The field is now exposed as `query_parameters`, so it is recorded in state and any
  removal appears in the plan. `indexes`, `referers`, `max_hits_per_query` and
  `max_queries_per_ip_per_hour` are reset by the same endpoint but were already read back into state,
  so their removal was always visible; that is now covered by a test.
- `algolia_api_key`: **fixed an expiring key silently becoming permanent.** `terraform import` left
  `expires_at` null, and since a validity is only sent when `expires_at` is known, the next apply
  reset the key to never expire with nothing in state or plan to show it. `expires_at` is now derived
  from the key's remaining validity, and a configured timestamp is kept verbatim when it denotes the
  same instant, so an unchanged key does not churn while a life shortened out of band shows as drift.
- `algolia_api_key`: `created_at` reported January 1970 for every key. The value is in seconds, but
  the generated client documents it as milliseconds and the provider read it that way.
- `algolia_api_key`: the wait for a new or deleted key to propagate is now interruptible. It polls
  through the client's own `WaitForApiKey`, which was called without a context and so ignored a
  cancelled `plan` or `apply` for up to several minutes; the update and delete calls likewise had no
  context.
- `algolia_agent`: **fixed four attributes that made an apply fail or silently lost data.**
  `tool_client_side.input_schema` was round-tripped through a client struct modelling only `type`,
  `properties` and `required`, so every other JSON Schema keyword was dropped before the request was
  sent; `tool_algolia_search.index.search_parameters` lost any parameter the vendored client does not
  model yet, from the request as well as from state. Both are now spliced into the request as the
  user's own bytes. Separately, Algolia does not return these documents as they were written - it
  strips schema keywords it does not model, expands search parameters into the full schema with every
  unset field explicitly null, and merges its own defaults into `config` - so for `input_schema`,
  `search_parameters` and `config` the configured document is what state records, and the API's is
  used only where there is none, on import and data source reads. As with
  `ranking.relevancy_strictness` on `algolia_index`, that means out-of-band drift on these attributes
  is not reported; the API does not say what it stored, so there is nothing to compare against.
  `config` and `tool_algolia_recommend.predefined_recommend_parameters` additionally turned a
  configured `jsonencode({})` into null.
- `algolia_rule`: `consequence.user_data` was re-encoded from the decoded response, so a configured
  document came back with its keys reordered and its whitespace gone, and an integer above 2^53 came
  back with different digits - a silent change to the value in state, not just a rejected apply. The
  configured document is now kept whenever it carries the same data.
- `algolia_rule`: `validity.from` and `validity.until` were rewritten as UTC whole seconds, so a
  window written as `2030-01-01T00:00:00+02:00` applied as `2029-12-31T22:00:00Z` and fractional
  seconds were dropped, either of which Terraform rejects as an inconsistent result. The configured
  string is now kept when it denotes the instant the API returned.
- `algolia_allowed_sources`: a `source` entry whose `description` was set to the empty string applied
  as null and aborted the apply, which a `description = each.value.description` resolving to `""`
  would hit. The applied state also no longer comes from a read-back: `ReplaceSources` has already
  accepted the planned set, so refreshing could only agree with it or write a value Terraform
  rejects, and a failed read-back used to abandon a change that had already been made remotely.
- data source `algolia_index`: `entries`, `data_size`, `created_at` and `updated_at` are now
  populated. The data source only called `GetSettings`, which does not return them, so all four were
  permanently null - an index holding 10,000 records reported `entries = null`. They come from the
  index listing, the same way the `algolia_index` resource has always read them.
- `algolia_index`, `algolia_virtual_index`, and the `algolia_ingestion_*` resources: the request
  context is now passed to every Algolia API call, so a cancelled `terraform plan` or `apply`
  (Ctrl-C) stops the in-flight request instead of running to completion. `algolia_virtual_index` had
  no context on any of its nine call sites, and the ingestion resources passed one on their read-back
  call but not on the update or delete that preceded it. Reading index metadata now also pages
  through the whole index listing rather than its first page, so an application with more indexes
  than fit on one page no longer reports zeroes for a listed index.
- `algolia_recommend_rule`: an unrecognised error response body is now flattened, truncated and
  UTF-8 sanitised before it reaches a diagnostic, instead of being interpolated whole. A `200 null`
  response is now an error rather than a rule with an empty `object_id`.
- `algolia_ingestion_source`: `input` is omitted from an update request when it has not changed,
  so a source whose type has no update variant in the Algolia client (`bigcommerce`) can still have
  its other fields changed. Only an actual attempt to change such a source's `input` is refused. The
  comparison is semantic and treats an array of scalars as unordered, so a change that only reorders
  such an array is not sent to the API.
- `algolia_index`: **fixed silent deletion of protected indexes.** `terraform import` left
  `deletion_protection` unset, and the delete guard read that absent value as `false`, so an
  `import` followed by `destroy` permanently deleted the index and its records even with
  `deletion_protection = true` in the configuration. Import now seeds the documented default
  (`true`), and both `algolia_index` and `algolia_virtual_index` refuse to delete on an absent value.
- `algolia_index`: **`terraform import` now imports index settings.** Import previously produced a
  state containing only the index name and timestamps: all ten settings blocks were read from the
  API and then discarded, so existing indexes could not be adopted. Imported state populates every
  settings block while an applied state leaves blocks absent from the configuration null, so the
  first plan after importing an index whose configuration omits blocks proposes removing them; it
  converges on one apply. `ranking.relevancy_strictness` remains unimportable: Algolia accepts it on
  write but never returns it from `GetSettings`.
  `performance.allow_compression_of_integer_array` is returned when it is `true` and omitted when it
  is `false`, so importing an index that has it disabled cannot tell that apart from the setting
  being unset - harmless, since `false` is also the default.
- `algolia_api_key`: fixed `Provider produced inconsistent result after apply` on `referers` and
  `indexes` when either is omitted. The resource could not be created with an ordinary
  configuration, and because the failed apply tainted the resource, every subsequent apply destroyed
  and recreated the key, rotating its value while never converging.
- `algolia_rule`: fixed the same failure on `consequence.hide` when omitted.
- `algolia_composition_rule`: fixed the mirror-image failure on `tags`, where an explicitly
  configured `tags = []` was flattened to null and aborted the apply.
- `algolia_ingestion_authentication`, `algolia_ingestion_source`: **fixed credentials being replaced
  with empty values.** The Algolia client's `AuthInput` and `SourceInput` unions carry no
  discriminator, so every variant decoded successfully and re-marshalling emitted the wrong one:
  `basic`, `oauth` and `googleServiceAccount` credentials were all sent to the API as
  `{"apiKey":"","appID":""}`. The variant is now selected from the declared `type`, and an
  unrecognised type is a loud error rather than a silent substitution.
- `algolia_agent_provider`: fixed `azure_openai` being unusable. `azure_endpoint`,
  `azure_deployment` and `api_version` were dropped by the same union defect, so every apply failed
  and orphaned a provider.
- `algolia_recommend_rule`: **fixed the resource being entirely non-functional.** Every non-delete
  code path called `GetRecommendRule`, which cannot decode the API's numeric `_metadata.lastUpdate`
  into the client's `*string` field, so create, read, update, plan and import all failed, while
  create still left a rule behind in Algolia that Terraform had no record of. Responses are now
  decoded defensively, and create persists the rule's identity before waiting on the task so a later
  failure cannot orphan it.

NOTES:

- Waiting for an Algolia operation to be applied now goes through one shared loop,
  `internal/algoliawait`. The bounded deadline, backoff and interruptible sleep had been hand-copied
  into six resources, which is how one of them ended up with a bare `time.Sleep` that made a
  30-minute wait uncancellable. Seven call sites share it now. Three waiters that poll for a *state*
  rather than a task keep their own shorter budgets, since they are a different shape.
- The diagnostics a resource raises when an Algolia call fails are now built by
  `internal/algoliaerr` rather than re-templated in each resource. The wording is unchanged; the
  point is that a fix to one resource's error handling reaches its siblings, which this repository's
  history repeatedly did not. `algolia_api_key` is deliberately excluded: its object ID is a secret
  and its diagnostics route through dedicated redaction.
- **Known issue, found while testing the above and not yet fixed:** `terraform destroy` can report
  success while an index survives. Deleting an index that is or recently was part of an A/B test is
  accepted by Algolia, the deletion task reaches `published`, and the index is still there.
  `algolia_index` waits for that task and the wait behaves correctly, so nothing anywhere reports a
  problem. Deleting the index again later succeeds once nothing references it.
  This is reachable on an ordinary `terraform destroy` of an A/B test declared alongside its indexes,
  not only in some edge case: it was observed on repeated acceptance runs, each leaving its indexes
  behind while reporting success. Before the task-wait fix above, the same situation surfaced loudly
  as `403 cannot delete with an index under AB testing index as destination` and failed the destroy;
  now the destroy succeeds and the index quietly remains, which is a better outcome for the A/B test
  and a worse one for the index. It is also why an opt-in "stop instead of delete" option for
  `algolia_ab_test` was prepared and then withheld, since that option would make the window wider.
- The provider's `crawler_user_id` and `crawler_api_key` arguments are deprecated. No crawler
  resource or data source exists and none is planned (descoped 2026-07-18), so both configure
  nothing. They are deprecated rather than removed so that a configuration already setting them keeps
  planning.
- 404 detection now goes through the shared `internal/algoliaerr` package at every call site. It was
  introduced for that purpose but adopted in only three files, leaving 36 hand-written `errors.As`
  checks across 17 others; a fix applied to one copy did not reach the rest.
- `ROADMAP.md` understated the provider as 12 resources and 15 data sources; it has 21 and 26. Its
  `agent-studio` row also claimed the surface was complete while allowed domains and secret keys have
  full client CRUD and no provider surface at all, so a Terraform-published agent is not reachable
  from any origin without an out-of-band step. Both are corrected there.
- Recommend rule acceptance tests are no longer gated behind `ALGOLIA_RUN_RECOMMEND_ACC`. The gate had
  already been removed, but three comments and `AGENTS.md` still described it as active.
- Initial release of the Algolia Terraform provider
- Built on the Terraform Plugin Framework (not SDK v2)
- Uses Algolia Go client v4
