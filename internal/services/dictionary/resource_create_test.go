package dictionary

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/call"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/transport"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestDictionaryEntryResourceCreate_persistsGeneratedObjectIDBeforeFailingWait
// pins the guarantee that keeps a failed create retryable rather than
// duplicating. object_id is generated locally by generateObjectID, so if Create
// returned without recording it, the entry would exist in Algolia under an
// object_id nothing knows about - and because generateObjectID mints a fresh
// UUID on every attempt, each retry would leave another undeleted entry behind
// instead of retrying the same one.
//
// The fake host below accepts the batch write and its task, then fails the
// search that waits for the entry to become visible.
func TestDictionaryEntryResourceCreate_persistsGeneratedObjectIDBeforeFailingWait(t *testing.T) {
	ctx := context.Background()

	var writtenObjectID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.HasSuffix(r.URL.Path, "/batch"):
			writtenObjectID = batchObjectID(t, r)
			_, _ = w.Write([]byte(`{"taskID":42,"updatedAt":"2024-01-01T00:00:00Z"}`))
		case strings.HasPrefix(r.URL.Path, "/1/task/"):
			_, _ = w.Write([]byte(`{"status":"published","pendingTask":false}`))
		default:
			// The searchable-visibility wait (SearchDictionaryEntries). 4xx is
			// not retried by the v4 transport, so it fails immediately.
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"dictionary search unavailable","status":400}`))
		}
	}))
	defer server.Close()

	r := &dictionaryEntryResource{client: newTestSearchClient(t, server)}
	resp := &resource.CreateResponse{State: emptyEntryState(t)}
	r.Create(ctx, resource.CreateRequest{Plan: dictionaryEntryCreatePlan(t)}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Create() reported no error although the visibility wait failed; this test is no longer exercising the failure path")
	}
	if resp.State.Raw.IsNull() {
		t.Fatal("Create() left state empty after the entry was written: the entry exists in Algolia and the next apply would generate a new object_id, leaving this one undeleted")
	}
	if !resp.State.Raw.IsFullyKnown() {
		t.Fatalf("Create() persisted unknown values, which Terraform rejects as an apply result: %s", resp.State.Raw)
	}
	if writtenObjectID == "" {
		t.Fatal("the fake host never saw an addEntry request; the test cannot prove which object_id was persisted")
	}

	var got DictionaryEntryResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading back the persisted state: %v", diags)
	}

	if got.ObjectID.ValueString() != writtenObjectID {
		t.Errorf("persisted object_id = %q, want %q (the object_id actually written), so a retry reuses the same entry", got.ObjectID.ValueString(), writtenObjectID)
	}
	if want := "stopwords/" + writtenObjectID; got.ID.ValueString() != want {
		t.Errorf("persisted id = %q, want %q", got.ID.ValueString(), want)
	}
	// expandDictionaryEntry defaults a stopwords entry's state to "enabled", so
	// the Optional+Computed `state` attribute is already decided at this point
	// and must not be left unknown.
	if got.State.ValueString() != "enabled" {
		t.Errorf("persisted state = %v, want \"enabled\" (the default expand applied to the entry it wrote)", got.State)
	}
}

// TestDictionarySettingsResourceCreate_persistsBeforeFailingReadBack covers the
// singleton settings resource: nothing is "created" remotely, but Create does
// disable standard entries application-wide, and only this resource's Delete
// ever resets them. A settings change that never reached state would therefore
// never be undone either.
func TestDictionarySettingsResourceCreate_persistsBeforeFailingReadBack(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPut:
			_, _ = w.Write([]byte(`{"taskID":7,"updatedAt":"2024-01-01T00:00:00Z"}`))
		case strings.HasPrefix(r.URL.Path, "/1/task/"):
			_, _ = w.Write([]byte(`{"status":"published","pendingTask":false}`))
		default:
			// GetDictionarySettings, the read-back.
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"settings unavailable","status":400}`))
		}
	}))
	defer server.Close()

	r := &dictionarySettingsResource{client: newTestSearchClient(t, server), appID: "test-app"}
	resp := &resource.CreateResponse{State: emptySettingsState(t)}
	r.Create(ctx, resource.CreateRequest{Plan: dictionarySettingsCreatePlan(t)}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Create() reported no error although the read-back failed; this test is no longer exercising the failure path")
	}
	if resp.State.Raw.IsNull() {
		t.Fatal("Create() left state empty after the settings were applied: standard entries stay disabled with no Terraform record, so nothing would ever reset them")
	}
	if !resp.State.Raw.IsFullyKnown() {
		t.Fatalf("Create() persisted unknown values, which Terraform rejects as an apply result: %s", resp.State.Raw)
	}

	var got DictionarySettingsResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading back the persisted state: %v", diags)
	}

	if got.ID.ValueString() != "test-app" {
		t.Errorf("persisted id = %q, want the application ID", got.ID.ValueString())
	}
	// disable_standard_entries is Optional+Computed, so it must be persisted as
	// a typed Object rather than left unknown.
	if got.DisableStandardEntries.IsNull() || got.DisableStandardEntries.IsUnknown() {
		t.Fatalf("persisted disable_standard_entries = %v, want the settings that were just written", got.DisableStandardEntries)
	}

	var block DisableStandardEntriesModel
	if diags := got.DisableStandardEntries.As(ctx, &block, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatalf("decoding the persisted disable_standard_entries: %v", diags)
	}
	if len(block.Stopwords.Elements()) != 1 {
		t.Errorf("persisted disable_standard_entries.stopwords = %v, want the single configured language", block.Stopwords)
	}
}

// batchObjectID extracts the objectID from a BatchDictionaryEntries request, so
// a test can compare what was written remotely with what was persisted.
func batchObjectID(t *testing.T, r *http.Request) string {
	t.Helper()

	var body struct {
		Requests []struct {
			Body struct {
				ObjectID string `json:"objectID"`
			} `json:"body"`
		} `json:"requests"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode the batch request body: %v", err)
	}
	if len(body.Requests) != 1 {
		t.Fatalf("batch request carried %d entries, want 1", len(body.Requests))
	}

	return body.Requests[0].Body.ObjectID
}

// newTestSearchClient returns a Search client whose only host is the given test
// server, mirroring newTestSearchClient in internal/services/apikey.
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

// dictionaryEntryCreatePlan builds the plan Terraform hands to Create for a
// stopwords entry with no configured object_id: id, object_id and state are all
// unknown, everything else holds the configuration verbatim.
func dictionaryEntryCreatePlan(t *testing.T) tfsdk.Plan {
	t.Helper()

	ctx := context.Background()
	entrySchema := dictionaryEntryResourceSchema()
	plan := tfsdk.Plan{
		Raw:    tftypes.NewValue(entrySchema.Type().TerraformType(ctx), nil),
		Schema: entrySchema,
	}

	diags := plan.Set(ctx, &DictionaryEntryResourceModel{
		ID:            types.StringUnknown(),
		Dictionary:    types.StringValue("stopwords"),
		ObjectID:      types.StringUnknown(),
		Language:      types.StringValue("en"),
		Word:          types.StringValue("tf-test-stopword"),
		Words:         types.ListNull(types.StringType),
		Decomposition: types.ListNull(types.StringType),
		State:         types.StringUnknown(),
	})
	if diags.HasError() {
		t.Fatalf("could not build the create plan: %v", diags)
	}

	return plan
}

func dictionarySettingsCreatePlan(t *testing.T) tfsdk.Plan {
	t.Helper()

	ctx := context.Background()
	settingsSchema := dictionarySettingsResourceSchema()
	plan := tfsdk.Plan{
		Raw:    tftypes.NewValue(settingsSchema.Type().TerraformType(ctx), nil),
		Schema: settingsSchema,
	}

	stopwords := types.MapValueMust(types.BoolType, map[string]attr.Value{"en": types.BoolValue(true)})
	disableStandardEntries, objDiags := types.ObjectValue(disableStandardEntriesAttrTypes, map[string]attr.Value{"stopwords": stopwords})
	if objDiags.HasError() {
		t.Fatalf("could not build disable_standard_entries: %v", objDiags)
	}

	diags := plan.Set(ctx, &DictionarySettingsResourceModel{
		ID:                     types.StringUnknown(),
		DisableStandardEntries: disableStandardEntries,
	})
	if diags.HasError() {
		t.Fatalf("could not build the create plan: %v", diags)
	}

	return plan
}

func emptyEntryState(t *testing.T) tfsdk.State {
	t.Helper()

	entrySchema := dictionaryEntryResourceSchema()

	return tfsdk.State{
		Raw:    tftypes.NewValue(entrySchema.Type().TerraformType(context.Background()), nil),
		Schema: entrySchema,
	}
}

func emptySettingsState(t *testing.T) tfsdk.State {
	t.Helper()

	settingsSchema := dictionarySettingsResourceSchema()

	return tfsdk.State{
		Raw:    tftypes.NewValue(settingsSchema.Type().TerraformType(context.Background()), nil),
		Schema: settingsSchema,
	}
}
