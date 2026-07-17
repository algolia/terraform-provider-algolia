package mcm

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func clustersDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to list every cluster in a multi-cluster (MCM) Algolia application. " +
			"Only available on applications with multi-cluster management enabled; calling this on a single-cluster " +
			"application returns an error.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Description: "Terraform identifier for this listing. Set to the Algolia application ID.",
				Computed:    true,
			},
			"clusters": datasourceschema.ListNestedAttribute{
				Description: "Every cluster in the application. The underlying ListClusters API is deprecated and " +
					"only returns cluster names - no per-cluster record counts, user counts, or data size.",
				Computed: true,
				NestedObject: datasourceschema.NestedAttributeObject{
					Attributes: map[string]datasourceschema.Attribute{
						"cluster_name": datasourceschema.StringAttribute{
							Description: "Cluster name.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}
