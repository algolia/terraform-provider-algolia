package recommend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	recommendapi "github.com/algolia/algoliasearch-client-go/v4/algolia/recommend"
)

// metadataJSONKey is the response field the vendored client cannot decode.
// See getRecommendRule for the full story.
const metadataJSONKey = "_metadata"

// getRecommendRule fetches a single Recommend rule.
//
// It exists to work around a bug in the vendored algoliasearch-client-go v4:
// recommend.RuleMetadata declares `lastUpdate` as *string (see
// algolia/recommend/model_rule_metadata.go), but the API returns it as a Unix
// epoch number (`"_metadata":{"lastUpdate":1785242476}`). The client's own
// recommend.APIClient.GetRecommendRule therefore always fails with:
//
//	cannot decode result: failed to unmarshal response body: json: cannot
//	unmarshal number into Go struct field RuleMetadata._metadata.lastUpdate
//	of type string
//
// GetRecommendRule sits on every non-Delete path of algolia_recommend_rule
// (Create, Read, Update, ImportState, and the data source), so the resource
// was entirely unusable without this.
//
// The workaround calls GetRecommendRuleWithHTTPInfo, which hands back the raw
// response body, drops the `_metadata` object, and decodes the rest into the
// client's own recommend.RecommendRule. `_metadata` carries no configuration -
// flattenRecommendRule never reads it - so discarding it loses nothing. No
// field list is mirrored here, so fields added by a future client bump are
// picked up automatically.
//
// Remove this helper (and its test) and call client.GetRecommendRule directly
// once the upstream client decodes a numeric `lastUpdate` - i.e. once
// recommend.RuleMetadata.LastUpdate is no longer a plain *string. See
// https://github.com/algolia/api-clients-automation. The
// ALGOLIA_RUN_RECOMMEND_ACC gate in resource_test.go exists for the same bug.
func getRecommendRule(
	client *recommendapi.APIClient,
	req recommendapi.ApiGetRecommendRuleRequest,
	opts ...recommendapi.RequestOption,
) (*recommendapi.RecommendRule, error) {
	res, body, err := client.GetRecommendRuleWithHTTPInfo(req, opts...)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, fmt.Errorf("no response from the Recommend API")
	}
	defer func() { _ = res.Body.Close() }()

	// Mirror the client's own error handling so that callers can keep using
	// errors.As on *recommendapi.APIError (Read and Delete rely on Status ==
	// 404 to detect a rule that no longer exists).
	if res.StatusCode >= 300 {
		return nil, &recommendapi.APIError{
			Message: apiErrorMessage(body, res.StatusCode),
			Status:  res.StatusCode,
		}
	}

	return decodeRecommendRule(body)
}

// decodeRecommendRule decodes a GetRecommendRule response body into the
// client's RecommendRule, after removing the `_metadata` field the client
// mistypes (see getRecommendRule). Decoding is otherwise identical to the
// client's own unexported decode: RecommendRule is a plain struct with no
// custom UnmarshalJSON and is not a oneOf schema, so the client also just
// calls json.Unmarshal on it.
func decodeRecommendRule(body []byte) (*recommendapi.RecommendRule, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, fmt.Errorf("cannot decode Recommend rule response: %w", err)
	}
	// A literal `null` unmarshals into a nil map without error, and every step
	// below is a no-op on nil, so it would otherwise yield a zero-value rule with
	// an empty objectID and flatten that into state.
	if fields == nil {
		return nil, fmt.Errorf("cannot decode Recommend rule response: expected an object, got null")
	}
	delete(fields, metadataJSONKey)

	sanitized, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("cannot re-encode Recommend rule response: %w", err)
	}

	rule := recommendapi.NewEmptyRecommendRule()
	if err := json.Unmarshal(sanitized, rule); err != nil {
		return nil, fmt.Errorf("cannot decode Recommend rule response: %w", err)
	}

	return rule, nil
}

// apiErrorMessage extracts a human-readable message from an error response
// body, falling back to the raw body and then to the status text, mirroring
// what the client's unexported decodeError does.
func apiErrorMessage(body []byte, status int) string {
	var payload struct {
		Message *string `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && payload.Message != nil {
		return *payload.Message
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
// diagnostic: newlines collapse to spaces so it cannot forge extra log lines,
// invalid UTF-8 is dropped, and the result is truncated.
func summarizeErrorBody(body []byte) string {
	// Drop invalid sequences first: strings.Map turns an invalid byte into
	// U+FFFD, which is itself valid UTF-8 and would then survive the cleanup.
	flattened := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}

		return r
	}, strings.ToValidUTF8(string(body), ""))

	msg := strings.TrimSpace(flattened)
	if len(msg) > maxErrorBodyLen {
		return msg[:maxErrorBodyLen] + "... (truncated)"
	}

	return msg
}
