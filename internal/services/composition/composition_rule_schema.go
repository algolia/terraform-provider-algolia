package composition

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func compositionRuleResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages an Algolia composition rule: a condition/consequence pair that overrides a " +
			"composition's behavior when its conditions are met (for example, changing which sources are " +
			"blended, or how they are sorted, for a specific query pattern or facet-based context). Composition " +
			"rules are similar to `algolia_rule` (Search Rules) and `algolia_recommend_rule`, except their " +
			"consequence is a `behavior` override with the same shape as `algolia_composition`'s own `behavior` " +
			"attribute.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform identifier in the form <composition_id>/<object_id>.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"composition_id": schema.StringAttribute{
				Description: "The composition the rule applies to. Changing this forces replacement.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"object_id": schema.StringAttribute{
				Description: "Unique identifier of the composition rule. Generated automatically if omitted. " +
					"Changing this forces replacement.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"conditions": schema.StringAttribute{
				Description: "JSON-encoded array of conditions that trigger the rule (`pattern`/`anchoring`/" +
					"`context`/`filters`/`sortBy`), e.g. `jsonencode([{ filters = \"brand:apple\" }])`. If " +
					"omitted, the rule always applies. Refreshed on read, but the configured value is only " +
					"replaced when it is not semantically equivalent (ignoring key/array order) to what the API " +
					"returned, to avoid a perpetual diff from harmless reordering.",
				Optional: true,
			},
			"consequence": schema.StringAttribute{
				Description: "JSON-encoded effect of the rule: an object with a `behavior` key, using the same " +
					"shape as `algolia_composition`'s own `behavior` attribute, e.g. `jsonencode({ behavior = { " +
					"injection = { main = { source = { search = { index = \"products_featured\" } } } } } })`. " +
					"Refreshed on read using the same semantic-equality rule as `conditions`.",
				Required: true,
			},
			"description": schema.StringAttribute{
				Description: "Human-readable description of the rule's purpose. This can be helpful for display " +
					"in the Algolia dashboard.",
				Optional: true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the rule is active. If it isn't enabled, it isn't applied when running " +
					"the composition.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"validity": schema.StringAttribute{
				Description: "JSON-encoded array of time periods (`from`/`until`, Unix timestamps in seconds) " +
					"during which the rule is active, e.g. `jsonencode([{ from = 1893456000, until = 1893542400 " +
					"}])`. If omitted, the rule is always active. Refreshed on read using the same " +
					"semantic-equality rule as `conditions`.",
				Optional: true,
			},
			"tags": schema.ListAttribute{
				Description: "Tags used to categorize the rule, for example for display in the Algolia " +
					"dashboard.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}
