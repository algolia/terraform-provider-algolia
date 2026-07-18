# Ingestion pipeline

An end-to-end example of Algolia's Ingestion API modelled as code. It wires the
four pieces of a pipeline together so records flow automatically from an
external feed into a search index:

```
source ──▶ transformation ──▶ destination ──▶ index
                                    ▲
                                    │
                                   task  (schedules the run)
```

| Resource | Role |
|---|---|
| `algolia_ingestion_source.products_csv` | Reads a CSV product feed over HTTPS. |
| `algolia_ingestion_transformation.stamp_indexed_at` | Adds an `indexedAt` timestamp to each record. |
| `algolia_ingestion_destination.products_index` | Writes to the `products` index; applies the transformation on the way in via `transformation_ids`. |
| `algolia_ingestion_task.nightly_sync` | Runs `source -> destination` every night at 02:00, fully replacing the index contents. |

The task references `source_id` and `destination_id`, and the destination
references the transformation's `transformation_id`, so Terraform creates them
in the right order automatically.

## Usage

```bash
export TF_VAR_algolia_app_id="YourApplicationID"
export TF_VAR_algolia_api_key="your-admin-api-key"

terraform init
terraform apply \
  -var 'analytics_region=us' \
  -var 'products_csv_url=https://your-host.example.com/products.csv'
```

## Notes

- **Region-routed.** The Ingestion API requires `analytics_region` on the
  provider (or the `ALGOLIA_ANALYTICS_REGION` environment variable). Set it to
  the region your application lives in.
- `action = "replace"` clears the destination index on each run. Use `"save"`
  to upsert instead, or `"partial"` for partial record updates.
- `action` and `source_id` force resource replacement if changed: the
  Ingestion API's task update endpoint cannot change either after creation.
- Drop `cron` to make the task on-demand (it then only runs when triggered
  manually from the dashboard or API) - useful for `push` sources.
