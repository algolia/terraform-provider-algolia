package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	agentStudio "github.com/algolia/algoliasearch-client-go/v4/algolia/agent-studio"
)

// agentDocument is an agent as the API returned it: the client's typed model,
// plus the raw JSON of the documents Terraform stores verbatim.
//
// Four attributes hold a whole JSON document written by the user - `config`,
// `tool_client_side.input_schema`,
// `tool_algolia_recommend.predefined_recommend_parameters` and
// `tool_algolia_search.index.search_parameters`. None of them can be rebuilt
// from the typed model: agentStudio.ClientToolsArgsSchema and
// agentStudio.SearchParameters have no catch-all field, so decoding through
// them drops every key the vendored client does not model yet, and even the
// lossless map fields come back with their keys reordered and their numbers
// reformatted. All four attributes are Required or Optional, so their planned
// value is the configuration verbatim and any of those differences is enough
// for Terraform to reject the apply with "Provider produced inconsistent result
// after apply".
type agentDocument struct {
	agent *agentStudio.AgentWithVersionResponse
	raw   rawAgent
}

// rawAgent holds an agent payload's untouched JSON documents.
type rawAgent struct {
	config json.RawMessage
	// tools is positional: entry i describes the same tool as entry i of
	// AgentWithVersionResponse.Tools, both being decoded from one `tools` array.
	tools []rawTool
}

// rawTool holds the untouched JSON documents nested in one entry of a payload's
// `tools` array. searchParameters is keyed by index name, which is unique
// within a tool.
type rawTool struct {
	inputSchema                   json.RawMessage
	predefinedRecommendParameters json.RawMessage
	searchParameters              map[string]json.RawMessage
}

// agentAPICall is one of the client's *WithHTTPInfo methods bound to its
// request. Those are used in place of the typed methods so the response body is
// available verbatim.
type agentAPICall func() (*http.Response, []byte, error)

func createAgent(ctx context.Context, client *agentStudio.APIClient, config *agentStudio.AgentConfigCreate) (*agentDocument, error) {
	return callAgentAPI(func() (*http.Response, []byte, error) {
		return client.CreateAgentWithHTTPInfo(client.NewApiCreateAgentRequest(config), agentStudio.WithContext(ctx))
	})
}

func getAgent(ctx context.Context, client *agentStudio.APIClient, agentID string) (*agentDocument, error) {
	return callAgentAPI(func() (*http.Response, []byte, error) {
		return client.GetAgentWithHTTPInfo(client.NewApiGetAgentRequest(agentID), agentStudio.WithContext(ctx))
	})
}

func updateAgent(ctx context.Context, client *agentStudio.APIClient, agentID string, config *agentStudio.AgentConfigUpdate) (*agentDocument, error) {
	return callAgentAPI(func() (*http.Response, []byte, error) {
		return client.UpdateAgentWithHTTPInfo(client.NewApiUpdateAgentRequest(agentID, config), agentStudio.WithContext(ctx))
	})
}

func publishAgent(ctx context.Context, client *agentStudio.APIClient, agentID string) (*agentDocument, error) {
	return callAgentAPI(func() (*http.Response, []byte, error) {
		return client.PublishAgentWithHTTPInfo(client.NewApiPublishAgentRequest(agentID), agentStudio.WithContext(ctx))
	})
}

// callAgentAPI runs an agent endpoint and decodes its raw response.
//
// Errors are turned into *agentStudio.APIError, mirroring the client's own
// unexported error handling, so callers can keep matching on the status to
// detect an agent that no longer exists.
func callAgentAPI(call agentAPICall) (*agentDocument, error) {
	res, body, err := call()
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, fmt.Errorf("no response from the Agent Studio API")
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode >= 300 {
		return nil, &agentStudio.APIError{
			Message: apiErrorMessage(body, res.StatusCode),
			Status:  res.StatusCode,
		}
	}

	return decodeAgentDocument(body)
}

// decodeAgentDocument decodes a successful agent payload into both its typed and
// its raw form.
func decodeAgentDocument(body []byte) (*agentDocument, error) {
	var agent agentStudio.AgentWithVersionResponse
	if err := json.Unmarshal(body, &agent); err != nil {
		return nil, fmt.Errorf("cannot decode agent response: %w", err)
	}

	raw, err := extractRawAgent(body)
	if err != nil {
		return nil, err
	}
	// Both slices are decoded from the same `tools` array, so a mismatch would
	// mean rawAgent.tools no longer lines up with the typed tools it annotates.
	if len(raw.tools) != len(agent.Tools) {
		return nil, fmt.Errorf("cannot decode agent response: %d raw tools for %d decoded tools",
			len(raw.tools), len(agent.Tools))
	}

	return &agentDocument{agent: &agent, raw: raw}, nil
}

// extractRawAgent pulls the JSON documents Terraform stores verbatim out of an
// agent payload without decoding their contents, so the bytes the API returned
// are the bytes that reach state.
//
// Every field it reads is a json.RawMessage, which accepts any shape, so this
// never rejects a payload the typed decode accepts: being stricter than the
// client would turn a working agent into a failed read. `indices` is the one
// field needing a shape, and it is decoded separately for the same reason - see
// extractRawIndices.
func extractRawAgent(payload []byte) (rawAgent, error) {
	var decoded struct {
		Config json.RawMessage `json:"config"`
		Tools  []struct {
			InputSchema                   json.RawMessage `json:"inputSchema"`
			PredefinedRecommendParameters json.RawMessage `json:"predefinedRecommendParameters"`
			Indices                       json.RawMessage `json:"indices"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return rawAgent{}, fmt.Errorf("cannot decode agent response: %w", err)
	}

	raw := rawAgent{
		config: decoded.Config,
		tools:  make([]rawTool, 0, len(decoded.Tools)),
	}
	for _, tool := range decoded.Tools {
		raw.tools = append(raw.tools, rawTool{
			inputSchema:                   tool.InputSchema,
			predefinedRecommendParameters: tool.PredefinedRecommendParameters,
			searchParameters:              extractRawIndices(tool.Indices),
		})
	}

	return raw, nil
}

// extractRawIndices maps an algolia_search_index tool's `indices` array to each
// index's searchParameters document, keyed by index name.
//
// A shape it cannot read yields no documents rather than an error. `indices` is
// only an array of named objects for algolia_search_index tools, and the client's
// own oneOf decoder tolerates any other tool carrying the key, so failing here
// would reject payloads that decode fine today. A search tool whose `indices` is
// genuinely malformed fails the typed decode anyway, since
// AlgoliaSearchToolIndexConfig.Index is a plain string.
func extractRawIndices(indices json.RawMessage) map[string]json.RawMessage {
	var decoded []struct {
		Index            string          `json:"index"`
		SearchParameters json.RawMessage `json:"searchParameters"`
	}
	if err := json.Unmarshal(indices, &decoded); err != nil {
		return nil
	}

	documents := make(map[string]json.RawMessage, len(decoded))
	for _, index := range decoded {
		documents[index.Index] = index.SearchParameters
	}

	return documents
}

// apiErrorMessage extracts a human-readable message from an error response
// body, falling back to the raw body and then to the status text, mirroring
// what the client's unexported decodeError does.
func apiErrorMessage(body []byte, status int) string {
	var payload struct {
		Message *string `json:"message"`
	}
	// A structured `message` is no more trustworthy than the raw body: it is the
	// shape an intermediary's error page uses too, and it is just as unbounded.
	if err := json.Unmarshal(body, &payload); err == nil && payload.Message != nil {
		return summarizeErrorBody([]byte(*payload.Message))
	}
	if len(body) > 0 {
		return summarizeErrorBody(body)
	}

	return http.StatusText(status)
}

// maxErrorBodyLen bounds how much of an unrecognised error body reaches a
// diagnostic. Bodies are read unbounded by the client's transport and may come
// from an intermediary rather than Algolia (a proxy or WAF error page), so they
// are neither size- nor content-trusted.
const maxErrorBodyLen = 2048

// summarizeErrorBody makes an arbitrary response body safe to put in a
// diagnostic: control characters collapse to spaces so it cannot forge extra log
// lines or emit terminal escapes, invalid UTF-8 is dropped, and truncation lands
// on a rune boundary.
func summarizeErrorBody(body []byte) string {
	// Drop invalid sequences first: strings.Map turns an invalid byte into
	// U+FFFD, which is itself valid UTF-8 and would then survive the cleanup.
	flattened := strings.Map(func(r rune) rune {
		// Covers \n, \r, \t, \v, \f and ANSI escapes, all of which end up in CI
		// logs where they can forge lines or move the cursor.
		if unicode.IsControl(r) {
			return ' '
		}

		return r
	}, strings.ToValidUTF8(string(body), ""))

	msg := strings.TrimSpace(flattened)
	if len(msg) <= maxErrorBodyLen {
		return msg
	}

	// Truncate on a rune boundary; slicing raw bytes could split a multi-byte
	// rune and reintroduce the invalid UTF-8 removed just above.
	cut := maxErrorBodyLen
	for cut > 0 && !utf8.RuneStart(msg[cut]) {
		cut--
	}

	return msg[:cut] + "... (truncated)"
}
