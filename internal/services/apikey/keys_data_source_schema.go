package apikey

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func apiKeysDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to list every API key configured for the Algolia application.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Description: "Terraform identifier for this listing. Set to the Algolia application ID.",
				Computed:    true,
			},
			"keys": datasourceschema.ListNestedAttribute{
				Description: "Every API key configured for the application.",
				Computed:    true,
				NestedObject: datasourceschema.NestedAttributeObject{
					Attributes: map[string]datasourceschema.Attribute{
						"value": datasourceschema.StringAttribute{
							Description: "The actual API key value.",
							Computed:    true,
							Sensitive:   true,
						},
						"acl": datasourceschema.ListAttribute{
							Description: "Permissions associated with the API key.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"description": datasourceschema.StringAttribute{
							Description: "Description of the API key.",
							Computed:    true,
						},
						"indexes": datasourceschema.ListAttribute{
							Description: "Index names or patterns the API key can access.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"max_hits_per_query": datasourceschema.Int64Attribute{
							Description: "Maximum number of results this API key can retrieve in one query.",
							Computed:    true,
						},
						"max_queries_per_ip_per_hour": datasourceschema.Int64Attribute{
							Description: "Maximum number of API requests allowed per IP address or user token per hour.",
							Computed:    true,
						},
						"query_parameters": datasourceschema.StringAttribute{
							Description: "Query parameters applied when making API requests with this API key.",
							Computed:    true,
						},
						"referers": datasourceschema.ListAttribute{
							Description: "Allowed HTTP referrers for this API key.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"validity": datasourceschema.Int64Attribute{
							Description: "Duration (in seconds) after which the API key expires. 0 means the key doesn't expire.",
							Computed:    true,
						},
						"created_at": datasourceschema.StringAttribute{
							Description: "RFC3339 timestamp of when the API key was created.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}
