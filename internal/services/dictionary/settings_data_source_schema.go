package dictionary

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func dictionarySettingsDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read the Algolia app-level dictionary settings: which of Algolia's " +
			"built-in standard dictionary entries (stopwords, plurals, compounds) are disabled, per language.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Description: "Terraform identifier for the singleton dictionary settings resource. Set to the Algolia application ID.",
				Computed:    true,
			},
			"disable_standard_entries": datasourceschema.SingleNestedAttribute{
				Description: "Standard dictionary entries disabled, per dictionary type and language.",
				Computed:    true,
				Attributes: map[string]datasourceschema.Attribute{
					"stopwords": datasourceschema.MapAttribute{
						Description: "Language ISO codes mapped to whether Algolia's built-in stopwords for that language are disabled.",
						Computed:    true,
						ElementType: types.BoolType,
					},
					"plurals": datasourceschema.MapAttribute{
						Description: "Language ISO codes mapped to whether Algolia's built-in plurals for that language are disabled.",
						Computed:    true,
						ElementType: types.BoolType,
					},
					"compounds": datasourceschema.MapAttribute{
						Description: "Language ISO codes mapped to whether Algolia's built-in compounds for that language are disabled.",
						Computed:    true,
						ElementType: types.BoolType,
					},
				},
			},
		},
	}
}
