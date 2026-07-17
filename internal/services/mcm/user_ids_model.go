package mcm

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// UserIdsDataSourceModel describes the algolia_user_ids data source: a
// listing of every user ID mapped to a cluster in a multi-cluster (MCM)
// Algolia application, via ListUserIds.
type UserIdsDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	UserIds types.List   `tfsdk:"user_ids"`
}

// UserIdListItemModel describes a single entry within the algolia_user_ids
// "user_ids" list; it mirrors the Algolia UserId API response.
type UserIdListItemModel struct {
	UserID      types.String `tfsdk:"user_id"`
	ClusterName types.String `tfsdk:"cluster_name"`
	NbRecords   types.Int64  `tfsdk:"nb_records"`
	DataSize    types.Int64  `tfsdk:"data_size"`
}

// userIdListItemAttrTypes mirrors the "user_ids" nested object schema
// exactly; used to convert []UserIdListItemModel to types.List.
var userIdListItemAttrTypes = map[string]attr.Type{
	"user_id":      types.StringType,
	"cluster_name": types.StringType,
	"nb_records":   types.Int64Type,
	"data_size":    types.Int64Type,
}
