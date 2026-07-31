// Package algoliawait polls an Algolia asynchronous operation until it finishes.
//
// Almost every write in the Algolia API is queued rather than applied: the
// response carries a task ID and the operation lands some time later. A resource
// that returns before the task has run reports success for work that has not
// happened yet, and the next call can then observe stale state - an index that
// still exists, or an A/B test that still holds a lock on one.
//
// Each API surface checks task completion differently, so the part worth sharing
// is the loop rather than the check: a bounded deadline, exponential backoff, and
// a sleep that a cancelled context can interrupt. That loop was hand-copied into
// six packages, which is how one of them ended up with a bare time.Sleep that
// made a 30-minute wait uncancellable. It lives here once instead.
package algoliawait

import (
	"context"
	"fmt"
	"time"
)

const (
	// Timeout bounds a single wait. Algolia's own indexing can legitimately take
	// minutes on a large index, so this is generous by design; it exists to stop a
	// wait hanging forever, not to express an expected duration.
	Timeout = 30 * time.Minute

	initialInterval = 2 * time.Second
	intervalStep    = time.Second
	maxInterval     = 10 * time.Second
)

// Until polls check until it reports done, the deadline passes, or ctx is
// cancelled, whichever happens first.
//
// check reports (true, nil) when the operation has finished, (false, nil) when it
// is still pending, and a non-nil error to abort. An error from check is returned
// as-is rather than retried: callers that need a transient status tolerated
// should absorb it in their own check and keep reporting "pending".
//
// subject names what is being waited on and is used only to build the timeout
// error, so it should read naturally in `<subject> did not complete within 30m0s`.
func Until(ctx context.Context, subject string, check func(context.Context) (bool, error)) error {
	return Within(ctx, subject, Timeout, check)
}

// Within is Until with a caller-chosen budget, for a wait whose expected duration
// is known to be short and where exceeding it is itself the news.
//
// Timeout suits waiting on Algolia to finish work that can legitimately take
// minutes. It is the wrong bound for confirming that work already reported as
// finished actually took effect: there the answer is normally immediate, so a
// generous deadline turns a fast, useful failure into a very long hang.
func Within(ctx context.Context, subject string, budget time.Duration, check func(context.Context) (bool, error)) error {
	deadline := time.Now().Add(budget)
	interval := initialInterval

	for time.Now().Before(deadline) {
		done, err := check(ctx)
		if err != nil {
			return err
		}
		if done {
			return nil
		}

		// Sleep interruptibly. A bare time.Sleep here would make the whole budget
		// uncancellable, so Ctrl-C could not stop a plan that was polling.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}

		if interval < maxInterval {
			interval += intervalStep
		}
	}

	return timeoutError(subject, budget)
}

// timeoutError builds the error returned when a wait exhausts its deadline. It is
// separate so a test can assert the message without waiting out the deadline.
func timeoutError(subject string, budget time.Duration) error {
	return fmt.Errorf("%s did not complete within %s", subject, budget)
}
