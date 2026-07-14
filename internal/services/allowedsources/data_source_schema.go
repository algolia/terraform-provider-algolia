package allowedsources

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func allowedSourcesDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read the Algolia app-level allowed sources: the complete " +
			"allowlist of source IP addresses and CIDR ranges permitted to use the Algolia API for this " +
			"application.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Description: "Terraform identifier for the singleton allowed sources data source. Set to the Algolia application ID.",
				Computed:    true,
			},
			"source": datasourceschema.SetNestedAttribute{
				Description: "The complete set of allowed source IP addresses/ranges currently configured for this application.",
				Computed:    true,
				NestedObject: datasourceschema.NestedAttributeObject{
					Attributes: map[string]datasourceschema.Attribute{
						"source": datasourceschema.StringAttribute{
							Description: "The allowed IP address or CIDR range.",
							Computed:    true,
						},
						"description": datasourceschema.StringAttribute{
							Description: "Human-readable description of this source.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}
