package recommend

import (
	"strings"

	recommendapi "github.com/algolia/algoliasearch-client-go/v4/algolia/recommend"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func recommendRuleResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages an Algolia Recommend rule: a condition/consequence pair that customizes a " +
			"Recommend model's results for a specific index (for example, hiding or promoting specific items, " +
			"or filtering recommendations based on the source item). Recommend rules are similar to `algolia_rule` " +
			"(Search Rules), except their conditions/consequences apply to a source item rather than a search " +
			"query. Unlike most other Algolia APIs the provider talks to, Recommend is not region-routed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform identifier in the form <index_name>/<model>/<object_id>.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"index_name": schema.StringAttribute{
				Description: "The index the Recommend rule applies to. Changing this forces replacement.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"model": schema.StringAttribute{
				Description: "Recommend model the rule applies to. One of: " +
					strings.Join(allowedRecommendModelStrings(), ", ") + ". Changing this forces replacement.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedRecommendModelStrings()...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"object_id": schema.StringAttribute{
				Description: "Unique identifier of the Recommend rule. Generated automatically if omitted. " +
					"Changing this forces replacement.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"condition": schema.StringAttribute{
				Description: "JSON-encoded condition that triggers the rule (`context`/`filters`), e.g. " +
					"`jsonencode({ filters = \"brand:apple\" })`. If omitted, the rule applies to every " +
					"recommendation computed for this model/index. Refreshed on read, but the configured value " +
					"is only replaced when it is not semantically equivalent (ignoring key/array order) to what " +
					"the API returned, to avoid a perpetual diff from harmless reordering.",
				Optional: true,
			},
			"consequence": schema.StringAttribute{
				Description: "JSON-encoded effect of the rule (`hide`/`promote`/`params`), e.g. " +
					"`jsonencode({ hide = [{ objectID = \"42\" }] })`. Refreshed on read using the same " +
					"semantic-equality rule as `condition`.",
				Required: true,
			},
			"description": schema.StringAttribute{
				Description: "Human-readable description of the rule's purpose. This can be helpful for display " +
					"in the Algolia dashboard.",
				Optional: true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the rule is active. If it isn't enabled, it isn't applied when computing " +
					"recommendations.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"validity": schema.StringAttribute{
				Description: "JSON-encoded array of time periods (`from`/`until`, Unix timestamps in seconds) " +
					"during which the rule is active, e.g. `jsonencode([{ from = 1893456000, until = 1893542400 " +
					"}])`. If omitted, the rule is always active. Refreshed on read using the same " +
					"semantic-equality rule as `condition`.",
				Optional: true,
			},
		},
	}
}

// allowedRecommendModelStrings derives the list of valid `model` values from
// the Go client's RecommendModels enum rather than hard-coding it, so a new
// Recommend model added upstream doesn't require a provider code change to
// become selectable (only a client bump).
func allowedRecommendModelStrings() []string {
	values := make([]string, 0, len(recommendapi.AllowedRecommendModelsEnumValues))
	for _, v := range recommendapi.AllowedRecommendModelsEnumValues {
		values = append(values, string(v))
	}

	return values
}
