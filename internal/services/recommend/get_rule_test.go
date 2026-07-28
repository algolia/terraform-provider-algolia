package recommend

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/call"
	recommendapi "github.com/algolia/algoliasearch-client-go/v4/algolia/recommend"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/transport"
)

// ruleBodyWithNumericLastUpdate is a verbatim-shaped GET
// /1/indexes/{indexName}/{model}/recommend/rules/{objectID} response body:
// `_metadata.lastUpdate` is a Unix epoch *number*, not the RFC 3339 string the
// vendored client's recommend.RuleMetadata declares. See getRecommendRule.
const ruleBodyWithNumericLastUpdate = `{
  "_metadata": {"lastUpdate": 1785242476},
  "objectID": "hide-rule",
  "condition": {"filters": "brand:apple"},
  "consequence": {"hide": [{"objectID": "42"}]},
  "description": "hide discontinued items",
  "enabled": true,
  "validity": [{"from": 1893456000, "until": 1893542400}]
}`

// TestUpstreamClientStillMistypesLastUpdate documents the upstream bug this
// package works around. If this ever reports that the plain decode now works,
// recommend.RuleMetadata.LastUpdate has been fixed upstream and getRecommendRule
// (plus the ALGOLIA_RUN_RECOMMEND_ACC gate in resource_test.go) can be removed.
func TestUpstreamClientStillMistypesLastUpdate(t *testing.T) {
	var rule recommendapi.RecommendRule
	err := json.Unmarshal([]byte(ruleBodyWithNumericLastUpdate), &rule)
	if err == nil {
		t.Log("recommend.RecommendRule now decodes a numeric _metadata.lastUpdate; " +
			"the getRecommendRule workaround and the ALGOLIA_RUN_RECOMMEND_ACC gate can be removed")
		return
	}
	if !strings.Contains(err.Error(), "RuleMetadata") {
		t.Fatalf("unexpected decode error, want one mentioning RuleMetadata: %v", err)
	}
}

func TestDecodeRecommendRule_NumericLastUpdate(t *testing.T) {
	rule, err := decodeRecommendRule([]byte(ruleBodyWithNumericLastUpdate))
	if err != nil {
		t.Fatalf("decodeRecommendRule() error = %v, want nil", err)
	}

	if got := rule.GetObjectID(); got != "hide-rule" {
		t.Fatalf("objectID = %q, want hide-rule", got)
	}
	if rule.Condition == nil || rule.Condition.GetFilters() != "brand:apple" {
		t.Fatalf("condition = %v, want filters brand:apple", rule.Condition)
	}
	if rule.Consequence == nil || len(rule.Consequence.Hide) != 1 ||
		rule.Consequence.Hide[0].GetObjectID() != "42" {
		t.Fatalf("consequence = %v, want a single hide of objectID 42", rule.Consequence)
	}
	if got := rule.GetDescription(); got != "hide discontinued items" {
		t.Fatalf("description = %q, want hide discontinued items", got)
	}
	if !rule.GetEnabled() {
		t.Fatal("enabled = false, want true")
	}
	if len(rule.Validity) != 1 || rule.Validity[0].GetFrom() != 1893456000 ||
		rule.Validity[0].GetUntil() != 1893542400 {
		t.Fatalf("validity = %v, want a single 1893456000-1893542400 range", rule.Validity)
	}
	// _metadata is dropped rather than surfaced, since nothing in the provider
	// reads it.
	if rule.Metadata != nil {
		t.Fatalf("_metadata = %v, want it to be discarded", rule.Metadata)
	}
}

// TestGetRecommendRule exercises the whole helper against an HTTP server that
// replies exactly like the real API does, so it covers the raw-body decode and
// the error mapping Read/Delete depend on.
func TestGetRecommendRule(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantErr    bool
		wantStatus int
	}{
		{
			name:   "numeric lastUpdate decodes",
			status: http.StatusOK,
			body:   ruleBodyWithNumericLastUpdate,
		},
		{
			name:       "not found maps to a 404 APIError",
			status:     http.StatusNotFound,
			body:       `{"message":"ObjectID does not exist"}`,
			wantErr:    true,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := newTestRecommendClient(t, server)
			rule, err := getRecommendRule(client,
				client.NewApiGetRecommendRuleRequest("products", recommendapi.RECOMMEND_MODELS_RELATED_PRODUCTS, "hide-rule"))

			if want := "/1/indexes/products/related-products/recommend/rules/hide-rule"; gotPath != want {
				t.Fatalf("request path = %q, want %q", gotPath, want)
			}

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("getRecommendRule() error = %v, want nil", err)
				}
				if got := rule.GetObjectID(); got != "hide-rule" {
					t.Fatalf("objectID = %q, want hide-rule", got)
				}

				return
			}

			if err == nil {
				t.Fatal("getRecommendRule() error = nil, want an error")
			}
			var apiErr *recommendapi.APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error = %T (%v), want *recommendapi.APIError", err, err)
			}
			if apiErr.Status != tt.wantStatus {
				t.Fatalf("APIError.Status = %d, want %d", apiErr.Status, tt.wantStatus)
			}
		})
	}
}

// newTestRecommendClient returns a Recommend client whose only host is the
// given test server.
func newTestRecommendClient(t *testing.T, server *httptest.Server) *recommendapi.APIClient {
	t.Helper()

	client, err := recommendapi.NewClientWithConfig(recommendapi.RecommendConfiguration{
		Configuration: transport.Configuration{
			AppID:  "test-app",
			ApiKey: "test-key",
			Hosts: []transport.StatefulHost{
				transport.NewStatefulHost("http", server.Listener.Addr().String(), func(k call.Kind) bool { return true }),
			},
		},
	})
	if err != nil {
		t.Fatalf("could not build test Recommend client: %v", err)
	}

	return client
}

// TestDecodeRecommendRule_RejectsNullBody covers a review finding: a literal
// `null` body unmarshals into a nil map without error, and stripping _metadata
// then re-marshalling are both no-ops on nil, so a 200 null would have produced a
// zero-value rule with an empty objectID and flattened that into state.
func TestDecodeRecommendRule_RejectsNullBody(t *testing.T) {
	for _, body := range []string{`null`, "  null\n"} {
		rule, err := decodeRecommendRule([]byte(body))
		if err == nil {
			t.Errorf("decodeRecommendRule(%q) returned rule %+v, want an error", body, rule)
			continue
		}
		if !strings.Contains(err.Error(), "got null") {
			t.Errorf("decodeRecommendRule(%q) error = %q, want it to mention null", body, err.Error())
		}
	}
}

// TestSummarizeErrorBody bounds and flattens an untrusted error body before it
// reaches a diagnostic. The body is read unbounded by the client's transport and
// may come from an intermediary rather than Algolia, so it is neither size- nor
// content-trusted.
func TestSummarizeErrorBody(t *testing.T) {
	t.Run("collapses newlines so it cannot forge log lines", func(t *testing.T) {
		got := summarizeErrorBody([]byte("upstream failed\nERROR: injected line\r\n\tindented"))
		if strings.ContainsAny(got, "\n\r\t") {
			t.Errorf("summarizeErrorBody kept a control character: %q", got)
		}
		if !strings.Contains(got, "upstream failed") {
			t.Errorf("summarizeErrorBody dropped the message: %q", got)
		}
	})

	t.Run("truncates an oversized body", func(t *testing.T) {
		got := summarizeErrorBody([]byte(strings.Repeat("A", maxErrorBodyLen*3)))
		if len(got) > maxErrorBodyLen+len("... (truncated)") {
			t.Errorf("summarizeErrorBody returned %d bytes, want it bounded near %d", len(got), maxErrorBodyLen)
		}
		if !strings.HasSuffix(got, "... (truncated)") {
			t.Error("summarizeErrorBody did not mark the body as truncated")
		}
	})

	t.Run("drops invalid UTF-8", func(t *testing.T) {
		if got := summarizeErrorBody([]byte{0xff, 0xfe, 'o', 'k'}); got != "ok" {
			t.Errorf("summarizeErrorBody = %q, want %q", got, "ok")
		}
	})

	t.Run("a structured message is preferred over the raw body", func(t *testing.T) {
		if got := apiErrorMessage([]byte(`{"message":"Rule not found","status":404}`), 404); got != "Rule not found" {
			t.Errorf("apiErrorMessage = %q, want the structured message", got)
		}
	})

	// The structured `message` path used to return the value unbounded, which is
	// the shape most intermediaries emit, so the bound was bypassed in practice.
	t.Run("a structured message is bounded too", func(t *testing.T) {
		body, err := json.Marshal(map[string]string{"message": strings.Repeat("B", maxErrorBodyLen*2) + "\ninjected"})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		got := apiErrorMessage(body, 500)
		if len(got) > maxErrorBodyLen+len("... (truncated)") {
			t.Errorf("apiErrorMessage returned %d bytes, want it bounded near %d", len(got), maxErrorBodyLen)
		}
		if strings.ContainsAny(got, "\n\r") {
			t.Error("apiErrorMessage kept a newline from the structured message")
		}
	})

	t.Run("truncation lands on a rune boundary", func(t *testing.T) {
		// Multi-byte runes so a byte-wise cut would split one and reintroduce
		// invalid UTF-8 -- exactly what ToValidUTF8 above removed.
		got := summarizeErrorBody([]byte(strings.Repeat("é", maxErrorBodyLen)))
		if !utf8.ValidString(got) {
			t.Errorf("summarizeErrorBody produced invalid UTF-8: %q", got)
		}
	})
}
