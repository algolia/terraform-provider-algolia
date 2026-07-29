package agent

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	agentStudio "github.com/algolia/algoliasearch-client-go/v4/algolia/agent-studio"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"
)

// TestCallAgentAPI_errorsCarryTheStatus pins the 404 detection Read and
// ImportState rely on to drop a deleted agent from state. The client's typed
// methods, which built the *agentStudio.APIError themselves, are no longer used,
// so callAgentAPI has to produce an error callers can still match on.
func TestCallAgentAPI_errorsCarryTheStatus(t *testing.T) {
	_, err := callAgentAPI(func() (*http.Response, []byte, error) {
		return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody},
			[]byte(`{"message":"Agent not found"}`), nil
	})

	var apiErr *agentStudio.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want an *agentStudio.APIError", err)
	}
	if apiErr.Status != http.StatusNotFound {
		t.Errorf("status = %d, want %d", apiErr.Status, http.StatusNotFound)
	}
	if apiErr.Message != "Agent not found" {
		t.Errorf("message = %q, want %q", apiErr.Message, "Agent not found")
	}
}

// TestCallAgentAPI_notFoundIsRecognisedByAlgoliaerr pins the seam between the
// error callAgentAPI builds and the check Read uses to drop a deleted agent from
// state. The two halves live in different packages, so nothing else would notice
// if they stopped agreeing.
func TestCallAgentAPI_notFoundIsRecognisedByAlgoliaerr(t *testing.T) {
	_, err := callAgentAPI(func() (*http.Response, []byte, error) {
		return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody},
			[]byte(`{"message":"Agent not found"}`), nil
	})

	if !algoliaerr.IsNotFound(err) {
		t.Fatalf("algoliaerr.IsNotFound(%#v) = false, want true", err)
	}

	// A different failure must not be mistaken for a missing agent, or a
	// transient error would silently delete the resource from state.
	_, err = callAgentAPI(func() (*http.Response, []byte, error) {
		return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}, nil, nil
	})

	if algoliaerr.IsNotFound(err) {
		t.Errorf("algoliaerr.IsNotFound(%#v) = true for a 500, want false", err)
	}
}

// TestDecodeAgentDocument_toolSlicesStayAligned covers the invariant flattenTools
// indexes on, for the payload most likely to break it: a tool whose `type` this
// provider version does not know. The typed decode does not skip such a tool - it
// falls back to another variant, or fails the whole response - so the raw and
// typed tool slices cannot end up different lengths.
func TestDecodeAgentDocument_toolSlicesStayAligned(t *testing.T) {
	body := `{
	  "id": "a", "name": "a", "instructions": "i", "status": "draft", "createdAt": "t",
	  "tools": [
	    {"name": "get_order", "type": "client_side", "description": "d", "inputSchema": {"type": "object"}},
	    {"name": "some_future_tool", "type": "future_tool", "someNewField": true}
	  ]
	}`

	doc, err := decodeAgentDocument([]byte(body))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(doc.raw.tools) != len(doc.agent.Tools) {
		t.Fatalf("%d raw tools for %d typed tools", len(doc.raw.tools), len(doc.agent.Tools))
	}
	if len(doc.agent.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(doc.agent.Tools))
	}
}

// TestDecodeAgentDocument_toleratesUnreadableIndices pins that the raw extraction
// is never stricter than the typed decode. `indices` is the only field it needs a
// shape for, and a tool carrying that key with some other shape still decodes
// today, so rejecting it here would break a read that currently works.
func TestDecodeAgentDocument_toleratesUnreadableIndices(t *testing.T) {
	body := `{
	  "id": "a", "name": "a", "instructions": "i", "status": "draft", "createdAt": "t",
	  "tools": [
	    {"name": "my_mcp", "type": "mcp_tools", "url": "https://mcp.example.com", "headers": {}, "indices": {"not": "an array"}}
	  ]
	}`

	doc, err := decodeAgentDocument([]byte(body))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc.raw.tools[0].searchParameters != nil {
		t.Errorf("searchParameters = %v, want none", doc.raw.tools[0].searchParameters)
	}
	if doc.agent.Tools[0].McpServerToolConfig == nil {
		t.Errorf("expected the tool to still decode as an MCP tool, got %#v", doc.agent.Tools[0])
	}
}

// TestCallAgentAPI_sanitizesErrorBodies covers a body that did not come from
// Algolia - a proxy's error page, say. It reaches a diagnostic and CI logs, so it
// must not be able to forge log lines or emit terminal escapes.
func TestCallAgentAPI_sanitizesErrorBodies(t *testing.T) {
	_, err := callAgentAPI(func() (*http.Response, []byte, error) {
		return &http.Response{StatusCode: http.StatusBadGateway, Body: http.NoBody},
			[]byte("bad\n\x1b[31mgateway\x1b[0m"), nil
	})

	var apiErr *agentStudio.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want an *agentStudio.APIError", err)
	}
	if strings.ContainsAny(apiErr.Message, "\n\x1b") {
		t.Errorf("message kept control characters: %q", apiErr.Message)
	}
}
