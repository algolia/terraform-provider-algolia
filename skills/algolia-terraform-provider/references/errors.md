# What an error means

| Message | Cause |
| --- | --- |
| `no available releases match the given constraints` | The pinned version is not in the mirror. Check what `install.sh` reported installing. |
| `does not support resource type "algolia_..."` for a resource that exists, or `Unsupported argument` for a documented argument | Terraform loaded a different build than the pinned version. An old unpacked build under `~/.terraform.d/plugins/registry.terraform.io/algolia/algolia/<version>/` shadows the release archive of the same version; remove it. See [INSTALL.md](https://github.com/algolia/terraform-provider-algolia/blob/main/INSTALL.md). |
| `Deletion Protection Enabled` | Set `deletion_protection = false` and apply that before destroying. |
| `Virtual replica declared on the wrong resource` | A `virtual(...)` entry is in `advanced.replicas`. Declare it as an `algolia_virtual_index` instead. |
| `Index is a standard replica` | Algolia holds the index as a standard replica, which carries a copy of the records, so `algolia_virtual_index` will not adopt it. |
| `Index still exists after deletion` | Algolia accepted the delete and the index survived, usually because an A/B test still references it. Destroy again once nothing does. |
| `cannot apply the deleteIndex operation on a replica index` | The index is still listed as a replica of a primary that is not itself going away. |
| A `401` from Query Suggestions, Personalization, A/B testing or Ingestion complaining about the region | `ALGOLIA_ANALYTICS_REGION` is unset or wrong for the application. These four APIs are region-routed; the message comes from Algolia, not the provider, so its exact wording can change. |

## A plan that never settles

If a second `terraform plan` after a successful apply still proposes changes, the
configuration and Algolia disagree about a value and applying again will not converge. Look
for a JSON-encoded attribute whose formatting or key order differs from what Algolia returns,
or a setting written into the wrong block.

Not every attribute is refreshed from Algolia, so some are exempt from that reasoning, and
drift in them is not reported. This is not an exhaustive list; each resource's docs page states
its own exceptions.

- `algolia_ab_test`: `variants` and `metrics` are never refreshed, because the read endpoint
  returns an enriched shape that would otherwise cause a perpetual diff. `configuration` is
  filled from the response only when the configuration does not set it; a value you did set is
  kept verbatim.
- `algolia_ingestion_authentication`: `input` is not refreshed, since Algolia redacts it.
- `algolia_ingestion_task`: `cursor` is never refreshed, because it advances on its own as the
  task runs.
- `algolia_agent`: a configured `config`, `search_parameters` or `input_schema` wins over what
  the API returns, because Agent Studio does not hand those documents back the way they were
  written.
- `algolia_agent_provider`: a provider block's `api_key` is preserved rather than read back.
- `algolia_index`, `algolia_virtual_index`: `ranking.relevancy_strictness` and
  `performance.allow_compression_of_integer_array` are kept from the plan, because Algolia
  accepts them on write but omits them from `GetSettings`.
