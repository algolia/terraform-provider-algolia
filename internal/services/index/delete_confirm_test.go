package index

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/call"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/transport"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// A published delete task does not prove an index went away: Algolia refuses to
// delete an index that is a destination of an A/B test, and that association can
// outlive the test. The destroy then reports success, Terraform drops the resource,
// and an index nothing tracks keeps running. These tests cover the confirming read
// that turns that into a failure.

// newDeleteConfirmClient answers GetSettings with 200 until the nth call, then
// 404s, so a test can say "the index disappears after this many reads". A count of
// 0 means it never disappears.
func newDeleteConfirmClient(t *testing.T, goneAfter int32) (*search.APIClient, *atomic.Int32) {
	t.Helper()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if n := calls.Add(1); goneAfter > 0 && n >= goneAfter {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"Index does not exist","status":404}`))

			return
		}

		_, _ = w.Write([]byte(`{"replicas":[]}`))
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

	return client, &calls
}

func TestConfirmIndexDeleted_ReturnsOnceTheIndexIsGone(t *testing.T) {
	client, calls := newDeleteConfirmClient(t, 1)

	if err := confirmIndexDeleted(context.Background(), client, "tf-test-gone"); err != nil {
		t.Fatalf("confirmIndexDeleted() error = %v, want nil for an index that no longer exists", err)
	}
	// The measured happy path is a single read: an index is gone the moment its
	// delete task publishes, so this must not cost a poll cycle.
	if got := calls.Load(); got != 1 {
		t.Errorf("made %d reads, want exactly 1 when the index is already gone", got)
	}
}

func TestConfirmIndexDeleted_FailsWhenTheIndexSurvives(t *testing.T) {
	client, _ := newDeleteConfirmClient(t, 0)

	err := confirmIndexDeletedWithin(context.Background(), client, "tf-test-survivor", 50*time.Millisecond)
	if err == nil {
		t.Fatal("confirmIndexDeleted() succeeded for an index that never disappeared, want an error")
	}
	if !strings.Contains(err.Error(), "tf-test-survivor") {
		t.Errorf("error does not name the index: %v", err)
	}
	if !strings.Contains(err.Error(), "50ms") {
		t.Errorf("error does not report the budget it waited: %v", err)
	}
}

// TestConfirmIndexDeleted_ToleratesATransientReadFailure covers a read that fails
// for a reason other than absence. That says nothing about whether the delete
// worked, so it must not be mistaken for the index surviving.
func TestConfirmIndexDeleted_ToleratesATransientReadFailure(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// First read fails with a server error, then the index reads as gone.
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"internal error","status":500}`))

			return
		}

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

	if err := confirmIndexDeleted(context.Background(), client, "tf-test-transient"); err != nil {
		t.Errorf("confirmIndexDeleted() error = %v, want it to keep waiting through a transient failure", err)
	}
}

func TestConfirmIndexDeleted_StopsWhenContextIsCancelled(t *testing.T) {
	client, _ := newDeleteConfirmClient(t, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := confirmIndexDeletedWithin(ctx, client, "tf-test-cancelled", 50*time.Millisecond); err == nil {
		t.Error("confirmIndexDeleted() succeeded with a cancelled context, want an error")
	}
}

// TestDeleteConfirmBudget_StaysShort pins the intent behind the constant. Its job
// is to fail fast on an index that outlived its delete task, not to absorb normal
// latency - there is none to absorb, since a deleted index reads as gone the moment
// the task publishes. A generous budget here would turn a useful failure into a
// long hang on every destroy that hits it.
func TestDeleteConfirmBudget_StaysShort(t *testing.T) {
	if deleteConfirmBudget > time.Minute {
		t.Errorf("deleteConfirmBudget = %s, want something a person will wait out during a destroy", deleteConfirmBudget)
	}
	if deleteConfirmBudget < 5*time.Second {
		t.Errorf("deleteConfirmBudget = %s, want enough room for a slow read to retry", deleteConfirmBudget)
	}
}

func TestDeleteNotConfirmedDetail_IsActionable(t *testing.T) {
	detail := deleteNotConfirmedDetail("tf-test-products", errNotConfirmed{})

	for _, want := range []string{
		"tf-test-products",
		// The known cause, so the operator has somewhere to look.
		"A/B test",
		// And what it means for state, since the resource is deliberately kept.
		"left in Terraform state",
	} {
		if !strings.Contains(detail, want) {
			t.Errorf("detail does not mention %q:\n%s", want, detail)
		}
	}
}

type errNotConfirmed struct{}

func (errNotConfirmed) Error() string {
	return "deletion of index tf-test-products did not complete within 20s"
}

// TestIndexResourceDelete_failsWhenTheIndexSurvives drives Delete itself rather
// than the helper, so that removing the confirming read from the resource fails
// here. Without it the destroy reports success and Terraform drops a resource
// whose index is still running.
func TestIndexResourceDelete_failsWhenTheIndexSurvives(t *testing.T) {
	ctx := context.Background()

	// Accept the delete, publish its task, and keep answering settings reads: the
	// shape of a delete that took effect on paper only.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodDelete:
			_, _ = w.Write([]byte(`{"taskID":1,"deletedAt":"2026-07-31T00:00:00.000Z"}`))
		case strings.Contains(r.URL.Path, "/task/"):
			_, _ = w.Write([]byte(`{"status":"published","pendingTask":false}`))
		default:
			_, _ = w.Write([]byte(`{"replicas":[]}`))
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

	// Keep the wait short: this asserts the give-up path, not how long it waits.
	original := deleteConfirmBudget
	deleteConfirmBudget = 50 * time.Millisecond
	t.Cleanup(func() { deleteConfirmBudget = original })

	r := &indexResource{client: client}

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	indexSchema := schemaResp.Schema

	state := tfsdk.State{Schema: indexSchema, Raw: tftypes.NewValue(indexSchema.Type().TerraformType(ctx), nil)}
	if diags := state.Set(ctx, deletedIndexModel()); diags.HasError() {
		t.Fatalf("seeding state: %v", diags)
	}

	resp := &resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Delete() reported success for an index that still exists, want an error")
	}
	if got, want := resp.Diagnostics.Errors()[0].Summary(), "Index still exists after deletion"; got != want {
		t.Errorf("error summary = %q, want %q", got, want)
	}
}
