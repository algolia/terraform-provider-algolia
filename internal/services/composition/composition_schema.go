package composition

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func compositionResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages an Algolia composition: a saved configuration that blends results from multiple " +
			"sources - either an \"injection\" of recommend/search results into a main result set, or a " +
			"\"multifeed\" merge of several sources - behind a single compositionID that your frontend queries " +
			"like an index. Unlike most other Algolia APIs the provider talks to, Composition is not " +
			"region-routed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The composition's unique identifier. Same value as `object_id`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"object_id": schema.StringAttribute{
				Description: "Unique identifier of the composition (the compositionID referenced by search " +
					"clients). Changing this forces replacement.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Composition name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Composition description.",
				Optional:    true,
			},
			"behavior": schema.StringAttribute{
				Description: "JSON-encoded composition behavior: an object with either an `injection` or a " +
					"`multifeed` key (but not both), describing how the composition blends its sources, e.g. " +
					"`jsonencode({ injection = { main = { source = { search = { index = \"products\" } } } } " +
					"})`. Refreshed on read, but the configured value is only replaced when it is not " +
					"semantically equivalent (ignoring key/array order) to what the API returned, to avoid a " +
					"perpetual diff from harmless reordering.",
				Required: true,
			},
			"sorting_strategy": schema.StringAttribute{
				Description: "JSON-encoded map of up to 20 sorting labels to the index (or replica) that " +
					"implements that sorting rule, e.g. `jsonencode({ \"Price (asc)\" = \"products_price_asc\" " +
					"})`. Refreshed on read using the same semantic-equality rule as `behavior`.",
				Optional: true,
			},
		},
	}
}
