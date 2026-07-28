## 0.1.0 (Unreleased)

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

BUG FIXES:

- `algolia_ingestion_source`: `input` is omitted from an update request when it has not changed,
  so a source whose type has no update variant in the Algolia client (`bigcommerce`) can still have
  its other fields changed. Only an actual attempt to change such a source's `input` is refused.
- `algolia_index`: **fixed silent deletion of protected indexes.** `terraform import` left
  `deletion_protection` unset, and the delete guard read that absent value as `false`, so an
  `import` followed by `destroy` permanently deleted the index and its records even with
  `deletion_protection = true` in the configuration. Import now seeds the documented default
  (`true`), and both `algolia_index` and `algolia_virtual_index` refuse to delete on an absent value.
- `algolia_index`: **`terraform import` now imports index settings.** Import previously produced a
  state containing only the index name and timestamps: all ten settings blocks were read from the
  API and then discarded, so the plan immediately after an import was never clean and existing
  indexes could not be adopted. `ranking.relevancy_strictness` and
  `performance.allow_compression_of_integer_array` remain unimportable because Algolia accepts both
  on write but omits them from `GetSettings`.
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
  and orphaned a provider. `openai_compatible` no longer loses `default_model`.
- `algolia_recommend_rule`: **fixed the resource being entirely non-functional.** Every non-delete
  code path called `GetRecommendRule`, which cannot decode the API's numeric `_metadata.lastUpdate`
  into the client's `*string` field, so create, read, update, plan and import all failed, while
  create still left a rule behind in Algolia that Terraform had no record of. Responses are now
  decoded defensively, and create persists the rule's identity before waiting on the task so a later
  failure cannot orphan it.

NOTES:

- Initial release of the Algolia Terraform provider
- Built on the Terraform Plugin Framework (not SDK v2)
- Uses Algolia Go client v4
