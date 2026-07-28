package rule

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func ruleResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages an Algolia Rule for a specific index.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform identifier in the form <index_name>/<object_id>.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"index_name": schema.StringAttribute{
				Description: "The index that owns the rule. Changing this forces a new rule to be created.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"object_id": schema.StringAttribute{
				Description: "Unique identifier of the rule. Changing this forces a new rule to be created.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Human-readable description of the rule.",
				Optional:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the rule is active.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"tags": schema.ListAttribute{
				Description: "Free-form tags used to group and filter rules in the Algolia dashboard.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"scope": schema.StringAttribute{
				Description: "Rule scope. Algolia currently only accepts `redirect`, which turns the rule into a redirect rule and requires `consequence.redirect_index_name` to point at a virtual replica of the index.",
				Optional:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"conditions": schema.ListNestedBlock{
				Description: "Conditions that trigger the rule.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"pattern": schema.StringAttribute{
							Description: "Literal pattern or {facet:attribute} matcher.",
							Optional:    true,
						},
						"anchoring": schema.StringAttribute{
							Description: "How the pattern is matched.",
							Optional:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("contains", "is", "startsWith", "endsWith"),
							},
						},
						"alternatives": schema.BoolAttribute{
							Description: "Whether plurals, synonyms, and typos should match.",
							Optional:    true,
						},
						"context": schema.StringAttribute{
							Description: "Optional rule context restriction.",
							Optional:    true,
						},
						"filters": schema.StringAttribute{
							Description: "Optional facet filter expression.",
							Optional:    true,
						},
					},
				},
			},
			"consequence": schema.ListNestedBlock{
				Description: "Rule consequence definition.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"params_json": schema.StringAttribute{
							Description: "JSON-encoded consequence params object. The document is sent to Algolia verbatim, so search parameters that this provider release does not know about can still be set here.",
							Optional:    true,
						},
						"hide": schema.SetAttribute{
							Description: "Object IDs to hide.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"user_data": schema.StringAttribute{
							Description: "JSON-encoded userData payload appended to search responses.",
							Optional:    true,
						},
						"filter_promotes": schema.BoolAttribute{
							Description: "Whether promoted records must also match the active filters for the consequence to apply. Shown as \"Pinned items must match active filters to be displayed\" in the Algolia dashboard.",
							Optional:    true,
						},
						"redirect_index_name": schema.StringAttribute{
							Description: "Name of the virtual replica index searches are redirected to. Only valid together with `scope = \"redirect\"`.",
							Optional:    true,
						},
					},
					Blocks: map[string]schema.Block{
						"promote": schema.ListNestedBlock{
							Description: "Promoted object IDs and their position.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"object_ids": schema.SetAttribute{
										Description: "Object IDs to promote as a group.",
										Required:    true,
										ElementType: types.StringType,
									},
									"position": schema.Int64Attribute{
										Description: "Result position for the promoted group.",
										Required:    true,
									},
								},
							},
						},
					},
				},
			},
			"validity": schema.ListNestedBlock{
				Description: "Time windows during which the rule is active.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"from": schema.StringAttribute{
							Description: "RFC3339 timestamp when the rule becomes active.",
							Optional:    true,
						},
						"until": schema.StringAttribute{
							Description: "RFC3339 timestamp when the rule stops being active.",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}
