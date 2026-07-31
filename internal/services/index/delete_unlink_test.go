package index

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/call"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/transport"
)

// deleteFake models the part of the Search API that deleting a replica touches:
// the replica's own settings, which name its primary; the primary's replicas list;
// deleteIndex, which Algolia refuses with 403 for as long as the primary lists the
// index; and task status.
//
// It counts writes to the primary separately from delete attempts, because the
// distinction is the point. Unlinking means writing the primary's settings, and a
// settings write is what creates an index in Algolia, so a write that happens when
// the delete would have succeeded anyway is what recreates a primary another
// resource has just deleted.
type deleteFake struct {
	mu sync.Mutex

	replicaName string
	primaryName string

	// primaryReplicas is what the primary currently lists. The delete is refused
	// while it contains replicaName.
	primaryReplicas []string
	// refusalsBeforePrimaryDrops makes the primary let go of the replica on its own
	// after that many refusals, standing in for a primary deleted concurrently.
	refusalsBeforePrimaryDrops int

	deleteAttempts int
	primaryWrites  int
	replicaGone    bool
}

func newDeleteFake(t *testing.T, fake *deleteFake) *search.APIClient {
	t.Helper()

	// Retries are what this fake exists to exercise, so remove the wait between them.
	originalInterval := replicaDeleteInterval
	replicaDeleteInterval = time.Millisecond
	t.Cleanup(func() { replicaDeleteInterval = originalInterval })

	server := httptest.NewServer(http.HandlerFunc(fake.handle))
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

func (f *deleteFake) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	trimmed := strings.TrimPrefix(r.URL.Path, "/1/indexes/")

	switch {
	case strings.Contains(trimmed, "/task/"):
		_, _ = fmt.Fprint(w, `{"status":"published","pendingTask":false}`)

	case r.Method == http.MethodDelete && trimmed == f.replicaName:
		f.deleteAttempts++
		if f.replicaGone {
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, `{"message":"Index does not exist","status":404}`)

			return
		}
		if f.listsReplica() {
			if f.refusalsBeforePrimaryDrops > 0 && f.deleteAttempts >= f.refusalsBeforePrimaryDrops {
				// The primary has gone away of its own accord, which is what makes the
				// replica deletable without anything having unlinked it.
				f.primaryReplicas = nil
			} else {
				w.WriteHeader(http.StatusForbidden)
				_, _ = fmt.Fprint(w, `{"message":"cannot apply the deleteIndex operation on a replica index","status":403}`)

				return
			}
		}
		f.replicaGone = true
		_, _ = fmt.Fprint(w, `{"deletedAt":"2026-01-01T00:00:00.000Z","taskID":1}`)

	case r.Method == http.MethodGet && trimmed == f.replicaName+"/settings":
		if f.replicaGone {
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, `{"message":"Index does not exist","status":404}`)

			return
		}
		if !f.listsReplica() {
			_, _ = fmt.Fprint(w, `{}`)

			return
		}
		_, _ = fmt.Fprintf(w, `{"primary":%q}`, f.primaryName)

	case r.Method == http.MethodGet && trimmed == f.primaryName+"/settings":
		body, err := json.Marshal(map[string]any{"replicas": f.primaryReplicas})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
		_, _ = w.Write(body)

	case r.Method == http.MethodPut && trimmed == f.primaryName+"/settings":
		f.primaryWrites++
		var written struct {
			Replicas []string `json:"replicas"`
		}
		if err := json.NewDecoder(r.Body).Decode(&written); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}
		f.primaryReplicas = written.Replicas
		_, _ = fmt.Fprint(w, `{"updatedAt":"2026-01-01T00:00:00.000Z","taskID":2}`)

	default:
		http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusNotImplemented)
	}
}

func (f *deleteFake) listsReplica() bool {
	for _, entry := range f.primaryReplicas {
		if entry == f.replicaName || entry == virtualReplicaName(f.replicaName) {
			return true
		}
	}

	return false
}

func (f *deleteFake) counts() (deletes, primaryWrites int, gone bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.deleteAttempts, f.primaryWrites, f.replicaGone
}

// TestDeleteIndexWithUnlink_leavesThePrimaryAloneWhenTheDeleteWorks is the ordinary
// case: an index that is nobody's replica is deleted without its primary - or any
// other index - being written to.
func TestDeleteIndexWithUnlink_leavesThePrimaryAloneWhenTheDeleteWorks(t *testing.T) {
	fake := &deleteFake{replicaName: "products_price_asc", primaryName: "products"}
	client := newDeleteFake(t, fake)

	if err := deleteIndexWithUnlink(context.Background(), client, "products_price_asc"); err != nil {
		t.Fatalf("deleteIndexWithUnlink() = %v, want nil", err)
	}

	deletes, writes, gone := fake.counts()
	if !gone {
		t.Error("the index was not deleted")
	}
	if deletes != 1 || writes != 0 {
		t.Errorf("delete attempts = %d, primary writes = %d; want 1 and 0", deletes, writes)
	}
}

// TestDeleteIndexWithUnlink_retriesRatherThanWritingWhenThePrimaryGoesAway is the
// regression test for a primary that gets recreated by the very unlink meant to
// help. Destroying a primary and its replica together refuses the replica's delete
// only until the primary is gone, so retrying clears it with no write at all - and
// a write here would recreate the primary as an empty index nothing owns.
func TestDeleteIndexWithUnlink_retriesRatherThanWritingWhenThePrimaryGoesAway(t *testing.T) {
	fake := &deleteFake{
		replicaName:                "products_price_asc",
		primaryName:                "products",
		primaryReplicas:            []string{"products_price_asc"},
		refusalsBeforePrimaryDrops: 2,
	}
	client := newDeleteFake(t, fake)

	if err := deleteIndexWithUnlink(context.Background(), client, "products_price_asc"); err != nil {
		t.Fatalf("deleteIndexWithUnlink() = %v, want nil", err)
	}

	deletes, writes, gone := fake.counts()
	if !gone {
		t.Error("the index was not deleted")
	}
	if writes != 0 {
		t.Errorf("primary writes = %d, want 0: a retry cleared the refusal, so nothing needed unlinking", writes)
	}
	if deletes < 2 {
		t.Errorf("delete attempts = %d, want at least 2: the first was refused", deletes)
	}
}

// TestDeleteIndexWithUnlink_unlinksWhenTheRefusalPersists covers the other half: a
// primary that keeps listing the replica is not going away, so the entry has to be
// removed before Algolia will delete the index.
func TestDeleteIndexWithUnlink_unlinksWhenTheRefusalPersists(t *testing.T) {
	fake := &deleteFake{
		replicaName:     "products_price_asc",
		primaryName:     "products",
		primaryReplicas: []string{"products_price_asc", "virtual(products_cheapest)"},
	}
	client := newDeleteFake(t, fake)

	if err := deleteIndexWithUnlink(context.Background(), client, "products_price_asc"); err != nil {
		t.Fatalf("deleteIndexWithUnlink() = %v, want nil", err)
	}

	deletes, writes, gone := fake.counts()
	if !gone {
		t.Error("the index was not deleted")
	}
	if writes != 1 {
		t.Errorf("primary writes = %d, want exactly 1 unlink", writes)
	}
	if deletes != replicaDeleteAttempts+1 {
		t.Errorf("delete attempts = %d, want %d: every retry then one more after unlinking",
			deletes, replicaDeleteAttempts+1)
	}

	// The unlink must take out this replica's entry and nothing else.
	fake.mu.Lock()
	remaining := strings.Join(fake.primaryReplicas, ",")
	fake.mu.Unlock()
	if remaining != "virtual(products_cheapest)" {
		t.Errorf("primary replicas = %q, want the other replica untouched", remaining)
	}
}

// TestDeleteIndexWithUnlink_toleratesAnAbsentIndex keeps a repeated destroy a no-op,
// and asserts it does so without writing to a primary it can no longer identify.
func TestDeleteIndexWithUnlink_toleratesAnAbsentIndex(t *testing.T) {
	fake := &deleteFake{replicaName: "products_price_asc", primaryName: "products", replicaGone: true}
	client := newDeleteFake(t, fake)

	if err := deleteIndexWithUnlink(context.Background(), client, "products_price_asc"); err != nil {
		t.Fatalf("deleteIndexWithUnlink() = %v, want nil for an index that has already gone", err)
	}

	_, writes, _ := fake.counts()
	if writes != 0 {
		t.Errorf("primary writes = %d, want 0", writes)
	}
}
