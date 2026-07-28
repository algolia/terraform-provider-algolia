package recommend

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
