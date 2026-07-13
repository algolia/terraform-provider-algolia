package dictionary

import (
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func dictionaryEntryResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages a single custom entry in an Algolia dictionary (stopwords, plurals, or compounds). " +
			"Dictionaries are application-level: entries are not scoped to a specific index.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform identifier in the form <dictionary>/<object_id>.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dictionary": schema.StringAttribute{
				Description: "Dictionary the entry belongs to. One of \"stopwords\", \"plurals\", or \"compounds\".",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("stopwords", "plurals", "compounds"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"object_id": schema.StringAttribute{
				Description: "Unique identifier of the dictionary entry. Generated automatically if omitted.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"language": schema.StringAttribute{
				Description: "ISO code of the language the entry applies to, for example \"en\" or \"fr\".",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedLanguageStrings()...),
				},
			},
			"word": schema.StringAttribute{
				Description: "Matching dictionary word. Required for \"stopwords\" and \"compounds\" entries.",
				Optional:    true,
			},
			"words": schema.ListAttribute{
				Description: "Matching word forms, including declensions. Required for \"plurals\" entries.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"decomposition": schema.ListAttribute{
				Description: "Individual components of a compound word. Required for \"compounds\" entries.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"state": schema.StringAttribute{
				Description: "Whether the entry is active. One of \"enabled\" or \"disabled\". " +
					"Applies to \"stopwords\" entries; defaults to \"enabled\".",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf("enabled", "disabled"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func allowedLanguageStrings() []string {
	values := make([]string, 0, len(search.AllowedSupportedLanguageEnumValues))
	for _, value := range search.AllowedSupportedLanguageEnumValues {
		values = append(values, string(value))
	}

	return values
}
