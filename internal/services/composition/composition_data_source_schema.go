package composition

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func compositionDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read an Algolia composition.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Description: "The composition's unique identifier. Same value as `object_id`.",
				Computed:    true,
			},
			"object_id": datasourceschema.StringAttribute{
				Description: "Unique identifier of the composition.",
				Required:    true,
			},
			"name": datasourceschema.StringAttribute{
				Description: "Composition name.",
				Computed:    true,
			},
			"description": datasourceschema.StringAttribute{
				Description: "Composition description.",
				Computed:    true,
			},
			"behavior": datasourceschema.StringAttribute{
				Description: "JSON-encoded composition behavior (`injection` or `multifeed`).",
				Computed:    true,
			},
			"sorting_strategy": datasourceschema.StringAttribute{
				Description: "JSON-encoded map of sorting labels to the indices/replicas that implement them.",
				Computed:    true,
			},
		},
	}
}
