package mcm

import (
	"context"
	"fmt"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenUserIdsDataSource_Basic(t *testing.T) {
	items := []search.UserId{
		*search.NewUserId("user1", "c1-test", 100, 2048),
		*search.NewUserId("user2", "c2-test", 50, 1024),
	}

	var model UserIdsDataSourceModel
	diags := flattenUserIdsDataSource(context.Background(), items, "APPID123", &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.ID.ValueString(); got != "APPID123" {
		t.Fatalf("id = %q, want %q", got, "APPID123")
	}

	elements := model.UserIds.Elements()
	if len(elements) != 2 {
		t.Fatalf("user_ids = %#v, want 2 entries", elements)
	}

	first := elements[0].(types.Object).Attributes()
	if v, ok := first["user_id"].(types.String); !ok || v.ValueString() != "user1" {
		t.Fatalf("user_id = %#v, want %q", first["user_id"], "user1")
	}
	if v, ok := first["cluster_name"].(types.String); !ok || v.ValueString() != "c1-test" {
		t.Fatalf("cluster_name = %#v, want %q", first["cluster_name"], "c1-test")
	}
	if v, ok := first["nb_records"].(types.Int64); !ok || v.ValueInt64() != 100 {
		t.Fatalf("nb_records = %#v, want 100", first["nb_records"])
	}
	if v, ok := first["data_size"].(types.Int64); !ok || v.ValueInt64() != 2048 {
		t.Fatalf("data_size = %#v, want 2048", first["data_size"])
	}
}

func TestFlattenUserIdsDataSource_Empty(t *testing.T) {
	var model UserIdsDataSourceModel
	diags := flattenUserIdsDataSource(context.Background(), nil, "APPID123", &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.UserIds.IsNull() {
		t.Fatal("expected a non-null user_ids list")
	}
	if len(model.UserIds.Elements()) != 0 {
		t.Fatalf("user_ids = %#v, want empty", model.UserIds.Elements())
	}
}

// TestCollectAllUserIds_MultiplePages verifies the pagination loop
// accumulates items across pages and stops once a page comes back short of
// hitsPerPage - the only signal available, since ListUserIdsResponse
// carries no nbPages/nbUsers metadata to check against.
func TestCollectAllUserIds_MultiplePages(t *testing.T) {
	const hitsPerPage = int32(2)

	pages := map[int32][]search.UserId{
		0: {*search.NewUserId("user1", "c1-test", 1, 1), *search.NewUserId("user2", "c1-test", 1, 1)},
		1: {*search.NewUserId("user3", "c2-test", 1, 1), *search.NewUserId("user4", "c2-test", 1, 1)},
		2: {*search.NewUserId("user5", "c2-test", 1, 1)},
	}

	var calledPages []int32
	fetch := func(page int32) ([]search.UserId, error) {
		calledPages = append(calledPages, page)
		items, ok := pages[page]
		if !ok {
			return nil, fmt.Errorf("unexpected page requested: %d", page)
		}
		return items, nil
	}

	all, err := collectAllUserIds(fetch, hitsPerPage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(all) != 5 {
		t.Fatalf("got %d user IDs, want 5", len(all))
	}
	wantOrder := []string{"user1", "user2", "user3", "user4", "user5"}
	for i, want := range wantOrder {
		if got := all[i].GetUserID(); got != want {
			t.Fatalf("all[%d].UserID = %q, want %q", i, got, want)
		}
	}

	wantPages := []int32{0, 1, 2}
	if len(calledPages) != len(wantPages) {
		t.Fatalf("calledPages = %v, want %v", calledPages, wantPages)
	}
	for i, want := range wantPages {
		if calledPages[i] != want {
			t.Fatalf("calledPages[%d] = %d, want %d", i, calledPages[i], want)
		}
	}
}

// TestCollectAllUserIds_SinglePage verifies that a single short page (fewer
// items than hitsPerPage) stops pagination immediately without a second
// request.
func TestCollectAllUserIds_SinglePage(t *testing.T) {
	const hitsPerPage = int32(1000)

	calls := 0
	fetch := func(page int32) ([]search.UserId, error) {
		calls++
		if page != 0 {
			t.Fatalf("unexpected page requested: %d", page)
		}
		return []search.UserId{*search.NewUserId("user1", "c1-test", 1, 1)}, nil
	}

	all, err := collectAllUserIds(fetch, hitsPerPage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("got %d user IDs, want 1", len(all))
	}
	if calls != 1 {
		t.Fatalf("fetch called %d times, want 1", calls)
	}
}

// TestCollectAllUserIds_Empty verifies an empty first page yields an empty
// (non-error) result with exactly one request.
func TestCollectAllUserIds_Empty(t *testing.T) {
	const hitsPerPage = int32(1000)

	calls := 0
	fetch := func(page int32) ([]search.UserId, error) {
		calls++
		return nil, nil
	}

	all, err := collectAllUserIds(fetch, hitsPerPage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("got %d user IDs, want 0", len(all))
	}
	if calls != 1 {
		t.Fatalf("fetch called %d times, want 1", calls)
	}
}

// TestCollectAllUserIds_PropagatesError verifies a fetch error aborts
// pagination and surfaces the error.
func TestCollectAllUserIds_PropagatesError(t *testing.T) {
	wantErr := fmt.Errorf("boom")
	fetch := func(page int32) ([]search.UserId, error) {
		return nil, wantErr
	}

	_, err := collectAllUserIds(fetch, 1000)
	if err != wantErr {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}
