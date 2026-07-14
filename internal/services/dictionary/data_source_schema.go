package dictionary

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func dictionaryEntryDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read a single custom Algolia dictionary entry (stopwords, plurals, or compounds).",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Description: "Terraform identifier in the form <dictionary>/<object_id>.",
				Computed:    true,
			},
			"dictionary": datasourceschema.StringAttribute{
				Description: "Dictionary the entry belongs to. One of \"stopwords\", \"plurals\", or \"compounds\".",
				Required:    true,
			},
			"object_id": datasourceschema.StringAttribute{
				Description: "Unique identifier of the dictionary entry.",
				Required:    true,
			},
			"language": datasourceschema.StringAttribute{
				Description: "ISO code of the language the entry applies to.",
				Computed:    true,
			},
			"word": datasourceschema.StringAttribute{
				Description: "Matching dictionary word. Set for \"stopwords\" and \"compounds\" entries.",
				Computed:    true,
			},
			"words": datasourceschema.ListAttribute{
				Description: "Matching word forms, including declensions. Set for \"plurals\" entries.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"decomposition": datasourceschema.ListAttribute{
				Description: "Individual components of a compound word. Set for \"compounds\" entries.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"state": datasourceschema.StringAttribute{
				Description: "Whether the entry is active (\"enabled\" or \"disabled\").",
				Computed:    true,
			},
		},
	}
}
