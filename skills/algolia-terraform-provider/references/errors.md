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
| `401 The log processing region does not match` | `ALGOLIA_ANALYTICS_REGION` is wrong for the application. |

## A plan that never settles

If a second `terraform plan` after a successful apply still proposes changes, the
configuration and Algolia disagree about a value and applying again will not converge. Look
for a JSON-encoded attribute whose formatting or key order differs from what Algolia returns,
or a setting written into the wrong block. `algolia_ab_test` is the documented exception:
`variants`, `metrics` and `configuration` are never refreshed from the API, because the read
endpoint returns an enriched shape that would otherwise cause a perpetual diff.
