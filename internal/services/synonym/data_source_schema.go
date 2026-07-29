package synonym

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func synonymDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read a single Algolia synonym object in an index.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Description: "Terraform identifier in the form <index_name>/<object_id>.",
				Computed:    true,
			},
			"index_name": datasourceschema.StringAttribute{
				Description: "The index that owns the synonym.",
				Required:    true,
			},
			"object_id": datasourceschema.StringAttribute{
				Description: "Unique identifier of the synonym object.",
				Required:    true,
			},
			"type": datasourceschema.StringAttribute{
				Description: "Synonym type.",
				Computed:    true,
			},
			"synonyms": datasourceschema.SetAttribute{
				Description: "Synonyms used by regular and one-way synonyms.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"input": datasourceschema.StringAttribute{
				Description: "Base input used by one-way synonyms.",
				Computed:    true,
			},
			"word": datasourceschema.StringAttribute{
				Description: "Base word used by alternative correction synonyms.",
				Computed:    true,
			},
			"corrections": datasourceschema.SetAttribute{
				Description: "Corrections used by alternative correction synonyms.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"placeholder": datasourceschema.StringAttribute{
				Description: "Placeholder token used by placeholder synonyms.",
				Computed:    true,
			},
			"replacements": datasourceschema.SetAttribute{
				Description: "Replacement values used by placeholder synonyms.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}
