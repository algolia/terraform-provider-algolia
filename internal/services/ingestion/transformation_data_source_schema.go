package ingestion

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func transformationDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read an Algolia Ingestion transformation's configuration, " +
			"including its `code` and `input`: like the algolia_ingestion_source and algolia_ingestion_destination " +
			"data sources, the Ingestion API does not redact a transformation's configuration.",
		Attributes: map[string]datasourceschema.Attribute{
			"transformation_id": datasourceschema.StringAttribute{
				Description: "Universally unique identifier (UUID) of the transformation to read.",
				Required:    true,
			},
			"id": datasourceschema.StringAttribute{
				Description: "Terraform identifier for the resource. Equal to `transformation_id`.",
				Computed:    true,
			},
			"name": datasourceschema.StringAttribute{
				Description: "Descriptive, uniquely identified name for the transformation.",
				Computed:    true,
			},
			"code": datasourceschema.StringAttribute{
				Description: "The transformation's source code (for `type = \"code\"` transformations). Null for " +
					"no-code transformations (which have no `code`).",
				Computed: true,
			},
			"type": datasourceschema.StringAttribute{
				Description: "Type of transformation.",
				Computed:    true,
			},
			"input": datasourceschema.StringAttribute{
				Description: "JSON-encoded configuration matching `type`.",
				Computed:    true,
			},
			"description": datasourceschema.StringAttribute{
				Description: "A descriptive name for the transformation explaining what it does.",
				Computed:    true,
			},
			"authentication_ids": datasourceschema.ListAttribute{
				Description: "Universally unique identifiers (UUIDs) of the authentication resources associated " +
					"with this transformation.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"created_at": datasourceschema.StringAttribute{
				Description: "Date and time when the resource was created, in RFC 3339 format.",
				Computed:    true,
			},
			"updated_at": datasourceschema.StringAttribute{
				Description: "Date and time when the resource was last updated, in RFC 3339 format.",
				Computed:    true,
			},
		},
	}
}
