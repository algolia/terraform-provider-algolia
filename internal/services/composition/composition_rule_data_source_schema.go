package composition

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func compositionRuleDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read an Algolia composition rule for a specific composition.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Description: "Terraform identifier in the form <composition_id>/<object_id>.",
				Computed:    true,
			},
			"composition_id": datasourceschema.StringAttribute{
				Description: "The composition the rule applies to.",
				Required:    true,
			},
			"object_id": datasourceschema.StringAttribute{
				Description: "Unique identifier of the composition rule.",
				Required:    true,
			},
			"conditions": datasourceschema.StringAttribute{
				Description: "JSON-encoded array of conditions that trigger the rule.",
				Computed:    true,
			},
			"consequence": datasourceschema.StringAttribute{
				Description: "JSON-encoded effect of the rule (a `behavior` override).",
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
			"tags": datasourceschema.ListAttribute{
				Description: "Tags used to categorize the rule.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}
