package apikey

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/call"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/transport"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflogtest"
)

// secretKey is a stand-in for a real Algolia API key: this resource's id *is* the
// key value, so it must not reach diagnostics or logs. It is deliberately a long,
// distinctive string so a substring search cannot match by accident.
const secretKey = "d0n0tl0gth1sadm1nk3y0123456789ab"

// TestAPIKeyResource_KeyNeverLeaksIntoDiagnosticsOrLogs drives every CRUD path
// that has the key in hand against a failing API and asserts that neither the
// diagnostics nor the TF_LOG output contains the key value.
func TestAPIKeyResource_KeyNeverLeaksIntoDiagnosticsOrLogs(t *testing.T) {
	tests := []struct {
		name string
		// status is what the fake Algolia host returns. 4xx is not retried by the
		// v4 transport, so the operation fails on the first attempt.
		status int
		run    func(t *testing.T, ctx context.Context, r *apiKeyResource) diag.Diagnostics
	}{
		{
			name:   "Read",
			status: http.StatusBadRequest,
			run: func(t *testing.T, ctx context.Context, r *apiKeyResource) diag.Diagnostics {
				resp := &resource.ReadResponse{State: emptyResourceState(t)}
				r.Read(ctx, resource.ReadRequest{State: resourceStateWithKey(t)}, resp)

				return resp.Diagnostics
			},
		},
		{
			name: "Read404RemovesResource",
			// The 404 branch logs "API key not found; removing from state"; it used
			// to attach the key as an "id" log field.
			status: http.StatusNotFound,
			run: func(t *testing.T, ctx context.Context, r *apiKeyResource) diag.Diagnostics {
				resp := &resource.ReadResponse{State: resourceStateWithKey(t)}
				r.Read(ctx, resource.ReadRequest{State: resourceStateWithKey(t)}, resp)

				return resp.Diagnostics
			},
		},
		{
			name:   "Update",
			status: http.StatusBadRequest,
			run: func(t *testing.T, ctx context.Context, r *apiKeyResource) diag.Diagnostics {
				resp := &resource.UpdateResponse{State: emptyResourceState(t)}
				r.Update(ctx, resource.UpdateRequest{Plan: resourcePlanWithKey(t)}, resp)

				return resp.Diagnostics
			},
		},
		{
			name:   "Delete",
			status: http.StatusBadRequest,
			run: func(t *testing.T, ctx context.Context, r *apiKeyResource) diag.Diagnostics {
				resp := &resource.DeleteResponse{State: resourceStateWithKey(t)}
				r.Delete(ctx, resource.DeleteRequest{State: resourceStateWithKey(t)}, resp)

				return resp.Diagnostics
			},
		},
		{
			name:   "ImportState",
			status: http.StatusBadRequest,
			run: func(t *testing.T, ctx context.Context, r *apiKeyResource) diag.Diagnostics {
				resp := &resource.ImportStateResponse{State: emptyResourceState(t)}
				r.ImportState(ctx, resource.ImportStateRequest{ID: secretKey}, resp)

				return resp.Diagnostics
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				// Echo the key back the way the real API echoes the requested
				// resource, so the error string genuinely carries it.
				_, _ = w.Write([]byte(`{"message":"cannot operate on key ` + secretKey + `"}`))
			}))
			defer server.Close()

			var logs bytes.Buffer
			ctx := tflogtest.RootLogger(context.Background(), &logs)

			r := &apiKeyResource{client: newTestSearchClient(t, server), now: time.Now}
			diags := tt.run(t, ctx, r)

			for _, d := range diags {
				if strings.Contains(d.Summary(), secretKey) {
					t.Errorf("diagnostic summary leaks the API key: %s", d.Summary())
				}
				if strings.Contains(d.Detail(), secretKey) {
					t.Errorf("diagnostic detail leaks the API key: %s", d.Detail())
				}
			}

			assertLogsFreeOfKey(t, logs.String())
		})
	}
}

// TestAPIKeyResource_ReadLogsCarryNoKeyField pins the specific regression: the
// Read debug log used to be emitted with map[string]any{"id": key}.
func TestAPIKeyResource_ReadLogsCarryNoKeyField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":"` + secretKey + `","acl":["search"],"createdAt":1700000000}`))
	}))
	defer server.Close()

	var logs bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &logs)

	r := &apiKeyResource{client: newTestSearchClient(t, server), now: time.Now}
	resp := &resource.ReadResponse{State: emptyResourceState(t)}
	r.Read(ctx, resource.ReadRequest{State: resourceStateWithKey(t)}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() diagnostics = %v, want none", resp.Diagnostics)
	}

	entries, err := tflogtest.MultilineJSONDecode(&logs)
	if err != nil {
		t.Fatalf("could not decode log output: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("Read() emitted no log entries; the assertion below would be vacuous")
	}

	for _, entry := range entries {
		for field, value := range entry {
			if str, ok := value.(string); ok && strings.Contains(str, secretKey) {
				t.Errorf("log field %q leaks the API key: %s", field, str)
			}
		}
	}
}

// TestRedactKey covers the helper that strips the key from transport errors,
// which wrap the request URL (the key endpoints are /1/keys/{key}).
func TestRedactKey(t *testing.T) {
	if got := redactKey(nil, secretKey); got != "" {
		t.Errorf("redactKey(nil) = %q, want empty", got)
	}

	err := &search.APIError{Status: 400, Message: `Get "https://x.algolia.net/1/keys/` + secretKey + `": boom`}
	got := redactKey(err, secretKey)
	if strings.Contains(got, secretKey) {
		t.Errorf("redactKey() = %q, still contains the key", got)
	}
	if !strings.Contains(got, "***") {
		t.Errorf("redactKey() = %q, want the key replaced with ***", got)
	}
}

func TestKeyLabel(t *testing.T) {
	if got, want := keyLabel(types.StringNull()), "the API key"; got != want {
		t.Errorf("keyLabel(null) = %q, want %q", got, want)
	}
	if got, want := keyLabel(types.StringValue("")), "the API key"; got != want {
		t.Errorf("keyLabel(empty) = %q, want %q", got, want)
	}
	if got, want := keyLabel(types.StringValue("search-only")), `the API key described as "search-only"`; got != want {
		t.Errorf("keyLabel(description) = %q, want %q", got, want)
	}
}

func assertLogsFreeOfKey(t *testing.T, output string) {
	t.Helper()

	if output == "" {
		return
	}
	if strings.Contains(output, secretKey) {
		t.Errorf("log output leaks the API key:\n%s", output)
	}
}

// newTestSearchClient returns a Search client whose only host is the given test
// server, mirroring newTestRecommendClient in internal/services/recommend.
func newTestSearchClient(t *testing.T, server *httptest.Server) *search.APIClient {
	t.Helper()

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

func resourceStateWithKey(t *testing.T) tfsdk.State {
	t.Helper()

	state := emptyResourceState(t)
	diags := state.Set(context.Background(), &APIKeyResourceModel{
		ID:                     types.StringValue(secretKey),
		ACL:                    types.SetValueMust(types.StringType, []attr.Value{types.StringValue("search")}),
		Description:            types.StringValue("tf-acc-test-key"),
		ExpiresAt:              types.StringNull(),
		Indexes:                types.SetNull(types.StringType),
		Referers:               types.SetNull(types.StringType),
		MaxHitsPerQuery:        types.Int64Null(),
		MaxQueriesPerIPPerHour: types.Int64Null(),
		QueryParameters:        types.StringNull(),
		CreatedAt:              types.StringValue("2024-01-01T00:00:00Z"),
	})
	if diags.HasError() {
		t.Fatalf("could not build test state: %v", diags)
	}

	return state
}

func resourcePlanWithKey(t *testing.T) tfsdk.Plan {
	t.Helper()

	return tfsdk.Plan(resourceStateWithKey(t))
}

func emptyResourceState(t *testing.T) tfsdk.State {
	t.Helper()

	schema := apiKeyResourceSchema()

	return tfsdk.State{
		Raw:    tftypes.NewValue(schema.Type().TerraformType(context.Background()), nil),
		Schema: schema,
	}
}
