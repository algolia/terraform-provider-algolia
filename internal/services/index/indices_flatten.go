package index

import (
	"context"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// fetchAllIndices pages through ListIndices, aggregating every index in the
// application. The endpoint paginates its response (see
// ListIndicesResponse.NbPages), so a single call only returns one page; this
// loops with WithPage until every page has been retrieved.
func fetchAllIndices(client *search.APIClient) ([]search.FetchedIndex, error) {
	var all []search.FetchedIndex

	page := int32(0)
	for {
		resp, err := client.ListIndices(client.NewApiListIndicesRequest().WithPage(page))
		if err != nil {
			return nil, err
		}

		all = append(all, resp.GetItems()...)

		nbPages, ok := resp.GetNbPagesOk()
		if !ok || nbPages == nil || page+1 >= *nbPages {
			break
		}
		page++
	}

	return all, nil
}

// flattenIndicesDataSource converts the aggregated FetchedIndex list into
// the algolia_indices data source model.
func flattenIndicesDataSource(ctx context.Context, items []search.FetchedIndex, appID string, model *IndicesDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	entries := make([]IndexListItemModel, 0, len(items))
	for i := range items {
		item := items[i]

		entries = append(entries, IndexListItemModel{
			Name:                 types.StringValue(item.GetName()),
			CreatedAt:            types.StringValue(item.GetCreatedAt()),
			UpdatedAt:            types.StringValue(item.GetUpdatedAt()),
			Entries:              types.Int64Value(int64(item.GetEntries())),
			DataSize:             types.Int64Value(item.GetDataSize()),
			FileSize:             types.Int64Value(item.GetFileSize()),
			LastBuildTimeS:       types.Int64Value(int64(item.GetLastBuildTimeS())),
			NumberOfPendingTasks: types.Int64Value(int64(item.GetNumberOfPendingTasks())),
			PendingTask:          types.BoolValue(item.GetPendingTask()),
			Primary:              flattenNullableString(item.Primary),
			Replicas:             flattenStringList(ctx, item.GetReplicas()),
			Virtual:              flattenNullableBool(item.Virtual),
		})
	}

	indicesValue, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: indexListItemAttrTypes}, entries)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	model.ID = types.StringValue(appID)
	model.Indices = indicesValue

	return diags
}
