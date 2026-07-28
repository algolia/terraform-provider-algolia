package algoliaerr

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	abtesting "github.com/algolia/algoliasearch-client-go/v4/algolia/abtesting"
	abtestingV3 "github.com/algolia/algoliasearch-client-go/v4/algolia/abtesting-v3"
	agentStudio "github.com/algolia/algoliasearch-client-go/v4/algolia/agent-studio"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/analytics"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/composition"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/insights"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/monitoring"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/personalization"
	suggestions "github.com/algolia/algoliasearch-client-go/v4/algolia/query-suggestions"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/recommend"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
)

// apiErrorsWithStatus builds one error per supported Algolia client type, all
// carrying the given status. Adding a case pair to Status without adding an entry
// here leaves it untested, so keep the two lists in sync.
func apiErrorsWithStatus(status int) map[string]error {
	msg := fmt.Sprintf("status %d", status)

	return map[string]error{
		"search":          &search.APIError{Message: msg, Status: status},
		"recommend":       &recommend.APIError{Message: msg, Status: status},
		"ingestion":       &ingestion.APIError{Message: msg, Status: status},
		"composition":     &composition.APIError{Message: msg, Status: status},
		"personalization": &personalization.APIError{Message: msg, Status: status},
		"suggestions":     &suggestions.APIError{Message: msg, Status: status},
		"abtesting":       &abtesting.APIError{Message: msg, Status: status},
		"abtestingV3":     &abtestingV3.APIError{Message: msg, Status: status},
		"agentStudio":     &agentStudio.APIError{Message: msg, Status: status},
		"analytics":       &analytics.APIError{Message: msg, Status: status},
		"insights":        &insights.APIError{Message: msg, Status: status},
		"monitoring":      &monitoring.APIError{Message: msg, Status: status},
	}
}

// apiErrorValuesWithStatus mirrors apiErrorsWithStatus with the value rather than
// the pointer form of each type. The clients return pointers, but Error() is
// declared on the value receiver, so a bare value is a valid error too and must
// not be silently classified as "not an Algolia error".
func apiErrorValuesWithStatus(status int) map[string]error {
	msg := fmt.Sprintf("status %d", status)

	return map[string]error{
		"search":          search.APIError{Message: msg, Status: status},
		"recommend":       recommend.APIError{Message: msg, Status: status},
		"ingestion":       ingestion.APIError{Message: msg, Status: status},
		"composition":     composition.APIError{Message: msg, Status: status},
		"personalization": personalization.APIError{Message: msg, Status: status},
		"suggestions":     suggestions.APIError{Message: msg, Status: status},
		"abtesting":       abtesting.APIError{Message: msg, Status: status},
		"abtestingV3":     abtestingV3.APIError{Message: msg, Status: status},
		"agentStudio":     agentStudio.APIError{Message: msg, Status: status},
		"analytics":       analytics.APIError{Message: msg, Status: status},
		"insights":        insights.APIError{Message: msg, Status: status},
		"monitoring":      monitoring.APIError{Message: msg, Status: status},
	}
}

// TestSupportedSurfaceCount pins the number of API surfaces the fixtures cover,
// so that adding one to Status without a fixture (or the reverse) is visible.
func TestSupportedSurfaceCount(t *testing.T) {
	const wantSurfaces = 12

	if got := len(apiErrorsWithStatus(http.StatusNotFound)); got != wantSurfaces {
		t.Errorf("fixtures cover %d API surfaces, want %d", got, wantSurfaces)
	}
}

func TestStatus(t *testing.T) {
	for _, status := range []int{
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
	} {
		for name, err := range apiErrorsWithStatus(status) {
			t.Run(fmt.Sprintf("%s/%d", name, status), func(t *testing.T) {
				got, ok := Status(err)
				if !ok {
					t.Fatalf("Status(%T) reported no status", err)
				}
				if got != status {
					t.Errorf("Status() = %d, want %d", got, status)
				}
			})

			t.Run(fmt.Sprintf("%s/%d/wrapped", name, status), func(t *testing.T) {
				wrapped := fmt.Errorf("could not read resource: %w", err)
				got, ok := Status(wrapped)
				if !ok {
					t.Fatalf("Status(wrapped %T) reported no status", err)
				}
				if got != status {
					t.Errorf("Status() = %d, want %d", got, status)
				}
			})

			t.Run(fmt.Sprintf("%s/%d/doubleWrapped", name, status), func(t *testing.T) {
				wrapped := fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", err))
				got, ok := Status(wrapped)
				if !ok {
					t.Fatalf("Status(double-wrapped %T) reported no status", err)
				}
				if got != status {
					t.Errorf("Status() = %d, want %d", got, status)
				}
			})
		}
	}
}

// TestStatusJoined covers the multi-error Unwrap shape produced by errors.Join.
func TestStatusJoined(t *testing.T) {
	apiErr := &search.APIError{Message: "index does not exist", Status: http.StatusNotFound}
	joined := errors.Join(errors.New("first attempt failed"), apiErr)

	got, ok := Status(joined)
	if !ok {
		t.Fatal("Status(joined) reported no status")
	}
	if got != http.StatusNotFound {
		t.Errorf("Status(joined) = %d, want %d", got, http.StatusNotFound)
	}
	if !IsNotFound(joined) {
		t.Error("IsNotFound(joined) = false, want true")
	}
	if !IsNotFound(fmt.Errorf("wrapping a join: %w", joined)) {
		t.Error("IsNotFound(wrapped join) = false, want true")
	}
}

// TestStatusValueReceiver covers the non-pointer form of every supported type.
func TestStatusValueReceiver(t *testing.T) {
	for name, err := range apiErrorValuesWithStatus(http.StatusNotFound) {
		t.Run(name, func(t *testing.T) {
			got, ok := Status(err)
			if !ok {
				t.Fatalf("Status(%T value) reported no status", err)
			}
			if got != http.StatusNotFound {
				t.Errorf("Status() = %d, want %d", got, http.StatusNotFound)
			}
			if !IsNotFound(err) {
				t.Errorf("IsNotFound(%T value) = false, want true", err)
			}
			if !IsNotFound(fmt.Errorf("wrapped: %w", err)) {
				t.Errorf("IsNotFound(wrapped %T value) = false, want true", err)
			}
		})
	}

	for name, err := range apiErrorValuesWithStatus(http.StatusServiceUnavailable) {
		t.Run(name+"/retryable", func(t *testing.T) {
			if !IsRetryable(err) {
				t.Errorf("IsRetryable(%T value with status 503) = false, want true", err)
			}
			if IsNotFound(err) {
				t.Errorf("IsNotFound(%T value with status 503) = true, want false", err)
			}
		})
	}
}

func TestStatusUnsupportedErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "nil", err: nil},
		{name: "sentinel", err: errors.New("boom")},
		{name: "wrapped sentinel", err: fmt.Errorf("outer: %w", errors.New("boom"))},
		{name: "nil typed pointer", err: (*search.APIError)(nil)},
		{name: "unrelated struct error", err: &url404{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, ok := Status(tt.err); ok {
				t.Errorf("Status() = (%d, true), want (0, false)", got)
			}
			if IsNotFound(tt.err) {
				t.Error("IsNotFound() = true, want false")
			}
			if IsRetryable(tt.err) {
				t.Error("IsRetryable() = true, want false")
			}
		})
	}
}

// url404 is an unrelated error type that mentions a 404 without being an Algolia
// API error, guarding against a message-sniffing implementation.
type url404 struct{}

func (*url404) Error() string { return "GET /thing: 404 Not Found" }

func TestIsNotFound(t *testing.T) {
	for name, err := range apiErrorsWithStatus(http.StatusNotFound) {
		t.Run(name+"/404", func(t *testing.T) {
			if !IsNotFound(err) {
				t.Errorf("IsNotFound(%T) = false, want true", err)
			}
			if !IsNotFound(fmt.Errorf("wrapped: %w", err)) {
				t.Errorf("IsNotFound(wrapped %T) = false, want true", err)
			}
		})
	}

	for _, status := range []int{
		http.StatusOK,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusConflict,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
	} {
		for name, err := range apiErrorsWithStatus(status) {
			t.Run(fmt.Sprintf("%s/%d", name, status), func(t *testing.T) {
				if IsNotFound(err) {
					t.Errorf("IsNotFound(%T with status %d) = true, want false", err, status)
				}
			})
		}
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		status int
		want   bool
	}{
		{status: http.StatusBadRequest, want: false},
		{status: http.StatusUnauthorized, want: false},
		{status: http.StatusForbidden, want: false},
		{status: http.StatusNotFound, want: false},
		{status: http.StatusConflict, want: false},
		{status: http.StatusPaymentRequired, want: false},
		{status: http.StatusTooManyRequests, want: true},
		{status: http.StatusInternalServerError, want: true},
		{status: http.StatusBadGateway, want: true},
		{status: http.StatusServiceUnavailable, want: true},
		{status: http.StatusGatewayTimeout, want: true},
		{status: 599, want: true},
		{status: 600, want: false},
	}

	for _, tt := range tests {
		for name, err := range apiErrorsWithStatus(tt.status) {
			t.Run(fmt.Sprintf("%s/%d", name, tt.status), func(t *testing.T) {
				if got := IsRetryable(err); got != tt.want {
					t.Errorf("IsRetryable(%T with status %d) = %t, want %t", err, tt.status, got, tt.want)
				}
				if got := IsRetryable(fmt.Errorf("wrapped: %w", err)); got != tt.want {
					t.Errorf("IsRetryable(wrapped %T with status %d) = %t, want %t", err, tt.status, got, tt.want)
				}
			})
		}
	}
}

// TestCrossPackageIsolation checks that the generated APIError types really are
// nominally distinct, so that the per-surface cases in Status are load-bearing
// rather than a structural match that any one of them would satisfy.
func TestCrossPackageIsolation(t *testing.T) {
	err := &search.APIError{Message: "index does not exist", Status: http.StatusNotFound}

	var recommendErr *recommend.APIError
	if errors.As(err, &recommendErr) {
		t.Fatal("errors.As matched *recommend.APIError against a *search.APIError; the types are no longer distinct")
	}

	if !IsNotFound(err) {
		t.Error("IsNotFound(*search.APIError 404) = false, want true")
	}
}
