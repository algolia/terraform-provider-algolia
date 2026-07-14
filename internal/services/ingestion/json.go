package ingestion

import (
	"encoding/json"
	"reflect"
	"sort"
)

// jsonSemanticallyEqual returns true if two strings are both valid JSON and
// represent the same data structure, ignoring key order, whitespace, and
// array element order (the Ingestion API may echo back array elements in a
// different order than what was sent). Used by flattenSource to decide
// whether a JSON-encoded attribute (e.g. a source's `input`) actually
// changed, or the API just returned an equivalent encoding of the same
// configuration.
//
// This mirrors the identically named helper in internal/services/index -
// replicated here rather than imported, since it's two small, dependency-
// free functions and importing across service packages isn't worth it for
// that.
func jsonSemanticallyEqual(a, b string) bool {
	var va, vb any
	if err := json.Unmarshal([]byte(a), &va); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &vb); err != nil {
		return false
	}

	return reflect.DeepEqual(normalizeJSON(va), normalizeJSON(vb))
}

// normalizeJSON recursively sorts arrays of strings and normalizes maps so
// that JSON comparison is order-independent.
func normalizeJSON(v any) any {
	switch val := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(val))
		for k, v := range val {
			m[k] = normalizeJSON(v)
		}
		return m
	case []any:
		normalized := make([]any, len(val))
		for i, item := range val {
			normalized[i] = normalizeJSON(item)
		}
		// Sort the slice if all elements are strings.
		allStrings := true
		for _, item := range normalized {
			if _, ok := item.(string); !ok {
				allStrings = false
				break
			}
		}
		if allStrings {
			sort.Slice(normalized, func(i, j int) bool {
				return normalized[i].(string) < normalized[j].(string)
			})
		}
		return normalized
	default:
		return v
	}
}
