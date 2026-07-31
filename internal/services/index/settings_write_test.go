package index

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/call"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/transport"
)

// settingsWriteServer fakes the two calls a settings wait makes. It decides which
// task IDs ever publish, which is what lets a test reproduce the case that matters:
// a task ID that stays notPublished forever because the index's queue restarted
// under it, while a freshly issued write publishes normally.
type settingsWriteServer struct {
	mu                sync.Mutex
	writes            int
	nextTaskID        int64
	published         map[int64]bool
	reissuedPublishes bool
}

func newSettingsWriteServer(t *testing.T, initialTaskID int64, initialPublishes, reissuedPublishes bool) (*search.APIClient, *settingsWriteServer) {
	t.Helper()

	fake := &settingsWriteServer{
		nextTaskID:        initialTaskID,
		published:         map[int64]bool{},
		reissuedPublishes: reissuedPublishes,
	}
	if initialPublishes {
		fake.published[initialTaskID] = true
	}

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

	return client, fake
}

func (s *settingsWriteServer) handle(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/settings"):
		s.writes++
		s.nextTaskID++
		if s.reissuedPublishes {
			s.published[s.nextTaskID] = true
		}
		_, _ = fmt.Fprintf(w, `{"updatedAt":"2026-01-01T00:00:00.000Z","taskID":%d}`, s.nextTaskID)
	case strings.Contains(r.URL.Path, "/task/"):
		taskID, err := strconv.ParseInt(path.Base(r.URL.Path), 10, 64)
		if err != nil {
			http.Error(w, "bad task id", http.StatusBadRequest)

			return
		}
		status := "notPublished"
		if s.published[taskID] {
			status = "published"
		}
		_, _ = fmt.Fprintf(w, `{"status":%q,"pendingTask":false}`, status)
	default:
		http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusNotImplemented)
	}
}

func (s *settingsWriteServer) writeCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.writes
}

// shortenWriteReissueGrace removes the wait before a write is re-sent so a test
// does not have to sit through it, restoring the real value afterwards.
func shortenWriteReissueGrace(t *testing.T) {
	t.Helper()

	original := writeReissueGrace
	writeReissueGrace = 0
	t.Cleanup(func() { writeReissueGrace = original })
}

// TestWaitForSettingsWrite_publishedTaskIsNotResent is the case that must stay
// cheap: a healthy queue publishes the task, so the wait returns without touching
// the index again.
func TestWaitForSettingsWrite_publishedTaskIsNotResent(t *testing.T) {
	client, fake := newSettingsWriteServer(t, 1000, true, false)

	err := waitForSettingsWrite(context.Background(), client, "products", search.NewIndexSettings(), 1000)
	if err != nil {
		t.Fatalf("waitForSettingsWrite() = %v, want nil", err)
	}
	if got := fake.writeCount(); got != 0 {
		t.Errorf("settings written %d times, want 0: a task that publishes needs no re-send", got)
	}
}

// TestWaitForSettingsWrite_resendsAVoidTask covers the hang: Algolia restarts an
// index's task queue when another write turns the index into a replica, so the task
// ID from before never publishes even though the write itself landed. Waiting on it
// alone burns the full budget and fails a create that succeeded.
func TestWaitForSettingsWrite_resendsAVoidTask(t *testing.T) {
	shortenWriteReissueGrace(t)

	client, fake := newSettingsWriteServer(t, 1000, false, true)

	err := waitForSettingsWrite(context.Background(), client, "products", search.NewIndexSettings(), 1000)
	if err != nil {
		t.Fatalf("waitForSettingsWrite() = %v, want nil once the re-sent write publishes", err)
	}
	if got := fake.writeCount(); got != 1 {
		t.Errorf("settings written %d times, want exactly 1 re-send", got)
	}
}

// TestWaitForSettingsWrite_stopsResendingAtTheLimit guards the other direction: a
// wait that never progresses must not keep writing to the index for the rest of its
// budget. The context stands in for the deadline so the test does not wait one out.
func TestWaitForSettingsWrite_stopsResendingAtTheLimit(t *testing.T) {
	shortenWriteReissueGrace(t)

	client, fake := newSettingsWriteServer(t, 1000, false, false)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- waitForSettingsWrite(ctx, client, "products", search.NewIndexSettings(), 1000)
	}()

	// Every poll re-sends until the limit is reached, so waiting for one more write
	// than the limit would hang: wait for the limit itself, then stop the wait.
	waitForWrites(t, fake, writeReissueLimit)
	cancel()

	if err := <-done; err == nil {
		t.Fatal("waitForSettingsWrite() = nil, want the cancellation error")
	}
	if got := fake.writeCount(); got != writeReissueLimit {
		t.Errorf("settings written %d times, want %d: the limit caps re-sends", got, writeReissueLimit)
	}
}

// waitForWrites blocks until the fake has recorded want writes, failing rather than
// hanging if they never arrive.
func waitForWrites(t *testing.T, fake *settingsWriteServer, want int) {
	t.Helper()

	for range 200 {
		if fake.writeCount() >= want {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("only %d of %d re-sends happened", fake.writeCount(), want)
}
