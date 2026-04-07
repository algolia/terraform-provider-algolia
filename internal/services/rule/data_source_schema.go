package rule

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func ruleDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read an Algolia Rule for a specific index.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Description: "Terraform identifier in the form <index_name>/<object_id>.",
				Computed:    true,
			},
			"index_name": datasourceschema.StringAttribute{
				Description: "The index that owns the rule.",
				Required:    true,
			},
			"object_id": datasourceschema.StringAttribute{
				Description: "Unique identifier of the rule.",
				Required:    true,
			},
			"description": datasourceschema.StringAttribute{
				Description: "Human-readable description of the rule.",
				Computed:    true,
			},
			"enabled": datasourceschema.BoolAttribute{
				Description: "Whether the rule is active.",
				Computed:    true,
			},
		},
		Blocks: map[string]datasourceschema.Block{
			"conditions": datasourceschema.ListNestedBlock{
				Description: "Conditions that trigger the rule.",
				NestedObject: datasourceschema.NestedBlockObject{
					Attributes: map[string]datasourceschema.Attribute{
						"pattern":      datasourceschema.StringAttribute{Computed: true},
						"anchoring":    datasourceschema.StringAttribute{Computed: true},
						"alternatives": datasourceschema.BoolAttribute{Computed: true},
						"context":      datasourceschema.StringAttribute{Computed: true},
						"filters":      datasourceschema.StringAttribute{Computed: true},
					},
				},
			},
			"consequence": datasourceschema.ListNestedBlock{
				Description: "Rule consequence definition.",
				NestedObject: datasourceschema.NestedBlockObject{
					Attributes: map[string]datasourceschema.Attribute{
						"params_json": datasourceschema.StringAttribute{Computed: true},
						"hide": datasourceschema.SetAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
						"user_data": datasourceschema.StringAttribute{Computed: true},
					},
					Blocks: map[string]datasourceschema.Block{
						"promote": datasourceschema.ListNestedBlock{
							Description: "Promoted object IDs and their position.",
							NestedObject: datasourceschema.NestedBlockObject{
								Attributes: map[string]datasourceschema.Attribute{
									"object_ids": datasourceschema.SetAttribute{
										Computed:    true,
										ElementType: types.StringType,
									},
									"position": datasourceschema.Int64Attribute{Computed: true},
								},
							},
						},
					},
				},
			},
			"validity": datasourceschema.ListNestedBlock{
				Description: "Time windows during which the rule is active.",
				NestedObject: datasourceschema.NestedBlockObject{
					Attributes: map[string]datasourceschema.Attribute{
						"from":  datasourceschema.StringAttribute{Computed: true},
						"until": datasourceschema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}
