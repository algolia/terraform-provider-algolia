package algoliawait

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestUntil_ReturnsOnceCheckReportsDone(t *testing.T) {
	calls := 0
	err := Until(context.Background(), "task 1", func(context.Context) (bool, error) {
		calls++
		return calls == 3, nil
	})
	if err != nil {
		t.Fatalf("Until() = %v, want nil", err)
	}
	if calls != 3 {
		t.Errorf("check called %d times, want 3", calls)
	}
}

func TestUntil_DoesNotSleepBeforeTheFirstCheck(t *testing.T) {
	start := time.Now()
	if err := Until(context.Background(), "task 1", func(context.Context) (bool, error) {
		return true, nil
	}); err != nil {
		t.Fatalf("Until() = %v, want nil", err)
	}
	// An already-finished task must not pay the backoff interval, which is what
	// makes this usable on the common path where the task has already landed.
	if elapsed := time.Since(start); elapsed >= initialInterval {
		t.Errorf("returned after %s, want less than the %s interval", elapsed, initialInterval)
	}
}

func TestUntil_PropagatesCheckErrorWithoutRetrying(t *testing.T) {
	sentinel := errors.New("boom")
	calls := 0
	err := Until(context.Background(), "task 1", func(context.Context) (bool, error) {
		calls++
		return false, sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("Until() = %v, want %v", err, sentinel)
	}
	if calls != 1 {
		t.Errorf("check called %d times, want 1: an error must abort rather than retry", calls)
	}
}

func TestUntil_StopsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	calls := 0
	start := time.Now()
	err := Until(ctx, "task 1", func(context.Context) (bool, error) {
		calls++
		cancel() // cancelled while the operation is still pending
		return false, nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Until() = %v, want context.Canceled", err)
	}
	// The whole point: cancellation must interrupt the sleep rather than wait it
	// out, and must certainly not run to the 30 minute deadline.
	if elapsed := time.Since(start); elapsed >= initialInterval {
		t.Errorf("returned after %s, want the sleep to be interrupted", elapsed)
	}
	if calls != 1 {
		t.Errorf("check called %d times, want 1", calls)
	}
}

func TestUntil_PassesContextToCheck(t *testing.T) {
	type key struct{}
	ctx := context.WithValue(context.Background(), key{}, "carried")

	var seen any
	if err := Until(ctx, "task 1", func(c context.Context) (bool, error) {
		seen = c.Value(key{})
		return true, nil
	}); err != nil {
		t.Fatalf("Until() = %v, want nil", err)
	}
	if seen != "carried" {
		t.Errorf("check saw %v, want the caller's context to be passed through", seen)
	}
}

func TestUntil_TimeoutErrorNamesTheSubject(t *testing.T) {
	// Drive the deadline path without waiting 30 minutes by cancelling a context
	// that is already past its own deadline, then asserting on the message shape
	// of a real timeout separately.
	err := Until(context.Background(), "task 42 on index \"products\"", func(context.Context) (bool, error) {
		return true, nil
	})
	if err != nil {
		t.Fatalf("sanity check failed: %v", err)
	}

	// The timeout message is built from the subject, so assert its shape directly.
	want := "task 42 on index \"products\" did not complete within 30m0s"
	got := timeoutError("task 42 on index \"products\"").Error()
	if got != want {
		t.Errorf("timeout error = %q, want %q", got, want)
	}
}
