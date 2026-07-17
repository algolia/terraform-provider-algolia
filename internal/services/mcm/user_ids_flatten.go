package mcm

import (
	"context"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// userIdsHitsPerPage is the page size used when paging through ListUserIds.
// The endpoint's response (search.ListUserIdsResponse) carries no paging
// metadata (no nbPages/nbUsers, unlike ListIndicesResponse), so the only way
// to detect the last page is to notice a page came back with fewer than
// this many items.
const userIdsHitsPerPage = int32(1000)

// userIdsPageFetcher fetches a single page of ListUserIds results. It
// abstracts away the concrete *search.APIClient so the pagination logic in
// collectAllUserIds can be unit-tested with a fake, without a real HTTP
// client.
type userIdsPageFetcher func(page int32) ([]search.UserId, error)

// collectAllUserIds pages through fetchPage, aggregating every user ID,
// stopping once a page returns fewer than hitsPerPage items (or none at
// all). This mirrors fetchAllIndices' paging loop in
// internal/services/index/indices_flatten.go, adapted to an endpoint whose
// response has no nbPages field to check against.
func collectAllUserIds(fetchPage userIdsPageFetcher, hitsPerPage int32) ([]search.UserId, error) {
	var all []search.UserId

	page := int32(0)
	for {
		items, err := fetchPage(page)
		if err != nil {
			return nil, err
		}

		all = append(all, items...)

		if int32(len(items)) < hitsPerPage {
			break
		}
		page++
	}

	return all, nil
}

// fetchAllUserIds pages through ListUserIds, aggregating every user ID
// mapping in the application.
func fetchAllUserIds(client *search.APIClient) ([]search.UserId, error) {
	return collectAllUserIds(func(page int32) ([]search.UserId, error) {
		resp, err := client.ListUserIds(client.NewApiListUserIdsRequest().WithPage(page).WithHitsPerPage(userIdsHitsPerPage))
		if err != nil {
			return nil, err
		}

		return resp.GetUserIDs(), nil
	}, userIdsHitsPerPage)
}

// flattenUserIdsDataSource converts the aggregated UserId list into the
// algolia_user_ids data source model.
func flattenUserIdsDataSource(ctx context.Context, items []search.UserId, appID string, model *UserIdsDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	entries := make([]UserIdListItemModel, 0, len(items))
	for i := range items {
		item := items[i]

		entries = append(entries, UserIdListItemModel{
			UserID:      types.StringValue(item.GetUserID()),
			ClusterName: types.StringValue(item.GetClusterName()),
			NbRecords:   types.Int64Value(int64(item.GetNbRecords())),
			DataSize:    types.Int64Value(int64(item.GetDataSize())),
		})
	}

	userIdsValue, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: userIdListItemAttrTypes}, entries)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	model.ID = types.StringValue(appID)
	model.UserIds = userIdsValue

	return diags
}
