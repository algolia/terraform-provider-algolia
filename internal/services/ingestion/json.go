package ingestion

import (
	"encoding/json"
	"reflect"
	"sort"
)

// jsonSemanticallyEqual returns true if two strings are both valid JSON and
// represent the same data structure, ignoring object key order, whitespace,
// and the order of arrays whose elements are all the same scalar kind (all
// strings or all numbers) — the Ingestion API may echo such arrays back in a
// different order than what was sent. Arrays containing objects, or with
// mixed element kinds, keep their order, since element order there is
// usually significant. Used by flattenSource to decide whether a
// JSON-encoded attribute (e.g. a source's `input`) actually changed, or the
// API just returned an equivalent encoding of the same configuration.
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

// normalizeJSON recursively normalizes a decoded JSON value so that
// comparison is order-independent where order is insignificant: object keys
// (already order-independent as Go maps) are recursed into, and arrays whose
// elements are all strings or all numbers are sorted. Arrays containing
// objects or a mix of kinds are left in their original order.
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
		sortScalarSlice(normalized)
		return normalized
	default:
		return v
	}
}

// sortScalarSlice sorts s in place only when every element is a string, or
// every element is a float64 (JSON numbers decode to float64). Slices that
// contain objects or mix element kinds are left untouched, since reordering
// them could change meaning.
func sortScalarSlice(s []any) {
	if len(s) < 2 {
		return
	}

	allStrings, allNumbers := true, true
	for _, item := range s {
		if _, ok := item.(string); !ok {
			allStrings = false
		}
		if _, ok := item.(float64); !ok {
			allNumbers = false
		}
	}

	switch {
	case allStrings:
		sort.Slice(s, func(i, j int) bool { return s[i].(string) < s[j].(string) })
	case allNumbers:
		sort.Slice(s, func(i, j int) bool { return s[i].(float64) < s[j].(float64) })
	}
}
