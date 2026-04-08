package querysuggestions

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func querySuggestionsDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read an Algolia Query Suggestions configuration.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Description: "Terraform identifier in the form <index_name>.",
				Computed:    true,
			},
			"index_name": datasourceschema.StringAttribute{
				Description: "The Query Suggestions index name.",
				Required:    true,
			},
			"languages": datasourceschema.SetAttribute{
				Description: "Languages used to deduplicate singular and plural suggestions.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"exclude": datasourceschema.SetAttribute{
				Description: "Words and patterns to exclude from the Query Suggestions index.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
		Blocks: map[string]datasourceschema.Block{
			"source_indices": datasourceschema.ListNestedBlock{
				Description: "Source indices used to generate the Query Suggestions index.",
				NestedObject: datasourceschema.NestedBlockObject{
					Attributes: map[string]datasourceschema.Attribute{
						"index_name":     datasourceschema.StringAttribute{Computed: true},
						"analytics_tags": datasourceschema.SetAttribute{Computed: true, ElementType: types.StringType},
						"min_hits":       datasourceschema.Int64Attribute{Computed: true},
						"min_letters":    datasourceschema.Int64Attribute{Computed: true},
						"generate":       datasourceschema.ListAttribute{Computed: true, ElementType: types.ListType{ElemType: types.StringType}},
						"external":       datasourceschema.SetAttribute{Computed: true, ElementType: types.StringType},
					},
					Blocks: map[string]datasourceschema.Block{
						"facets": datasourceschema.ListNestedBlock{
							Description: "Facets used as categories for suggestions.",
							NestedObject: datasourceschema.NestedBlockObject{
								Attributes: map[string]datasourceschema.Attribute{
									"attribute": datasourceschema.StringAttribute{Computed: true},
									"amount":    datasourceschema.Int64Attribute{Computed: true},
								},
							},
						},
					},
				},
			},
		},
	}
}
