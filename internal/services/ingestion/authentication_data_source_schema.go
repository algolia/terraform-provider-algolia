package ingestion

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func authenticationDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read an Algolia Ingestion authentication resource's metadata. " +
			"The `input` credentials are redacted by the API and are therefore not exposed by this data source " +
			"at all; use the algolia_ingestion_authentication resource to manage credentials.",
		Attributes: map[string]datasourceschema.Attribute{
			"authentication_id": datasourceschema.StringAttribute{
				Description: "Universally unique identifier (UUID) of the authentication resource to read.",
				Required:    true,
			},
			"id": datasourceschema.StringAttribute{
				Description: "Terraform identifier for the resource. Equal to `authentication_id`.",
				Computed:    true,
			},
			"type": datasourceschema.StringAttribute{
				Description: "Type of authentication.",
				Computed:    true,
			},
			"name": datasourceschema.StringAttribute{
				Description: "Descriptive name for the authentication resource.",
				Computed:    true,
			},
			"platform": datasourceschema.StringAttribute{
				Description: "Name of the ecommerce platform this authentication is scoped to, if any.",
				Computed:    true,
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
