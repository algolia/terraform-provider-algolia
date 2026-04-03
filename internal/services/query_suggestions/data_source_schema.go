package query_suggestions

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func querySuggestionsConfigDataSourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Reads an Algolia Query Suggestions configuration.",
		Attributes: map[string]schema.Attribute{
			"index_name": schema.StringAttribute{
				Description: "Name of the Query Suggestions index (case-sensitive).",
				Required:    true,
			},
			"region": schema.StringAttribute{
				Description: "Algolia region where the Query Suggestions API is hosted. Must be one of: us, eu.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("us", "eu"),
				},
			},
			"languages": schema.ListAttribute{
				Description: "Language codes configured for deduplication.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"languages_enabled": schema.BoolAttribute{
				Description: "Whether language deduplication is enabled globally (bool mode).",
				Computed:    true,
			},
			"exclude": schema.ListAttribute{
				Description: "Queries excluded from the Query Suggestions index.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"enable_personalization": schema.BoolAttribute{
				Description: "Whether personalized query suggestions are enabled.",
				Computed:    true,
			},
			"allow_special_characters": schema.BoolAttribute{
				Description: "Whether suggestions with special characters are included.",
				Computed:    true,
			},
			"deletion_protection": schema.BoolAttribute{
				Description: "Always true when read from the data source.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"source_index": schema.ListNestedBlock{
				Description: "Source indices for query suggestions.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"index_name": schema.StringAttribute{
							Description: "Name of the source Algolia index.",
							Computed:    true,
						},
						"replicas": schema.BoolAttribute{
							Description: "Whether replica indices are included.",
							Computed:    true,
						},
						"analytics_tags": schema.ListAttribute{
							Description: "Analytics tags used for filtering.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"min_hits": schema.Int64Attribute{
							Description: "Minimum number of search results required.",
							Computed:    true,
						},
						"min_letters": schema.Int64Attribute{
							Description: "Minimum number of letters required.",
							Computed:    true,
						},
						"generate": schema.StringAttribute{
							Description: "JSON-encoded list of facet attribute groups for generated suggestions.",
							Computed:    true,
						},
						"external": schema.ListAttribute{
							Description: "External index names included as suggestion sources.",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
					Blocks: map[string]schema.Block{
						"facets": schema.ListNestedBlock{
							Description: "Facets used as categories for suggestions.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"attribute": schema.StringAttribute{
										Description: "Facet attribute name.",
										Computed:    true,
									},
									"amount": schema.Int64Attribute{
										Description: "Number of suggestions for this facet.",
										Computed:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
