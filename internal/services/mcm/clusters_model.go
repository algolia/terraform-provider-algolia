package mcm

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ClustersDataSourceModel describes the algolia_clusters data source: a
// listing of every cluster in a multi-cluster (MCM) Algolia application, via
// ListClusters.
type ClustersDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Clusters types.List   `tfsdk:"clusters"`
}

// ClusterListItemModel describes a single entry within the algolia_clusters
// "clusters" list.
//
// NOTE: ListClusters is a deprecated MCM endpoint, and its documented
// response (search.ListClustersResponse) only carries cluster *names*
// (the "topUsers" field, despite its confusing name, is a []string of
// cluster names) - it does not carry per-cluster nb_records/nb_user_ids/
// data_size the way search.UserId does for individual users. So this model
// only has cluster_name; there's no richer per-cluster data to surface.
type ClusterListItemModel struct {
	ClusterName types.String `tfsdk:"cluster_name"`
}

// clusterListItemAttrTypes mirrors the "clusters" nested object schema
// exactly; used to convert []ClusterListItemModel to types.List.
var clusterListItemAttrTypes = map[string]attr.Type{
	"cluster_name": types.StringType,
}
