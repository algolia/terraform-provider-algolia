package index

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// A primary index's replicas list holds both kinds of replica, told apart by a
// virtual(...) marker, and two resources write it. Ownership is split by kind:
// algolia_index's advanced.replicas owns the standard entries, each
// algolia_virtual_index owns its own virtual entry. These tests cover the two
// halves that keep those sets disjoint - rejecting a virtual entry declared on the
// wrong resource, and preserving the virtual entries a wholesale write would
// otherwise drop.

func TestStandardReplicasOnly_RejectsAVirtualEntry(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name       string
		entry      types.String
		wantErrors bool
	}{
		{name: "virtual form", entry: types.StringValue("virtual(products_price_asc)"), wantErrors: true},
		{name: "standard name", entry: types.StringValue("products_price_asc"), wantErrors: false},
		{
			// Not the virtual form: no parentheses, so Algolia reads it as a plain
			// index name.
			name:       "name merely starting with virtual",
			entry:      types.StringValue("virtual_products"),
			wantErrors: false,
		},
		{name: "null", entry: types.StringNull(), wantErrors: false},
		{name: "unknown", entry: types.StringUnknown(), wantErrors: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("advanced").AtName("replicas").AtListIndex(0),
				ConfigValue: tc.entry,
			}
			resp := &validator.StringResponse{}

			virtualReplicaNameRejected{}.ValidateString(ctx, req, resp)

			if got := resp.Diagnostics.HasError(); got != tc.wantErrors {
				t.Fatalf("HasError() = %v, want %v (diagnostics: %v)", got, tc.wantErrors, resp.Diagnostics)
			}
			if !tc.wantErrors {
				return
			}
			if detail := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, "algolia_virtual_index") {
				t.Errorf("error does not point at the resource that owns the entry:\n%s", detail)
			}
		})
	}
}

func TestMergeStandardReplicas(t *testing.T) {
	cases := []struct {
		name       string
		serverBody string
		configured []string
		want       []string
		wantError  string
	}{
		{
			// The whole point: a wholesale write of the standard entries must not
			// unlink a virtual replica that another resource owns.
			name:       "virtual entries are preserved",
			serverBody: `{"replicas":["virtual(products_cheapest)","products_price_asc"]}`,
			configured: []string{"products_price_asc"},
			want:       []string{"products_price_asc", "virtual(products_cheapest)"},
		},
		{
			name:       "removing a standard replica still removes it",
			serverBody: `{"replicas":["products_price_asc","products_price_desc"]}`,
			configured: []string{"products_price_asc"},
			want:       []string{"products_price_asc"},
		},
		{
			name:       "an empty configured list keeps the virtual entries",
			serverBody: `{"replicas":["virtual(products_cheapest)","products_price_asc"]}`,
			configured: []string{},
			want:       []string{"virtual(products_cheapest)"},
		},
		{
			name:       "nothing to preserve",
			serverBody: `{"replicas":[]}`,
			configured: []string{"products_price_asc"},
			want:       []string{"products_price_asc"},
		},
		{
			// Algolia cannot hold one index as a replica in both modes, so this is a
			// disagreement between two resources rather than something to merge.
			name:       "the same replica declared standard and linked virtual",
			serverBody: `{"replicas":["virtual(products_price_asc)"]}`,
			configured: []string{"products_price_asc"},
			wantError:  "both standard and virtual",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newSettingsSearchClient(t, tc.serverBody)

			got, diags := mergeStandardReplicas(context.Background(), client, "products", tc.configured)

			if tc.wantError != "" {
				if !diags.HasError() {
					t.Fatalf("mergeStandardReplicas() = %v with no error, want an error containing %q", got, tc.wantError)
				}
				if summary := diags.Errors()[0].Summary(); !strings.Contains(summary, tc.wantError) {
					t.Errorf("error summary = %q, want it to contain %q", summary, tc.wantError)
				}

				return
			}

			if diags.HasError() {
				t.Fatalf("mergeStandardReplicas() diagnostics = %v", diags)
			}
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Errorf("mergeStandardReplicas() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestMergeStandardReplicas_ToleratesAnAbsentIndex covers Create: the index does
// not exist yet, so there is nothing to preserve and the configured list stands.
func TestMergeStandardReplicas_ToleratesAnAbsentIndex(t *testing.T) {
	client := newNotFoundSearchClient(t)

	got, diags := mergeStandardReplicas(context.Background(), client, "products", []string{"products_price_asc"})

	if diags.HasError() {
		t.Fatalf("mergeStandardReplicas() diagnostics = %v, want none for an index that does not exist yet", diags)
	}
	if strings.Join(got, ",") != "products_price_asc" {
		t.Errorf("mergeStandardReplicas() = %v, want the configured list unchanged", got)
	}
}

// TestStandardReplicasOf is the read-side half of the ownership split. Surfacing a
// virtual entry through advanced.replicas would put a value in state the attribute
// can never declare, so every refresh would plan a change no apply could settle -
// which is exactly what a live acceptance run showed before this filter existed.
func TestStandardReplicasOf(t *testing.T) {
	cases := []struct {
		name     string
		replicas []string
		want     []string
	}{
		{
			name:     "virtual entries are dropped",
			replicas: []string{"products_price_asc", "virtual(products_cheapest)"},
			want:     []string{"products_price_asc"},
		},
		{
			name:     "only virtual entries yields an empty list",
			replicas: []string{"virtual(products_cheapest)"},
			want:     []string{},
		},
		{
			name:     "only standard entries are untouched",
			replicas: []string{"products_price_asc", "products_price_desc"},
			want:     []string{"products_price_asc", "products_price_desc"},
		},
		{
			// Distinct from an empty list: nil means the API reported no replicas
			// field at all, which must stay null rather than become [].
			name:     "nil stays nil",
			replicas: nil,
			want:     nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := standardReplicasOf(tc.replicas)

			if tc.want == nil {
				if got != nil {
					t.Errorf("standardReplicasOf(nil) = %v, want nil", got)
				}

				return
			}
			if got == nil {
				t.Fatalf("standardReplicasOf(%v) = nil, want %v", tc.replicas, tc.want)
			}
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Errorf("standardReplicasOf(%v) = %v, want %v", tc.replicas, got, tc.want)
			}
		})
	}
}
