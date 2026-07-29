package index

import (
	"context"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// fetchAllIndices pages through ListIndices, aggregating every index in the
// application. The endpoint paginates its response (see
// ListIndicesResponse.NbPages), so a single call only returns one page; this
// loops with WithPage until every page has been retrieved.
func fetchAllIndices(ctx context.Context, client *search.APIClient) ([]search.FetchedIndex, error) {
	var all []search.FetchedIndex

	page := int32(0)
	for {
		resp, err := client.ListIndices(client.NewApiListIndicesRequest().WithPage(page), search.WithContext(ctx))
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

// applyIndexMetadata fills the metadata attributes of model - entries,
// data_size, created_at and updated_at - from the application's index list.
// Those values are not part of a GetSettings response, so they need this second
// call.
//
// The metadata is best effort and never produces an error: the caller has
// already read the settings successfully, and failing the whole read over
// missing metadata would be worse than reporting zeroes. A listing failure is
// logged and a freshly created index that is not visible in the listing yet is
// treated the same way. Both write zero values rather than leaving nulls, so
// the attributes stay known.
func applyIndexMetadata(ctx context.Context, client *search.APIClient, model *IndexResourceModel) {
	items, err := fetchAllIndices(ctx, client)
	if err != nil {
		tflog.Warn(ctx, "Could not list indices for metadata", map[string]any{"error": err.Error()})
		clearIndexMetadata(model)
		return
	}

	indexName := model.Name.ValueString()
	for i := range items {
		if items[i].Name != indexName {
			continue
		}

		model.Entries = types.Int64Value(int64(items[i].Entries))
		model.DataSize = types.Int64Value(items[i].DataSize)
		model.CreatedAt = types.StringValue(items[i].CreatedAt)
		model.UpdatedAt = types.StringValue(items[i].UpdatedAt)
		return
	}

	clearIndexMetadata(model)
}

func clearIndexMetadata(model *IndexResourceModel) {
	model.Entries = types.Int64Value(0)
	model.DataSize = types.Int64Value(0)
	model.CreatedAt = types.StringValue("")
	model.UpdatedAt = types.StringValue("")
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
