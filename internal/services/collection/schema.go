package collection

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func collectionResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages an Algolia Collection: a curated set of index records defined manually (records) and/or by rules (conditions).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier (UUID) of the collection.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Display name of the collection (1-100 characters).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 100),
				},
			},
			"index_name": schema.StringAttribute{
				Description: "Name of the index the collection belongs to. Collections are index-scoped — changing this forces replacement.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Free-form description of the collection. Once set, the Algolia API does not support clearing this field on update — only changing it. To remove a description entirely, destroy and recreate the resource.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					PreserveOnUnsetString("The Algolia Collections API does not accept null for `description` on update."),
				},
			},
			"records": schema.ListAttribute{
				Description: "Desired set of objectIDs belonging to the collection. The provider computes add/remove deltas against prior state.",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.LengthAtMost(200)),
				},
			},
			"commit": schema.BoolAttribute{
				Description: "Whether upsert and delete requests auto-commit their changes to the underlying index. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"deletion_protection": schema.BoolAttribute{
				Description: "When true, prevents accidental deletion. Must be set to false before destroying.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"status": schema.StringAttribute{
				Description: "Commit status reported by the API: COMMITTED, COMMITTING, or TO_COMMIT. Only populated when the API key has write ACLs.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "RFC 3339 timestamp of when the collection was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "RFC 3339 timestamp of when the collection was last updated.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"conditions": schema.SingleNestedBlock{
				Description: "Filter rules selecting records. Each `facet_filter`/`numeric_filter` block is AND-ed with its siblings; items inside `filters` are OR-ed with each other.",
				Blocks: map[string]schema.Block{
					"facet_filter":   filterGroupBlockSchema(facetFilterRegex, "<attribute>:<value>, e.g. brand:Apple"),
					"numeric_filter": filterGroupBlockSchema(numericFilterRegex, "<attribute><op><number> (op in <, <=, =, >=, >) or <attribute>:<n> TO <n>"),
				},
			},
		},
	}
}

func filterGroupBlockSchema(re *regexp.Regexp, example string) schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "A single AND clause: items in `filters` are OR-ed together. Repeat the block to AND more clauses.",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"filters": schema.ListAttribute{
					Description: fmt.Sprintf("Filter expressions to OR together. Each must match the grammar: %s.", example),
					Required:    true,
					ElementType: types.StringType,
					Validators: []validator.List{
						listvalidator.SizeAtLeast(1),
						listvalidator.ValueStringsAre(
							stringvalidator.RegexMatches(re, fmt.Sprintf("expected %s", example)),
						),
					},
				},
			},
		},
	}
}
