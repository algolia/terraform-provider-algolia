package algoliaerr

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
)

// ingestionErrorFromBody builds the error the Ingestion client produces for a
// failure response, decoding a real response body so the test is pinned to the
// API's shape rather than to a hand-built map. This body is the one the API
// returns for a no-code transformation whose steps are missing required fields.
func ingestionErrorFromBody(t *testing.T, status int, body string) error {
	t.Helper()

	var properties map[string]any
	if err := json.Unmarshal([]byte(body), &properties); err != nil {
		t.Fatalf("test body is not valid JSON: %v", err)
	}

	message, _ := properties["message"].(string)
	// The client keeps everything but `message` as additional properties.
	delete(properties, "message")

	return &ingestion.APIError{Message: message, Status: status, AdditionalProperties: properties}
}

const invalidPayloadBody = `{
  "message": "Invalid payload, see error.details",
  "status": 400,
  "error": {
    "code": "invalid_payload",
    "details": [
      {"label": "input.steps[0].name", "message": "'name' is required"},
      {"label": "input.steps[0].id", "message": "'id' is required"}
    ]
  }
}`

func TestExplain_AddsAPIFieldDetails(t *testing.T) {
	err := ingestionErrorFromBody(t, 400, invalidPayloadBody)

	got := Explain(err)

	// The summary the client already produced must survive: it carries the status.
	if !strings.Contains(got, "Invalid payload, see error.details") {
		t.Errorf("explanation dropped the original message:\n%s", got)
	}
	for _, want := range []string{
		"input.steps[0].id: 'id' is required",
		"input.steps[0].name: 'name' is required",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("explanation does not mention %q:\n%s", want, got)
		}
	}

	// Sorted, so two runs of the same failure read identically.
	if idIndex, nameIndex := strings.Index(got, "[0].id"), strings.Index(got, "[0].name"); idIndex > nameIndex {
		t.Errorf("details are not in a stable order:\n%s", got)
	}
}

func TestExplain_SingleDetailReadsInline(t *testing.T) {
	err := ingestionErrorFromBody(t, 400, `{
	  "message": "Invalid payload, see error.details",
	  "status": 400,
	  "error": {"code": "invalid_payload", "details": [
	    {"label": "policies.criticalThreshold", "message": "'criticalThreshold' must be lower or equal to '10'"}
	  ]}
	}`)

	got := Explain(err)

	want := "(policies.criticalThreshold: 'criticalThreshold' must be lower or equal to '10')"
	if !strings.Contains(got, want) {
		t.Errorf("Explain() = %q, want it to contain %q", got, want)
	}
	if strings.Contains(got, "\n") {
		t.Errorf("a single detail should stay on one line:\n%s", got)
	}
}

func TestExplain_FallsBackToErrorText(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{
			// The common case: an error with a plain message and nothing structured.
			name: "api error with no details",
			err:  &ingestion.APIError{Message: "Index does not exist", Status: 404},
		},
		{
			// A different API surface, which does not use this shape at all.
			name: "search api error",
			err:  &search.APIError{Message: "Invalid Application-ID or API key", Status: 403},
		},
		{
			name: "not an algolia error",
			err:  errors.New("dial tcp: connection refused"),
		},
		{
			name: "wrapped plain error",
			err:  fmt.Errorf("while creating task: %w", errors.New("context deadline exceeded")),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got, want := Explain(tc.err), tc.err.Error(); got != want {
				t.Errorf("Explain() = %q, want exactly the error text %q", got, want)
			}
		})
	}
}

func TestExplain_NilIsEmpty(t *testing.T) {
	if got := Explain(nil); got != "" {
		t.Errorf("Explain(nil) = %q, want empty", got)
	}
}

// TestExplain_ToleratesUnexpectedShapes covers the defensive paths. These values
// are decoded from a failure response, so anything can arrive; a malformed detail
// must cost the explanation nothing rather than panic or render half a sentence.
func TestExplain_ToleratesUnexpectedShapes(t *testing.T) {
	cases := []struct {
		name       string
		properties map[string]any
		want       string // "" means: fall back to the message alone
	}{
		{
			name:       "error is not an object",
			properties: map[string]any{"error": "invalid_payload"},
		},
		{
			name:       "details is not a list",
			properties: map[string]any{"error": map[string]any{"details": "nope"}},
		},
		{
			name:       "details entries are not objects",
			properties: map[string]any{"error": map[string]any{"details": []any{"nope", 42}}},
		},
		{
			name:       "detail fields are not strings",
			properties: map[string]any{"error": map[string]any{"details": []any{map[string]any{"label": 1, "message": true}}}},
		},
		{
			name:       "details is empty",
			properties: map[string]any{"error": map[string]any{"details": []any{}}},
		},
		{
			name:       "no error key at all",
			properties: map[string]any{"status": float64(400)},
		},
		{
			// Partial detail: still worth surfacing what it does carry.
			name:       "message without a label",
			properties: map[string]any{"error": map[string]any{"details": []any{map[string]any{"message": "'id' is required"}}}},
			want:       "('id' is required)",
		},
		{
			name:       "label without a message",
			properties: map[string]any{"error": map[string]any{"details": []any{map[string]any{"label": "input.steps[0].id"}}}},
			want:       "(input.steps[0].id)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := &ingestion.APIError{Message: "Invalid payload", Status: 400, AdditionalProperties: tc.properties}

			got := Explain(err)

			if tc.want == "" {
				if got != err.Error() {
					t.Errorf("Explain() = %q, want the bare error text %q", got, err.Error())
				}

				return
			}
			if !strings.Contains(got, tc.want) {
				t.Errorf("Explain() = %q, want it to contain %q", got, tc.want)
			}
		})
	}
}
