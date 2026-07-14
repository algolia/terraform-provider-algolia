package index

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// IndicesDataSourceModel describes the algolia_indices data source: a
// listing of every index in the Algolia application, via ListIndices.
type IndicesDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	Indices types.List   `tfsdk:"indices"`
}

// IndexListItemModel describes a single entry within the algolia_indices
// "indices" list; it mirrors the Algolia FetchedIndex API response.
type IndexListItemModel struct {
	Name                 types.String `tfsdk:"name"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
	Entries              types.Int64  `tfsdk:"entries"`
	DataSize             types.Int64  `tfsdk:"data_size"`
	FileSize             types.Int64  `tfsdk:"file_size"`
	LastBuildTimeS       types.Int64  `tfsdk:"last_build_time_s"`
	NumberOfPendingTasks types.Int64  `tfsdk:"number_of_pending_tasks"`
	PendingTask          types.Bool   `tfsdk:"pending_task"`
	Primary              types.String `tfsdk:"primary"`
	Replicas             types.List   `tfsdk:"replicas"`
	Virtual              types.Bool   `tfsdk:"virtual"`
}

// indexListItemAttrTypes mirrors the "indices" nested object schema exactly;
// used to convert []IndexListItemModel to types.List.
var indexListItemAttrTypes = map[string]attr.Type{
	"name":                    types.StringType,
	"created_at":              types.StringType,
	"updated_at":              types.StringType,
	"entries":                 types.Int64Type,
	"data_size":               types.Int64Type,
	"file_size":               types.Int64Type,
	"last_build_time_s":       types.Int64Type,
	"number_of_pending_tasks": types.Int64Type,
	"pending_task":            types.BoolType,
	"primary":                 types.StringType,
	"replicas":                types.ListType{ElemType: types.StringType},
	"virtual":                 types.BoolType,
}
