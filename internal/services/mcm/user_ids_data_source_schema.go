package mcm

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func userIdsDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to list every user ID mapped to a cluster in a multi-cluster (MCM) " +
			"Algolia application. Only available on applications with multi-cluster management enabled; calling " +
			"this on a single-cluster application returns an error.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Description: "Terraform identifier for this listing. Set to the Algolia application ID.",
				Computed:    true,
			},
			"user_ids": datasourceschema.ListNestedAttribute{
				Description: "Every user ID mapped to a cluster in the application.",
				Computed:    true,
				NestedObject: datasourceschema.NestedAttributeObject{
					Attributes: map[string]datasourceschema.Attribute{
						"user_id": datasourceschema.StringAttribute{
							Description: "Unique identifier of the user.",
							Computed:    true,
						},
						"cluster_name": datasourceschema.StringAttribute{
							Description: "Cluster to which the user is assigned.",
							Computed:    true,
						},
						"nb_records": datasourceschema.Int64Attribute{
							Description: "Number of records belonging to the user.",
							Computed:    true,
						},
						"data_size": datasourceschema.Int64Attribute{
							Description: "Data size used by the user.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}
