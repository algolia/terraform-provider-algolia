package ingestion

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
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
// represent the same data structure, ignoring object key order and whitespace.
// Array order is significant: dismissing a reorder here can skip a PATCH.
//
// This comparison is strict about nulls: a key present with a null value is not
// the same as an absent key. That matters because expandSourceUpdate uses it to
// decide whether to send `input` in a PATCH at all, so a difference judged
// insignificant here is a write that never happens. For the looser comparison
// used when deciding whether to adopt an API response into state, see
// jsonEqualIgnoringAPINulls.
func jsonSemanticallyEqual(a, b string) bool {
	var va, vb any
	if err := json.Unmarshal([]byte(a), &va); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &vb); err != nil {
		return false
	}

	return reflect.DeepEqual(va, vb)
}

// jsonEqualIgnoringAPINulls reports whether an API response carries the same
// configuration as the configured value, tolerating keys the API added with an
// explicit null. The Ingestion API fills optional fields it was not given that
// way - a no-code transformation step comes back with a `condition` nobody wrote -
// and counting that as a change makes the applied value disagree with the plan on
// every apply.
//
// The tolerance is deliberately one-directional. Nulls are dropped from the API
// value only, never from the configured one, so this cannot be used to argue that
// two different configurations are the same: `{"a":null}` and `{"b":null}` are
// still different documents, and a null the operator wrote that the API dropped
// still reads as drift. It is also deliberately not used to gate any write.
func jsonEqualIgnoringAPINulls(configured, apiValue string) bool {
	var vc, va any
	if err := json.Unmarshal([]byte(configured), &vc); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(apiValue), &va); err != nil {
		return false
	}

	return reflect.DeepEqual(normalizeJSON(vc, false), normalizeJSON(va, true))
}

// normalizeJSON recursively normalizes a decoded JSON value, preserving array order.
// dropNulls omits object keys whose value is null. Only ever true for the side of
// a comparison that came from the API - see jsonEqualIgnoringAPINulls.
func normalizeJSON(v any, dropNulls bool) any {
	switch val := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(val))
		for k, v := range val {
			if dropNulls && v == nil {
				continue
			}
			m[k] = normalizeJSON(v, dropNulls)
		}
		return m
	case []any:
		normalized := make([]any, len(val))
		for i, item := range val {
			normalized[i] = normalizeJSON(item, dropNulls)
		}
		return normalized
	default:
		return v
	}
}
