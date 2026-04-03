package query_suggestions

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func querySuggestionsConfigResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages an Algolia Query Suggestions configuration.",
		Attributes: map[string]schema.Attribute{
			"index_name": schema.StringAttribute{
				Description: "Name of the Query Suggestions index (case-sensitive). Acts as the resource ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				Description: "Algolia region where the Query Suggestions API is hosted. Must be one of: us, eu.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("us", "eu"),
				},
			},
			"languages": schema.ListAttribute{
				Description: "Language codes for deduplicating singular and plural suggestions. " +
					"Mutually exclusive with languages_enabled. If neither is set, no language deduplication occurs.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"languages_enabled": schema.BoolAttribute{
				Description: "Enable (true) or disable (false) language deduplication for all languages. " +
					"Mutually exclusive with languages.",
				Optional: true,
			},
			"exclude": schema.ListAttribute{
				Description: "Queries to exclude from the Query Suggestions index.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"enable_personalization": schema.BoolAttribute{
				Description: "Whether to turn on personalized query suggestions.",
				Optional:    true,
				Computed:    true,
			},
			"allow_special_characters": schema.BoolAttribute{
				Description: "Whether to include suggestions with special characters.",
				Optional:    true,
				Computed:    true,
			},
			"deletion_protection": schema.BoolAttribute{
				Description: "Prevents accidental deletion when set to true. " +
					"Set to false and apply before running terraform destroy.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
		},
		Blocks: map[string]schema.Block{
			"source_index": schema.ListNestedBlock{
				Description: "Algolia indices to use as sources for query suggestions.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"index_name": schema.StringAttribute{
							Description: "Name of the source Algolia index (case-sensitive).",
							Required:    true,
						},
						"replicas": schema.BoolAttribute{
							Description: "If true, Query Suggestions also uses replica indices to find popular searches.",
							Optional:    true,
							Computed:    true,
						},
						"analytics_tags": schema.ListAttribute{
							Description: "Analytics tags to filter the popular searches used for suggestions.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"min_hits": schema.Int64Attribute{
							Description: "Minimum number of search results a query must generate to be included as a suggestion.",
							Optional:    true,
							Computed:    true,
						},
						"min_letters": schema.Int64Attribute{
							Description: "Minimum number of letters a query must have to be included as a suggestion.",
							Optional:    true,
							Computed:    true,
						},
						"generate": schema.StringAttribute{
							Description: "JSON-encoded list of lists of facet attributes used to generate additional suggestions. " +
								"Use jsonencode([[\"brand\"], [\"category\", \"brand\"]]) in HCL.",
							Optional: true,
						},
						"external": schema.ListAttribute{
							Description: "Names of external indices whose popular searches are included as suggestions.",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
					Blocks: map[string]schema.Block{
						"facets": schema.ListNestedBlock{
							Description: "Facets to use as categories for query suggestions.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"attribute": schema.StringAttribute{
										Description: "Facet attribute name.",
										Optional:    true,
									},
									"amount": schema.Int64Attribute{
										Description: "Number of suggestions to generate for this facet.",
										Optional:    true,
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
