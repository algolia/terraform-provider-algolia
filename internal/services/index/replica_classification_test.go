package index

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/call"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/transport"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Distinguishing a virtual replica from a standard one takes two reads - the
// replica's settings report a primary either way, and only the primary's replicas
// list carries the virtual(...) marker - so these tests need a server that answers
// per index rather than one body for every request.

// indexNameFromPath extracts the index name from an Algolia index path such as
// /1/indexes/{name}/settings.
func indexNameFromPath(path string) string {
	trimmed := strings.TrimPrefix(path, "/1/indexes/")
	if i := strings.Index(trimmed, "/"); i >= 0 {
		trimmed = trimmed[:i]
	}

	name, err := url.PathUnescape(trimmed)
	if err != nil {
		return trimmed
	}

	return name
}

// newRoutedSearchClient answers GetSettings per index name, and 404s for any index
// absent from the map. Writes are accepted and recorded; task lookups report the
// task as published so waitForIndexTask returns immediately.
func newRoutedSearchClient(t *testing.T, settingsByIndex map[string]string, recorded *[][]string) *search.APIClient {
	t.Helper()

	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		name := indexNameFromPath(r.URL.Path)

		switch {
		case strings.Contains(r.URL.Path, "/task/"):
			_, _ = w.Write([]byte(`{"status":"published","pendingTask":false}`))

		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/settings"):
			if recorded != nil {
				var body struct {
					Replicas []string `json:"replicas"`
				}
				_ = json.NewDecoder(r.Body).Decode(&body)

				mu.Lock()
				*recorded = append(*recorded, body.Replicas)
				mu.Unlock()
			}
			_, _ = w.Write([]byte(`{"taskID":1,"updatedAt":"2026-07-29T00:00:00.000Z"}`))

		default:
			body, ok := settingsByIndex[name]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"message":"Index does not exist","status":404}`))

				return
			}
			_, _ = w.Write([]byte(body))
		}
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

const (
	testReplicaName = "tf-test-replica"
	testPrimaryName = "tf-test-primary"
)

// TestReadVirtualIndexClassifiesLinkage pins the four-state contract, and in
// particular that a reported primary index is not on its own enough to call
// something a virtual replica.
func TestReadVirtualIndexClassifiesLinkage(t *testing.T) {
	cases := []struct {
		name     string
		settings map[string]string
		want     virtualIndexState
	}{
		{
			name: "primary lists it in virtual form",
			settings: map[string]string{
				testReplicaName: `{"primary":"tf-test-primary"}`,
				testPrimaryName: `{"replicas":["virtual(tf-test-replica)"]}`,
			},
			want: virtualIndexFound,
		},
		{
			name: "primary lists it under its plain name",
			settings: map[string]string{
				testReplicaName: `{"primary":"tf-test-primary"}`,
				testPrimaryName: `{"replicas":["tf-test-replica"]}`,
			},
			want: virtualIndexStandardReplica,
		},
		{
			name: "primary no longer lists it at all",
			settings: map[string]string{
				testReplicaName: `{"primary":"tf-test-primary"}`,
				testPrimaryName: `{"replicas":[]}`,
			},
			want: virtualIndexUnlinked,
		},
		{
			name: "primary no longer exists",
			settings: map[string]string{
				testReplicaName: `{"primary":"tf-test-primary"}`,
			},
			want: virtualIndexUnlinked,
		},
		{
			name: "index reports no primary",
			settings: map[string]string{
				testReplicaName: `{"replicas":[]}`,
			},
			want: virtualIndexUnlinked,
		},
		{
			name:     "index does not exist",
			settings: map[string]string{},
			want:     virtualIndexAbsent,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &virtualIndexResource{client: newRoutedSearchClient(t, tc.settings, nil)}
			model := VirtualIndexResourceModel{Name: types.StringValue(testReplicaName)}

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

// TestVirtualIndexResourceRead_keepsStandardReplicaInState covers the deliberate
// asymmetry with the unlinked case: a standard replica holds a full copy of the
// primary's records, so Terraform must keep tracking it - dropping it would forget
// a record-bearing index and put it beyond deletion_protection - while still
// saying loudly that it is not what the configuration asked for.
func TestVirtualIndexResourceRead_keepsStandardReplicaInState(t *testing.T) {
	ctx := context.Background()
	r := &virtualIndexResource{client: newRoutedSearchClient(t, map[string]string{
		"tf-test-deleted-virtual-out-of-band": `{"primary":"tf-test-primary"}`,
		testPrimaryName:                       `{"replicas":["tf-test-deleted-virtual-out-of-band"]}`,
	}, nil)}

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
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
		t.Fatalf("Read() errored for a standard replica, which would wedge plan, apply and destroy: %v", resp.Diagnostics)
	}
	if resp.State.Raw.IsNull() {
		t.Fatal("Read() removed a standard replica from state; Terraform would forget an index holding a copy of the primary's records")
	}

	warnings := resp.Diagnostics.Warnings()
	if len(warnings) == 0 {
		t.Fatal("Read() reported no warning for a standard replica")
	}

	detail := warnings[0].Detail()
	for _, want := range []string{"standard replica", "virtual(tf-test-deleted-virtual-out-of-band)", "algolia_index"} {
		if !strings.Contains(detail, want) {
			t.Errorf("warning detail does not mention %q:\n%s", want, detail)
		}
	}
}

// TestVirtualIndexResourceImportState_failsOnStandardReplica pins the loud half:
// adopting a standard replica would put a record-bearing index under a resource
// that promises a view over the primary.
func TestVirtualIndexResourceImportState_failsOnStandardReplica(t *testing.T) {
	ctx := context.Background()
	r := &virtualIndexResource{client: newRoutedSearchClient(t, map[string]string{
		testReplicaName: `{"primary":"tf-test-primary"}`,
		testPrimaryName: `{"replicas":["tf-test-replica"]}`,
	}, nil)}

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	virtualSchema := schemaResp.Schema

	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: virtualSchema,
			Raw:    tftypes.NewValue(virtualSchema.Type().TerraformType(ctx), nil),
		},
	}
	r.ImportState(ctx, resource.ImportStateRequest{ID: testReplicaName}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("ImportState() succeeded for a standard replica, want an error")
	}
	if got, want := resp.Diagnostics.Errors()[0].Summary(), "Index is a standard replica"; got != want {
		t.Errorf("error summary = %q, want %q", got, want)
	}
}

// TestRemoveVirtualReplicaLinkDropsEitherForm covers the destroy path. Algolia
// refuses deleteIndex on an index that is still a replica, so an unlink that only
// matched virtual(...) left a standard replica linked and the delete then failed
// with "cannot apply the deleteIndex operation on a replica index".
func TestRemoveVirtualReplicaLinkDropsEitherForm(t *testing.T) {
	cases := []struct {
		name         string
		primaryBody  string
		wantWritten  bool
		wantReplicas []string
	}{
		{
			name:         "virtual form is removed",
			primaryBody:  `{"replicas":["virtual(tf-test-replica)","other"]}`,
			wantWritten:  true,
			wantReplicas: []string{"other"},
		},
		{
			name:         "plain form is removed too",
			primaryBody:  `{"replicas":["tf-test-replica","other"]}`,
			wantWritten:  true,
			wantReplicas: []string{"other"},
		},
		{
			name:         "both forms are removed",
			primaryBody:  `{"replicas":["tf-test-replica","virtual(tf-test-replica)","other"]}`,
			wantWritten:  true,
			wantReplicas: []string{"other"},
		},
		{
			name:        "nothing to remove writes nothing",
			primaryBody: `{"replicas":["other"]}`,
			wantWritten: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var written [][]string
			client := newRoutedSearchClient(t, map[string]string{testPrimaryName: tc.primaryBody}, &written)

			if err := removeVirtualReplicaLink(context.Background(), client, testPrimaryName, testReplicaName); err != nil {
				t.Fatalf("removeVirtualReplicaLink() error = %v", err)
			}

			if !tc.wantWritten {
				if len(written) != 0 {
					t.Errorf("wrote replicas %v, want no write at all", written)
				}

				return
			}

			if len(written) != 1 {
				t.Fatalf("wrote %d times, want exactly 1: %v", len(written), written)
			}
			if strings.Join(written[0], ",") != strings.Join(tc.wantReplicas, ",") {
				t.Errorf("wrote replicas %v, want %v", written[0], tc.wantReplicas)
			}
		})
	}
}

// TestEnsureVirtualReplicaLinkedConvertsStandardEntry covers the repair path: a
// primary listing the replica under its plain name must end up listing it in
// virtual form only. Leaving both entries would have the primary declare one index
// as a replica twice, in two different modes.
func TestEnsureVirtualReplicaLinkedConvertsStandardEntry(t *testing.T) {
	cases := []struct {
		name         string
		primaryBody  string
		wantWritten  bool
		wantReplicas []string
	}{
		{
			name:         "plain entry is replaced by the virtual form",
			primaryBody:  `{"replicas":["tf-test-replica","other"]}`,
			wantWritten:  true,
			wantReplicas: []string{"other", "virtual(tf-test-replica)"},
		},
		{
			name:         "already virtual is left untouched",
			primaryBody:  `{"replicas":["virtual(tf-test-replica)"]}`,
			wantWritten:  false,
			wantReplicas: nil,
		},
		{
			name:         "absent is appended",
			primaryBody:  `{"replicas":["other"]}`,
			wantWritten:  true,
			wantReplicas: []string{"other", "virtual(tf-test-replica)"},
		},
		{
			name:         "both forms present collapses to the virtual one",
			primaryBody:  `{"replicas":["tf-test-replica","virtual(tf-test-replica)"]}`,
			wantWritten:  true,
			wantReplicas: []string{"virtual(tf-test-replica)"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var written [][]string
			client := newRoutedSearchClient(t, map[string]string{testPrimaryName: tc.primaryBody}, &written)

			if err := ensureVirtualReplicaLinked(context.Background(), client, testPrimaryName, testReplicaName); err != nil {
				t.Fatalf("ensureVirtualReplicaLinked() error = %v", err)
			}

			if !tc.wantWritten {
				if len(written) != 0 {
					t.Errorf("wrote replicas %v, want no write at all", written)
				}

				return
			}

			if len(written) != 1 {
				t.Fatalf("wrote %d times, want exactly 1: %v", len(written), written)
			}
			if strings.Join(written[0], ",") != strings.Join(tc.wantReplicas, ",") {
				t.Errorf("wrote replicas %v, want %v", written[0], tc.wantReplicas)
			}
		})
	}
}
