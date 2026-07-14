package abtest

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func abTestDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read an Algolia A/B test's current state, including runtime " +
			"results. Unlike the `algolia_ab_test` resource - which never refreshes `variants`/`metrics`/" +
			"`configuration` from the API to avoid corrupting a write-once configuration with runtime data - " +
			"this data source always reflects `GetABTest`'s enriched response, including per-variant metric " +
			"results, as of the last `terraform apply`/`refresh`.",
		Attributes: map[string]datasourceschema.Attribute{
			"ab_test_id": datasourceschema.Int64Attribute{
				Description: "Unique identifier of the A/B test to read.",
				Required:    true,
			},
			"id": datasourceschema.StringAttribute{
				Description: "Terraform identifier for the resource. Equal to `ab_test_id` as a string.",
				Computed:    true,
			},
			"name": datasourceschema.StringAttribute{
				Description: "Name of the A/B test.",
				Computed:    true,
			},
			"status": datasourceschema.StringAttribute{
				Description: "Status of the A/B test: `active`, `stopped`, `expired`, or `failed`.",
				Computed:    true,
			},
			"end_at": datasourceschema.StringAttribute{
				Description: "End date and time of the A/B test, in RFC 3339 format.",
				Computed:    true,
			},
			"created_at": datasourceschema.StringAttribute{
				Description: "Date and time when the A/B test was created, in RFC 3339 format.",
				Computed:    true,
			},
			"updated_at": datasourceschema.StringAttribute{
				Description: "Date and time when the A/B test was last updated, in RFC 3339 format.",
				Computed:    true,
			},
			"stopped_at": datasourceschema.StringAttribute{
				Description: "Date and time when the A/B test was stopped, in RFC 3339 format. Null if the " +
					"test has not been stopped.",
				Computed: true,
			},
			"variants": datasourceschema.StringAttribute{
				Description: "JSON-encoded array of the A/B test's variants as returned by `GetABTest`, " +
					"including runtime results (per-variant metric values, significance, estimated sample " +
					"size) and, for search-parameter A/B tests, `customSearchParameters`.",
				Computed: true,
			},
		},
	}
}
