package abtest

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func abTestResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages an Algolia A/B test. The A/B Testing API is region-routed, so the provider's " +
			"`analytics_region` (or the `ALGOLIA_ANALYTICS_REGION` environment variable) must be configured.\n\n" +
			"The A/B Testing API has no update endpoint, so `name`, `end_at`, `variants`, `metrics`, and " +
			"`configuration` are all write-once: changing any of them forces the A/B test to be replaced " +
			"(stopped test data is not preserved by the API across a replace).\n\n" +
			"Read limitation: `GetABTest` returns a response enriched with runtime results (per-variant " +
			"conversion/click counts, significance, status) whose shape diverges from what was submitted to " +
			"create the test. To avoid corrupting state with that runtime data and causing a perpetual diff, " +
			"`variants`, `metrics`, and `configuration` are never refreshed from the API - the value you " +
			"configure is the value that stays in state. `name` and `end_at` *are* refreshed on read - the API " +
			"returns them in the same shape they were submitted - so changing either outside Terraform shows up " +
			"as drift and, since both force replacement, plans a replace. Use the `algolia_ab_test` data source " +
			"to inspect the enriched, read-only view of a test (including per-variant results).\n\n" +
			"Import limitation: `variants` and `metrics` cannot be perfectly reconstructed from the enriched " +
			"read response on `terraform import` (metrics in particular: `GetABTest` only returns per-metric " +
			"*results* nested under each variant, not the original metrics list). See each attribute's " +
			"description.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform identifier for the resource. Equal to `ab_test_id` as a string.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ab_test_id": schema.Int64Attribute{
				Description: "Unique identifier of the A/B test.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the A/B test. Changing this forces replacement: the A/B Testing API has " +
					"no update endpoint. Refreshed from `GetABTest` on every read, so a rename outside " +
					"Terraform is detected as drift.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"end_at": schema.StringAttribute{
				Description: "End date and time of the A/B test, in RFC 3339 format. Changing this forces " +
					"replacement: the A/B Testing API has no update endpoint. Refreshed from `GetABTest` on " +
					"every read, so a reschedule outside Terraform is detected as drift.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"variants": schema.StringAttribute{
				Description: "JSON-encoded array of A/B test variants, e.g. `jsonencode([{ index = \"prod\", " +
					"trafficPercentage = 50 }, { index = \"prod_variant\", trafficPercentage = 50 }])`. The first " +
					"variant is conventionally the control (typically the production index); the rest are " +
					"indexes with changed settings to test against it. Each variant needs `index` and " +
					"`trafficPercentage`, and may set `description` and, for search-parameter A/B tests, " +
					"`customSearchParameters`. Changing this forces replacement: the A/B Testing API has no " +
					"update endpoint. Write-once: never refreshed from `GetABTest`, whose response nests " +
					"per-variant runtime results (metrics, metadata) that don't match this shape - use the " +
					"`algolia_ab_test` data source to read those. Not perfectly recoverable on `terraform " +
					"import`: the imported value is derived from the enriched response and may need to be " +
					"reconciled with configuration by hand.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"metrics": schema.StringAttribute{
				Description: "JSON-encoded array of metrics to track for this A/B test, e.g. " +
					"`jsonencode([{ name = \"addToCartRate\" }, { name = \"revenue\", dimension = \"USD\" }])`. " +
					"Only these metrics are considered when calculating results. Changing this forces " +
					"replacement: the A/B Testing API has no update endpoint. Write-once: never refreshed from " +
					"`GetABTest`, which does not return the metrics list that was submitted at creation - only " +
					"per-metric *results* nested under each variant. Not recoverable on `terraform import`; " +
					"left null after import (see the resource description).",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"configuration": schema.StringAttribute{
				Description: "JSON-encoded A/B test configuration, e.g. `jsonencode({ minimumDetectableEffect " +
					"= { size = 0.1, metric = \"conversionRate\" }, errorCorrection = \"bonferroni\" })`. Changing " +
					"this forces replacement: the A/B Testing API has no update endpoint. Write-once: never " +
					"refreshed from `GetABTest` (see the resource description). Best-effort recovered on " +
					"`terraform import`, since `GetABTest` echoes this back in the same shape it was submitted.",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Status of the A/B test: `active`, `stopped`, `expired`, or `failed`. Refreshed " +
					"from `GetABTest` on every read.",
				Computed: true,
			},
		},
	}
}
