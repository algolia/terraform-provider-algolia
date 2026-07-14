package recommend

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func recommendRuleDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read an Algolia Recommend rule for a specific index and model.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Description: "Terraform identifier in the form <index_name>/<model>/<object_id>.",
				Computed:    true,
			},
			"index_name": datasourceschema.StringAttribute{
				Description: "The index the Recommend rule applies to.",
				Required:    true,
			},
			"model": datasourceschema.StringAttribute{
				Description: "Recommend model the rule applies to.",
				Required:    true,
			},
			"object_id": datasourceschema.StringAttribute{
				Description: "Unique identifier of the Recommend rule.",
				Required:    true,
			},
			"condition": datasourceschema.StringAttribute{
				Description: "JSON-encoded condition that triggers the rule (`context`/`filters`).",
				Computed:    true,
			},
			"consequence": datasourceschema.StringAttribute{
				Description: "JSON-encoded effect of the rule (`hide`/`promote`/`params`).",
				Computed:    true,
			},
			"description": datasourceschema.StringAttribute{
				Description: "Human-readable description of the rule's purpose.",
				Computed:    true,
			},
			"enabled": datasourceschema.BoolAttribute{
				Description: "Whether the rule is active.",
				Computed:    true,
			},
			"validity": datasourceschema.StringAttribute{
				Description: "JSON-encoded array of time periods during which the rule is active.",
				Computed:    true,
			},
		},
	}
}
