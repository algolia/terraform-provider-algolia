package crawler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClient_hasRequestTimeout(t *testing.T) {
	client := NewClient("user", "key")

	if client.http.Timeout != defaultTimeout {
		t.Errorf("http.Client.Timeout = %v, want %v (an unbounded client blocks the apply on a hung endpoint)",
			client.http.Timeout, defaultTimeout)
	}
}

func TestDo_rejectsOversizedResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// One byte past the cap is enough: the read is bounded by a LimitReader,
		// so the handler is never asked for more than that.
		_, _ = w.Write([]byte(`{"name":"` + strings.Repeat("a", maxResponseBytes) + `"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server)

	_, err := client.GetCrawler(context.Background(), "crawler-1")
	if err == nil {
		t.Fatal("GetCrawler() error = nil, want an oversized-body error")
	}
	if !strings.Contains(err.Error(), "exceeds the") {
		t.Errorf("GetCrawler() error = %v, want an oversized-body error", err)
	}
}

func TestAPIError_bodyIsTruncatedAndSanitized(t *testing.T) {
	// The body reaches Terraform diagnostics verbatim: it must be bounded and must
	// not carry control characters (here an ANSI escape sequence).
	hostile := "\x1b[31mboom\x1b[0m\n\tdetail\n" + strings.Repeat("x", maxErrorBodyBytes)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(hostile))
	}))
	defer server.Close()

	client := newTestClient(t, server)

	err := client.DeleteCrawler(context.Background(), "crawler-1")
	if err == nil {
		t.Fatal("DeleteCrawler() error = nil, want an *APIError")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}

	if len(apiErr.Body) > maxErrorBodyBytes+len(" [truncated]") {
		t.Errorf("APIError.Body length = %d, want at most %d", len(apiErr.Body), maxErrorBodyBytes+len(" [truncated]"))
	}
	if strings.ContainsAny(apiErr.Body, "\x1b\n\r\t") {
		t.Errorf("APIError.Body retains control characters: %q", apiErr.Body)
	}
	if !strings.HasSuffix(apiErr.Body, "[truncated]") {
		t.Errorf("APIError.Body = %q, want a truncation marker", apiErr.Body)
	}
	if !strings.Contains(apiErr.Body, "boom") || !strings.Contains(apiErr.Body, "detail") {
		t.Errorf("APIError.Body = %q, want the printable message preserved", apiErr.Body)
	}
}

func TestSanitizeErrorBody(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "json passes through", in: `{"error":"not found"}`, want: `{"error":"not found"}`},
		{name: "whitespace collapsed", in: "a\n\tb   c", want: "a b c"},
		{name: "ansi escape stripped", in: "\x1b[31mred\x1b[0m", want: "[31mred [0m"},
		{name: "invalid utf8 replaced", in: "ok\xff", want: "ok"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeErrorBody([]byte(tt.in)); got != tt.want {
				t.Errorf("sanitizeErrorBody(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
