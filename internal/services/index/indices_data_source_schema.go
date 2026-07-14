package index

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func indicesDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to list every index in the Algolia application.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Description: "Terraform identifier for this listing. Set to the Algolia application ID.",
				Computed:    true,
			},
			"indices": datasourceschema.ListNestedAttribute{
				Description: "Every index in the application.",
				Computed:    true,
				NestedObject: datasourceschema.NestedAttributeObject{
					Attributes: map[string]datasourceschema.Attribute{
						"name": datasourceschema.StringAttribute{
							Description: "Index name.",
							Computed:    true,
						},
						"created_at": datasourceschema.StringAttribute{
							Description: "Index creation date. An empty string means the index has no records.",
							Computed:    true,
						},
						"updated_at": datasourceschema.StringAttribute{
							Description: "Date and time when the index was last updated, in RFC 3339 format.",
							Computed:    true,
						},
						"entries": datasourceschema.Int64Attribute{
							Description: "Number of records contained in the index.",
							Computed:    true,
						},
						"data_size": datasourceschema.Int64Attribute{
							Description: "Number of bytes of the index in minified format.",
							Computed:    true,
						},
						"file_size": datasourceschema.Int64Attribute{
							Description: "Number of bytes of the index binary file.",
							Computed:    true,
						},
						"last_build_time_s": datasourceschema.Int64Attribute{
							Description: "Last build time, in seconds.",
							Computed:    true,
						},
						"number_of_pending_tasks": datasourceschema.Int64Attribute{
							Description: "Number of pending indexing operations. Deprecated by the API; should not be relied upon.",
							Computed:    true,
						},
						"pending_task": datasourceschema.BoolAttribute{
							Description: "Whether the index has pending tasks. Deprecated by the API; should not be relied upon.",
							Computed:    true,
						},
						"primary": datasourceschema.StringAttribute{
							Description: "Only present if the index is a replica. Name of the related primary index.",
							Computed:    true,
						},
						"replicas": datasourceschema.ListAttribute{
							Description: "Only present if the index is a primary index with replicas. Names of all linked replicas.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"virtual": datasourceschema.BoolAttribute{
							Description: "Only present if the index is a virtual replica.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}
