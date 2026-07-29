package index

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/call"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/transport"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// These tests cover recovery from an out-of-band deletion. An index deleted with
// `algolia indices delete` used to wedge the resource: Read turned the 404 from
// GetSettings into a diagnostic, so plan, apply and destroy all failed until the
// operator ran `terraform state rm`. Read must instead drop the resource from
// state -- while ImportState must keep failing, because importing an index that
// does not exist cannot produce usable state.

// newNotFoundSearchClient returns a client whose every request answers 404, the
// way the Algolia API answers a request for a deleted index.
func newNotFoundSearchClient(t *testing.T) *search.APIClient {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Index does not exist","status":404}`))
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

	return client
}

func TestIndexResourceRead_removesDeletedIndexFromState(t *testing.T) {
	ctx := context.Background()
	r := &indexResource{client: newNotFoundSearchClient(t)}

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("building schema: %v", schemaResp.Diagnostics)
	}
	indexSchema := schemaResp.Schema

	state := tfsdk.State{
		Schema: indexSchema,
		Raw:    tftypes.NewValue(indexSchema.Type().TerraformType(ctx), nil),
	}
	if diags := state.Set(ctx, deletedIndexModel()); diags.HasError() {
		t.Fatalf("seeding state: %v", diags)
	}

	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported an error for a deleted index, want a clean removal: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("Read() left the deleted index in state; RemoveResource was not called, so plan/apply/destroy stay blocked")
	}
}

func TestIndexResourceImportState_failsOnMissingIndex(t *testing.T) {
	ctx := context.Background()
	r := &indexResource{client: newNotFoundSearchClient(t)}

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	indexSchema := schemaResp.Schema

	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: indexSchema,
			Raw:    tftypes.NewValue(indexSchema.Type().TerraformType(ctx), nil),
		},
	}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "tf-test-nonexistent"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("ImportState() succeeded for an index that does not exist, want an error")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Error reading index" {
		t.Errorf("error summary = %q, want %q", got, "Error reading index")
	}
}

func TestVirtualIndexResourceRead_removesDeletedIndexFromState(t *testing.T) {
	ctx := context.Background()
	r := &virtualIndexResource{client: newNotFoundSearchClient(t)}

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("building schema: %v", schemaResp.Diagnostics)
	}
	virtualSchema := schemaResp.Schema

	state := tfsdk.State{
		Schema: virtualSchema,
		Raw:    tftypes.NewValue(virtualSchema.Type().TerraformType(ctx), nil),
	}
	if diags := state.Set(ctx, deletedVirtualIndexModel()); diags.HasError() {
		t.Fatalf("seeding state: %v", diags)
	}

	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported an error for a deleted virtual index, want a clean removal: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("Read() left the deleted virtual index in state; RemoveResource was not called")
	}
}

func TestVirtualIndexResourceImportState_failsOnMissingIndex(t *testing.T) {
	ctx := context.Background()
	r := &virtualIndexResource{client: newNotFoundSearchClient(t)}

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	virtualSchema := schemaResp.Schema

	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: virtualSchema,
			Raw:    tftypes.NewValue(virtualSchema.Type().TerraformType(ctx), nil),
		},
	}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "tf-test-nonexistent"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("ImportState() succeeded for a virtual index that does not exist, want an error")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Error reading index" {
		t.Errorf("error summary = %q, want %q", got, "Error reading index")
	}
}

// TestReadIndexReportsAbsenceWithoutDiagnostics pins the contract the two callers
// rely on: a 404 is reported as absence, not as a diagnostic.
func TestReadIndexReportsAbsenceWithoutDiagnostics(t *testing.T) {
	r := &indexResource{client: newNotFoundSearchClient(t)}
	model := IndexResourceModel{Name: types.StringValue("tf-test-nonexistent")}

	found, diags := r.readIndex(context.Background(), &model)

	if diags.HasError() {
		t.Errorf("readIndex() diagnostics = %v, want none for a 404", diags)
	}
	if found {
		t.Error("readIndex() found = true for a 404, want false")
	}
}

func TestReadIndexModelReportsAbsenceWithoutDiagnostics(t *testing.T) {
	r := &virtualIndexResource{client: newNotFoundSearchClient(t)}
	model := IndexResourceModel{Name: types.StringValue("tf-test-nonexistent")}

	found, diags := r.readIndexModel(context.Background(), &model)

	if diags.HasError() {
		t.Errorf("readIndexModel() diagnostics = %v, want none for a 404", diags)
	}
	if found {
		t.Error("readIndexModel() found = true for a 404, want false")
	}
}

func deletedIndexModel() *IndexResourceModel {
	return &IndexResourceModel{
		Name:               types.StringValue("tf-test-deleted-out-of-band"),
		DeletionProtection: types.BoolValue(false),
		Primary:            types.StringNull(),
		Entries:            types.Int64Value(0),
		DataSize:           types.Int64Value(0),
		CreatedAt:          types.StringValue(""),
		UpdatedAt:          types.StringValue(""),
		Attributes:         types.ObjectNull(attributesAttrTypes),
		Ranking:            types.ObjectNull(rankingAttrTypes),
		Faceting:           types.ObjectNull(facetingAttrTypes),
		Highlighting:       types.ObjectNull(highlightingAttrTypes),
		Pagination:         types.ObjectNull(paginationAttrTypes),
		Typos:              types.ObjectNull(typosAttrTypes),
		Languages:          types.ObjectNull(languagesAttrTypes),
		QueryStrategy:      types.ObjectNull(queryStrategyAttrTypes),
		Performance:        types.ObjectNull(performanceAttrTypes),
		Advanced:           types.ObjectNull(advancedAttrTypes),
	}
}

func deletedVirtualIndexModel() *VirtualIndexResourceModel {
	return &VirtualIndexResourceModel{
		Name:               types.StringValue("tf-test-deleted-virtual-out-of-band"),
		PrimaryIndexName:   types.StringValue("tf-test-primary"),
		DeletionProtection: types.BoolValue(false),
		Entries:            types.Int64Value(0),
		DataSize:           types.Int64Value(0),
		CreatedAt:          types.StringValue(""),
		UpdatedAt:          types.StringValue(""),
		Attributes:         types.ObjectNull(attributesAttrTypes),
		Ranking:            types.ObjectNull(virtualRankingAttrTypes),
		Faceting:           types.ObjectNull(facetingAttrTypes),
		Highlighting:       types.ObjectNull(highlightingAttrTypes),
		Pagination:         types.ObjectNull(paginationAttrTypes),
		Typos:              types.ObjectNull(typosAttrTypes),
		Languages:          types.ObjectNull(languagesAttrTypes),
		QueryStrategy:      types.ObjectNull(queryStrategyAttrTypes),
		Performance:        types.ObjectNull(performanceAttrTypes),
		Advanced:           types.ObjectNull(advancedAttrTypes),
	}
}
