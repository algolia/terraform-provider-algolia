package ingestion

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func destinationDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read an Algolia Ingestion destination's configuration, including " +
			"its `input`: like the algolia_ingestion_source data source, the Ingestion API does not redact a " +
			"destination's configuration.",
		Attributes: map[string]datasourceschema.Attribute{
			"destination_id": datasourceschema.StringAttribute{
				Description: "Universally unique identifier (UUID) of the destination to read.",
				Required:    true,
			},
			"id": datasourceschema.StringAttribute{
				Description: "Terraform identifier for the resource. Equal to `destination_id`.",
				Computed:    true,
			},
			"type": datasourceschema.StringAttribute{
				Description: "Type of destination.",
				Computed:    true,
			},
			"name": datasourceschema.StringAttribute{
				Description: "Descriptive name for the destination.",
				Computed:    true,
			},
			"input": datasourceschema.StringAttribute{
				Description: "JSON-encoded configuration matching `type`.",
				Computed:    true,
			},
			"authentication_id": datasourceschema.StringAttribute{
				Description: "Universally unique identifier (UUID) of the authentication resource this " +
					"destination uses to connect to its underlying platform, if any.",
				Computed: true,
			},
			"transformation_ids": datasourceschema.ListAttribute{
				Description: "Universally unique identifiers (UUIDs) of the transformations applied to records " +
					"before they reach this destination, in order.",
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
