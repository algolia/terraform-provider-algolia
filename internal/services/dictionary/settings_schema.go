package dictionary

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func dictionarySettingsResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages the Algolia app-level dictionary settings: which of Algolia's built-in standard " +
			"dictionary entries (stopwords, plurals, compounds) are disabled, per language. This is a singleton " +
			"resource — there is exactly one dictionary settings configuration per Algolia application.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform identifier for the singleton dictionary settings resource. Set to the Algolia application ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"disable_standard_entries": schema.SingleNestedAttribute{
				Description: "Standard dictionary entries to disable, per dictionary type and language. " +
					"Set a language ISO code to `true` to disable Algolia's built-in entries for it, or `false` to keep them enabled. " +
					"Applied values are preserved across plans (Computed): to re-enable a previously disabled language, set it to `false` " +
					"or supply an updated map — removing the attribute from configuration does not clear settings that were already applied.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"stopwords": schema.MapAttribute{
						Description: "Language ISO codes mapped to whether Algolia's built-in stopwords for that language are disabled.",
						Optional:    true,
						Computed:    true,
						ElementType: types.BoolType,
						PlanModifiers: []planmodifier.Map{
							mapplanmodifier.UseStateForUnknown(),
						},
					},
					"plurals": schema.MapAttribute{
						Description: "Language ISO codes mapped to whether Algolia's built-in plurals for that language are disabled.",
						Optional:    true,
						Computed:    true,
						ElementType: types.BoolType,
						PlanModifiers: []planmodifier.Map{
							mapplanmodifier.UseStateForUnknown(),
						},
					},
					"compounds": schema.MapAttribute{
						Description: "Language ISO codes mapped to whether Algolia's built-in compounds for that language are disabled.",
						Optional:    true,
						Computed:    true,
						ElementType: types.BoolType,
						PlanModifiers: []planmodifier.Map{
							mapplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
		},
	}
}
