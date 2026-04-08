package querysuggestions

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func querySuggestionsResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages an Algolia Query Suggestions configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform identifier in the form <index_name>.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"index_name": schema.StringAttribute{
				Description: "The Query Suggestions index name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"languages": schema.SetAttribute{
				Description: "Languages used to deduplicate singular and plural suggestions.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"exclude": schema.SetAttribute{
				Description: "Words and patterns to exclude from the Query Suggestions index.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"source_indices": schema.ListNestedBlock{
				Description: "Source indices used to generate the Query Suggestions index.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"index_name": schema.StringAttribute{
							Description: "Source Algolia index name.",
							Required:    true,
						},
						"analytics_tags": schema.SetAttribute{
							Description: "Analytics tags used to filter popular searches.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"min_hits": schema.Int64Attribute{
							Description: "Minimum hits required for a query to become a suggestion.",
							Optional:    true,
						},
						"min_letters": schema.Int64Attribute{
							Description: "Minimum letters required for a query to become a suggestion.",
							Optional:    true,
						},
						"generate": schema.ListAttribute{
							Description: "Facet combinations used to generate suggestions.",
							Optional:    true,
							ElementType: types.ListType{ElemType: types.StringType},
						},
						"external": schema.SetAttribute{
							Description: "External indices used to generate custom suggestions.",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
					Blocks: map[string]schema.Block{
						"facets": schema.ListNestedBlock{
							Description: "Facets to use as categories for suggestions.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"attribute": schema.StringAttribute{
										Description: "Facet attribute name.",
										Required:    true,
									},
									"amount": schema.Int64Attribute{
										Description: "Number of suggestions to generate for the facet.",
										Required:    true,
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
