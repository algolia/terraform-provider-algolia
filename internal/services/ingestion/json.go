package ingestion

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
)

// decodeJSONStrict decodes raw into target, rejecting keys that target does not
// declare.
//
// Strictness is deliberate. The Algolia client's oneOf request bodies (AuthInput,
// SourceInput and friends) carry no discriminator field, so the provider picks
// the variant from the resource's declared `type` and decodes into it directly.
// A plain json.Unmarshal would then silently drop every key belonging to some
// other variant, which turns an `input`/`type` mismatch into a request carrying
// the declared variant's zero values. Failing the decode surfaces the mismatch
// instead.
func decodeJSONStrict(raw []byte, target any) error {
	return decodeJSONObject(raw, target, true)
}

// decodeJSONLenient decodes raw into target while tolerating keys target does
// not declare. The update variants of the oneOf request bodies deliberately omit
// a resource's immutable fields, which are still present in the configured
// `input`, so strictness there would reject legitimate configurations.
func decodeJSONLenient(raw []byte, target any) error {
	return decodeJSONObject(raw, target, false)
}

// decodeJSONObject decodes a single JSON object from raw into target.
//
// Beyond the optional key strictness, it rejects two inputs that
// encoding/json accepts but that carry no configuration at all:
//
//   - A bare `null`. Decoding it into a struct target is a no-op, leaving every
//     field zero. `input = jsonencode(null)` would therefore reach Algolia as
//     `{"apiKey":"","appID":""}` - the same empty-credentials failure that
//     selecting the variant from `type` exists to prevent.
//   - Anything following the first value. Decode consumes one value and ignores
//     the rest, so `{"appID":"A"} {"appID":"other"}` would silently use the
//     first object and discard the second.
func decodeJSONObject(raw []byte, target any, strict bool) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		return errors.New("expected a JSON object, got an empty value")
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	if strict {
		decoder.DisallowUnknownFields()
	}

	var probe json.RawMessage
	if err := json.Unmarshal(raw, &probe); err == nil && bytes.Equal(bytes.TrimSpace(probe), []byte("null")) {
		return errors.New("expected a JSON object, got null")
	}

	if err := decoder.Decode(target); err != nil {
		return err
	}

	if decoder.More() {
		return errors.New("unexpected trailing content after the JSON object")
	}

	return nil
}

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
