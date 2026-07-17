package mcm

import (
	"context"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// flattenClustersDataSource converts a ListClustersResponse into the
// algolia_clusters data source model.
func flattenClustersDataSource(ctx context.Context, resp *search.ListClustersResponse, appID string, model *ClustersDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	names := resp.GetTopUsers()
	items := make([]ClusterListItemModel, 0, len(names))
	for _, name := range names {
		items = append(items, ClusterListItemModel{
			ClusterName: types.StringValue(name),
		})
	}

	clustersValue, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: clusterListItemAttrTypes}, items)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	model.ID = types.StringValue(appID)
	model.Clusters = clustersValue

	return diags
}
