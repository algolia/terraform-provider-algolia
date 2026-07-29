package index

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/call"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/transport"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// These tests cover the index metadata attributes -- entries, data_size,
// created_at and updated_at. They are not part of a GetSettings response, so
// reading them needs a second call to ListIndices; the data source used to call
// GetSettings only, which left all four permanently null even for an index
// holding thousands of records.

// TestIndexDataSourceRead_populatesMetadata drives the data source end to end
// against a stubbed API. The index is deliberately returned on the *second*
// listing page, since a caller that reads only the first page cannot find it.
func TestIndexDataSourceRead_populatesMetadata(t *testing.T) {
	ctx := context.Background()
	client, _ := newIndexMetadataSearchClient(t, indexMetadataStub{
		settings: `{"primary":"tf-test-primary","searchableAttributes":["title"]}`,
		pages: []string{
			`{"items":[{"name":"tf-test-other","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","entries":1,"dataSize":1,"fileSize":1,"lastBuildTimeS":0,"numberOfPendingTasks":0,"pendingTask":false}],"nbPages":2}`,
			`{"items":[{"name":"tf-test-metadata","createdAt":"2026-02-03T04:05:06Z","updatedAt":"2026-02-04T05:06:07Z","entries":10000,"dataSize":8900000,"fileSize":9100000,"lastBuildTimeS":3,"numberOfPendingTasks":0,"pendingTask":false}],"nbPages":2}`,
		},
	})

	d := &indexDataSource{client: client}
	config, state := newIndexDataSourceConfigAndState(ctx, t, d, "tf-test-metadata")

	resp := &datasource.ReadResponse{State: state}
	d.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() diagnostics = %v, want none", resp.Diagnostics)
	}

	var got IndexDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading state back: %v", diags)
	}

	if got.Entries.ValueInt64() != 10000 {
		t.Errorf("entries = %v, want 10000", got.Entries)
	}
	if got.DataSize.ValueInt64() != 8900000 {
		t.Errorf("data_size = %v, want 8900000", got.DataSize)
	}
	if got.CreatedAt.ValueString() != "2026-02-03T04:05:06Z" {
		t.Errorf("created_at = %v, want 2026-02-03T04:05:06Z", got.CreatedAt)
	}
	if got.UpdatedAt.ValueString() != "2026-02-04T05:06:07Z" {
		t.Errorf("updated_at = %v, want 2026-02-04T05:06:07Z", got.UpdatedAt)
	}
	if got.Primary.ValueString() != "tf-test-primary" {
		t.Errorf("primary = %v, want tf-test-primary", got.Primary)
	}
	if got.Attributes.IsNull() {
		t.Error("attributes block is null; the settings read regressed")
	}
}

// TestIndexDataSourceRead_failsOnMissingIndex pins the data source's opposite
// behaviour to the resource's: a named index that does not exist is a
// configuration error, not drift to reconcile by dropping it from state.
func TestIndexDataSourceRead_failsOnMissingIndex(t *testing.T) {
	ctx := context.Background()
	d := &indexDataSource{client: newNotFoundSearchClient(t)}
	config, state := newIndexDataSourceConfigAndState(ctx, t, d, "tf-test-nonexistent")

	resp := &datasource.ReadResponse{State: state}
	d.Read(ctx, datasource.ReadRequest{Config: config}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Read() succeeded for an index that does not exist, want an error")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Error reading index" {
		t.Errorf("error summary = %q, want %q", got, "Error reading index")
	}
}

// TestIndexDataSourceSchema_omitsDeletionProtection guards the decision to keep
// deletion_protection out of the data source: it is a provider-side guard on
// destroy with no Algolia API representation, so a read-only data source has
// nothing to report for it and used to always return null.
func TestIndexDataSourceSchema_omitsDeletionProtection(t *testing.T) {
	if _, ok := indexDataSourceSchema().Attributes["deletion_protection"]; ok {
		t.Error("data source schema declares deletion_protection, which no read can populate")
	}
}

// TestApplyIndexMetadata_zeroesWhenIndexNotListed covers an index that exists
// -- GetSettings already succeeded -- but is not in the listing yet, which is
// what a just-created index looks like.
func TestApplyIndexMetadata_zeroesWhenIndexNotListed(t *testing.T) {
	client, _ := newIndexMetadataSearchClient(t, indexMetadataStub{
		pages: []string{`{"items":[{"name":"tf-test-other","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","entries":1,"dataSize":1,"fileSize":1,"lastBuildTimeS":0,"numberOfPendingTasks":0,"pendingTask":false}],"nbPages":1}`},
	})

	model := IndexResourceModel{Name: types.StringValue("tf-test-fresh")}
	applyIndexMetadata(context.Background(), client, &model)

	assertZeroedIndexMetadata(t, &model)
}

// TestApplyIndexMetadata_zeroesWhenListingFails covers the best-effort contract:
// a failing ListIndices must not fail the read that already got the settings.
func TestApplyIndexMetadata_zeroesWhenListingFails(t *testing.T) {
	model := IndexResourceModel{Name: types.StringValue("tf-test-metadata")}
	applyIndexMetadata(context.Background(), newNotFoundSearchClient(t), &model)

	assertZeroedIndexMetadata(t, &model)
}

// TestApplyIndexMetadata_honoursCancelledContext proves the request context is
// actually threaded into the paging loop: with an already-cancelled context no
// request may reach the API at all. Before the context was passed through, the
// loop would page on regardless of a cancelled plan.
func TestApplyIndexMetadata_honoursCancelledContext(t *testing.T) {
	client, requests := newIndexMetadataSearchClient(t, indexMetadataStub{
		pages: []string{`{"items":[],"nbPages":1}`},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	model := IndexResourceModel{Name: types.StringValue("tf-test-metadata")}
	applyIndexMetadata(ctx, client, &model)

	if got := requests.Load(); got != 0 {
		t.Errorf("%d request(s) reached the API under a cancelled context, want 0", got)
	}
	assertZeroedIndexMetadata(t, &model)
}

func assertZeroedIndexMetadata(t *testing.T, model *IndexResourceModel) {
	t.Helper()

	// Zeroes rather than nulls: every metadata attribute is Computed, so leaving
	// it null would surface as an unresolved value rather than "unknown yet".
	if model.Entries.IsNull() || model.Entries.ValueInt64() != 0 {
		t.Errorf("entries = %v, want 0", model.Entries)
	}
	if model.DataSize.IsNull() || model.DataSize.ValueInt64() != 0 {
		t.Errorf("data_size = %v, want 0", model.DataSize)
	}
	if model.CreatedAt.IsNull() || model.CreatedAt.ValueString() != "" {
		t.Errorf("created_at = %v, want an empty string", model.CreatedAt)
	}
	if model.UpdatedAt.IsNull() || model.UpdatedAt.ValueString() != "" {
		t.Errorf("updated_at = %v, want an empty string", model.UpdatedAt)
	}
}

// indexMetadataStub describes the responses of a stubbed Search API: one
// GetSettings body, and one ListIndices body per page in order.
type indexMetadataStub struct {
	settings string
	pages    []string
}

// newIndexMetadataSearchClient returns a Search client bound to a stub server,
// along with a counter of the requests that reached it.
func newIndexMetadataSearchClient(t *testing.T, stub indexMetadataStub) (*search.APIClient, *atomic.Int64) {
	t.Helper()

	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")

		if strings.HasSuffix(r.URL.Path, "/settings") {
			_, _ = w.Write([]byte(stub.settings))
			return
		}

		page := 0
		if raw := r.URL.Query().Get("page"); raw != "" && raw != "0" {
			page = int(raw[0] - '0')
		}
		if page >= len(stub.pages) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"unexpected page","status":500}`))
			return
		}
		_, _ = w.Write([]byte(stub.pages[page]))
	}))
	t.Cleanup(server.Close)

	client, err := search.NewClientWithConfig(search.SearchConfiguration{
		Configuration: transport.Configuration{
			AppID:  "test-app",
			ApiKey: "test-key",
			Hosts: []transport.StatefulHost{
				transport.NewStatefulHost("http", server.Listener.Addr().String(), func(call.Kind) bool { return true }),
			},
		},
	})
	if err != nil {
		t.Fatalf("could not build test Search client: %v", err)
	}

	return client, &requests
}

// newIndexDataSourceConfigAndState returns a config holding just the index name,
// plus an empty state to read into, both bound to the data source's schema.
func newIndexDataSourceConfigAndState(ctx context.Context, t *testing.T, d *indexDataSource, indexName string) (tfsdk.Config, tfsdk.State) {
	t.Helper()

	schemaResp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("building schema: %v", schemaResp.Diagnostics)
	}
	dataSourceSchema := schemaResp.Schema

	empty := tftypes.NewValue(dataSourceSchema.Type().TerraformType(ctx), nil)

	// tfsdk.Config cannot be written through a model, so shape the raw value with
	// a State first and hand its Raw to the Config.
	seed := tfsdk.State{Schema: dataSourceSchema, Raw: empty}
	if diags := seed.Set(ctx, nameOnlyIndexDataSourceModel(indexName)); diags.HasError() {
		t.Fatalf("seeding config: %v", diags)
	}

	return tfsdk.Config{Schema: dataSourceSchema, Raw: seed.Raw}, tfsdk.State{Schema: dataSourceSchema, Raw: empty}
}

func nameOnlyIndexDataSourceModel(indexName string) *IndexDataSourceModel {
	return &IndexDataSourceModel{
		Name:          types.StringValue(indexName),
		Primary:       types.StringNull(),
		Entries:       types.Int64Null(),
		DataSize:      types.Int64Null(),
		CreatedAt:     types.StringNull(),
		UpdatedAt:     types.StringNull(),
		Attributes:    types.ObjectNull(attributesAttrTypes),
		Ranking:       types.ObjectNull(rankingAttrTypes),
		Faceting:      types.ObjectNull(facetingAttrTypes),
		Highlighting:  types.ObjectNull(highlightingAttrTypes),
		Pagination:    types.ObjectNull(paginationAttrTypes),
		Typos:         types.ObjectNull(typosAttrTypes),
		Languages:     types.ObjectNull(languagesAttrTypes),
		QueryStrategy: types.ObjectNull(queryStrategyAttrTypes),
		Performance:   types.ObjectNull(performanceAttrTypes),
		Advanced:      types.ObjectNull(advancedAttrTypes),
	}
}
