package querysuggestions

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
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
				Description: "Languages used to deduplicate singular and plural suggestions. Mutually " +
					"exclusive with `all_languages`, which covers every supported language instead of an " +
					"explicit list.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"all_languages": schema.BoolAttribute{
				Description: "Whether to deduplicate singular and plural suggestions in every language the " +
					"Query Suggestions API supports. The API models `languages` as either a list of languages " +
					"or the boolean `true`, so this attribute is the boolean form of `languages` and the two " +
					"are mutually exclusive.",
				Optional: true,
				Validators: []validator.Bool{
					boolvalidator.ConflictsWith(path.MatchRoot("languages")),
				},
			},
			"exclude": schema.SetAttribute{
				Description: "Words and patterns to exclude from the Query Suggestions index.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"enable_personalization": schema.BoolAttribute{
				Description: "Whether to turn on personalized query suggestions. Optional and computed: " +
					"when omitted the value currently configured for the index (for example through the " +
					"Algolia dashboard) is kept, because updating a Query Suggestions configuration replaces " +
					"it in full.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_special_characters": schema.BoolAttribute{
				Description: "Whether to include suggestions containing special characters. Optional and " +
					"computed: when omitted the value currently configured for the index (for example " +
					"through the Algolia dashboard) is kept, because updating a Query Suggestions " +
					"configuration replaces it in full.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
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
						"replicas": schema.BoolAttribute{
							Description: "Whether Query Suggestions uses all replica indices of this source " +
								"index to find popular searches. Optional and computed: when omitted the " +
								"value currently configured for the source index is kept, because updating a " +
								"Query Suggestions configuration replaces it in full.",
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
						"analytics_tags": schema.SetAttribute{
							Description: "Analytics tags used to filter popular searches.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"min_hits": schema.Int64Attribute{
							Description: "Minimum hits required for a query to become a suggestion. Optional " +
								"and computed: the Query Suggestions API applies its own default when this is " +
								"omitted, and reports that default back.",
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"min_letters": schema.Int64Attribute{
							Description: "Minimum letters required for a query to become a suggestion. Optional " +
								"and computed: the Query Suggestions API applies its own default when this is " +
								"omitted, and reports that default back.",
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
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
