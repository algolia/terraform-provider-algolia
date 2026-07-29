package index

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/call"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/transport"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// These tests cover the unlinked-virtual-replica paths without credentials, so
// they run in CI. The acceptance tests alongside them prove the same behaviour
// against the live API but skip entirely unless TF_ACC and credentials are set,
// which would leave every one of these branches unguarded on a normal `make test`.

// newSettingsSearchClient returns a client answering every request with the given
// JSON body and HTTP 200, which is enough to drive GetSettings.
func newSettingsSearchClient(t *testing.T, body string) *search.APIClient {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
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

// TestVirtualIndexResourceRead_dropsUnlinkedIndexFromState is the CI-visible
// regression test for the wedge: an index that exists but reports no primary used
// to make Read raise an error, which failed plan, apply and destroy together.
func TestVirtualIndexResourceRead_dropsUnlinkedIndexFromState(t *testing.T) {
	ctx := context.Background()
	// Settings for an index that exists and is not a replica of anything.
	r := &virtualIndexResource{client: newSettingsSearchClient(t, `{"replicas":[]}`)}

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
		t.Fatalf("Read() errored for an unlinked virtual index, which wedges plan, apply and destroy: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("Read() left the unlinked virtual index in state; RemoveResource was not called, so the next apply cannot re-link it")
	}

	warnings := resp.Diagnostics.Warnings()
	if len(warnings) == 0 {
		t.Fatal("Read() dropped the resource without a warning; the user would see a silent recreate")
	}

	detail := warnings[0].Detail()
	for _, want := range []string{"still exists", "advanced.replicas"} {
		if !strings.Contains(detail, want) {
			t.Errorf("warning detail does not mention %q:\n%s", want, detail)
		}
	}
}

// TestVirtualIndexResourceImportState_failsOnUnlinkedIndex pins the asymmetry
// with Read: importing an index that is not a virtual replica is a mistaken
// command, not drift, so it must fail rather than be reconciled.
func TestVirtualIndexResourceImportState_failsOnUnlinkedIndex(t *testing.T) {
	ctx := context.Background()
	r := &virtualIndexResource{client: newSettingsSearchClient(t, `{"replicas":[]}`)}

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	virtualSchema := schemaResp.Schema

	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: virtualSchema,
			Raw:    tftypes.NewValue(virtualSchema.Type().TerraformType(ctx), nil),
		},
	}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "tf-test-standalone"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("ImportState() succeeded for an index that is not a virtual replica, want an error")
	}
	if got, want := resp.Diagnostics.Errors()[0].Summary(), "Index is not a virtual replica"; got != want {
		t.Errorf("error summary = %q, want %q", got, want)
	}
}

// TestReadVirtualIndexClassifiesUnlinked pins the tri-state contract the four
// callers branch on.
func TestReadVirtualIndexClassifiesUnlinked(t *testing.T) {
	cases := []struct {
		name string
		body string
		want virtualIndexState
	}{
		{name: "no primary reported", body: `{"replicas":[]}`, want: virtualIndexUnlinked},
		{name: "empty primary reported", body: `{"primary":""}`, want: virtualIndexUnlinked},
		{name: "primary reported", body: `{"primary":"tf-test-primary"}`, want: virtualIndexFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &virtualIndexResource{client: newSettingsSearchClient(t, tc.body)}
			model := VirtualIndexResourceModel{Name: types.StringValue("tf-test-replica")}

			got, diags := r.readVirtualIndex(context.Background(), &model)

			if diags.HasError() {
				t.Fatalf("readVirtualIndex() diagnostics = %v, want none", diags)
			}
			if got != tc.want {
				t.Errorf("readVirtualIndex() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestWarnDroppedVirtualReplicas covers the warning end to end, including its
// read of the index's current replicas.
func TestWarnDroppedVirtualReplicas(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		planned     []string
		wantWarning bool
	}{
		{
			name:        "planned list omits a linked virtual replica",
			body:        `{"replicas":["virtual(tf-test-asc)"]}`,
			planned:     []string{},
			wantWarning: true,
		},
		{
			name:        "planned list keeps it",
			body:        `{"replicas":["virtual(tf-test-asc)"]}`,
			planned:     []string{"virtual(tf-test-asc)"},
			wantWarning: false,
		},
		{
			name: "nothing planned means the field is not written",
			body: `{"replicas":["virtual(tf-test-asc)"]}`,
			// nil, not empty: the caller nils this out when replicas is undeclared,
			// and an unwritten field cannot drop anything.
			planned:     nil,
			wantWarning: false,
		},
		{
			name:        "standard replica omitted is not reported",
			body:        `{"replicas":["tf-test-standard"]}`,
			planned:     []string{},
			wantWarning: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newSettingsSearchClient(t, tc.body)

			var diags diag.Diagnostics
			warnDroppedVirtualReplicas(context.Background(), client, "tf-test-primary", tc.planned, &diags)

			if diags.HasError() {
				t.Fatalf("warnDroppedVirtualReplicas() errored, it must never fail an apply on its own: %v", diags)
			}
			if got := diags.WarningsCount() > 0; got != tc.wantWarning {
				t.Errorf("warning emitted = %v, want %v (diagnostics: %v)", got, tc.wantWarning, diags)
			}
		})
	}
}

// TestWarnDroppedVirtualReplicasToleratesUnreadableIndex covers the Create case:
// the index does not exist yet, so there is nothing to compare against and the
// check must stay silent rather than fail the apply.
func TestWarnDroppedVirtualReplicasToleratesUnreadableIndex(t *testing.T) {
	var diags diag.Diagnostics
	warnDroppedVirtualReplicas(context.Background(), newNotFoundSearchClient(t), "tf-test-absent", []string{}, &diags)

	if diags.HasError() {
		t.Errorf("warnDroppedVirtualReplicas() errored on an absent index: %v", diags)
	}
	if diags.WarningsCount() > 0 {
		t.Errorf("warnDroppedVirtualReplicas() warned about an index it could not read: %v", diags)
	}
}
