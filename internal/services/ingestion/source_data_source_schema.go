package ingestion

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func sourceDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read an Algolia Ingestion source's configuration, including its " +
			"`input`: unlike the algolia_ingestion_authentication data source, the Ingestion API does not " +
			"redact a source's configuration.",
		Attributes: map[string]datasourceschema.Attribute{
			"source_id": datasourceschema.StringAttribute{
				Description: "Universally unique identifier (UUID) of the source to read.",
				Required:    true,
			},
			"id": datasourceschema.StringAttribute{
				Description: "Terraform identifier for the resource. Equal to `source_id`.",
				Computed:    true,
			},
			"type": datasourceschema.StringAttribute{
				Description: "Type of source.",
				Computed:    true,
			},
			"name": datasourceschema.StringAttribute{
				Description: "Descriptive name for the source.",
				Computed:    true,
			},
			"input": datasourceschema.StringAttribute{
				Description: "JSON-encoded configuration matching `type`. Null if the source type requires no " +
					"configuration (e.g. \"push\"). Treated as sensitive: the API returns `input` unredacted, and " +
					"several source types carry credentials in it (a `docker` source's `configuration`, or a " +
					"presigned `url` for `csv`/`json`).",
				Computed:  true,
				Sensitive: true,
			},
			"authentication_id": datasourceschema.StringAttribute{
				Description: "Universally unique identifier (UUID) of the authentication resource this source " +
					"uses to connect to its underlying platform, if any.",
				Computed: true,
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
