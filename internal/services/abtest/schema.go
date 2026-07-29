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
			"Import: `terraform import` reconstructs `variants`, `metrics` and `configuration` in the shape " +
			"the create endpoint accepts, rather than the enriched shape the API answers with, so a " +
			"configuration describing the same test matches the imported state. A difference in formatting " +
			"or key order alone does not force a replace. See each attribute's description.",
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
					"`algolia_ab_test` data source to read those. On `terraform import` the enriched response is " +
					"reduced to just the keys the create endpoint accepts, so a configuration describing the " +
					"same variants matches without hand reconciliation.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					suppressEquivalentJSON(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"metrics": schema.StringAttribute{
				Description: "JSON-encoded array of metrics to track for this A/B test, e.g. " +
					"`jsonencode([{ name = \"addToCartRate\" }, { name = \"revenue\", dimension = \"USD\" }])`. " +
					"Only these metrics are considered when calculating results. Changing this forces " +
					"replacement: the A/B Testing API has no update endpoint. Write-once: never refreshed from " +
					"`GetABTest`, which does not return the metrics list that was submitted at creation - only " +
					"per-metric *results* nested under each variant. Recovered on `terraform import` from those " +
					"results, which carry each metric's `name` and `dimension` - exactly the two fields the " +
					"create endpoint accepts - but only once the test has gathered data: a test that has just " +
					"been created reports no results, and this is then left null for you to supply.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					suppressEquivalentJSON(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"configuration": schema.StringAttribute{
				Description: "JSON-encoded A/B test configuration, e.g. `jsonencode({ minimumDetectableEffect " +
					"= { size = 0.1, metric = \"conversionRate\" }, errorCorrection = \"bonferroni\" })`. Changing " +
					"this forces replacement: the A/B Testing API has no update endpoint. Write-once: never " +
					"refreshed from `GetABTest` once set (see the resource description), and recovered on " +
					"`terraform import`, since `GetABTest` echoes this back in the same shape it was submitted.\n\n" +
					"Computed as well as optional, because Algolia substitutes its own configuration when none is " +
					"given: a test created without this attribute comes back with an `errorCorrection` and an " +
					"`isOutlier` filter applied. Those server-chosen values are what get stored, so state reflects " +
					"the test as it actually exists rather than claiming no configuration at all. Removing the " +
					"attribute from a configuration that had it therefore does not reset it; set it explicitly to " +
					"the value you want instead.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					suppressEquivalentJSON(),
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Status of the A/B test: `active`, `stopped`, `expired`, or `failed`. Refreshed " +
					"from `GetABTest` on every read.",
				Computed: true,
			},
			"created_at": schema.StringAttribute{
				Description: "Date and time the A/B test was created, in RFC 3339 format.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Date and time the A/B test was last updated, in RFC 3339 format.",
				Computed:    true,
			},
			"stopped_at": schema.StringAttribute{
				Description: "Date and time the A/B test was stopped, in RFC 3339 format, or null while it is " +
					"still running.",
				Computed: true,
			},
		},
	}
}
