package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

// decodeJSONObject decodes a JSON object while keeping numbers as json.Number,
// so re-encoding never reformats a value the user wrote.
func decodeJSONObject(data []byte) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var decoded map[string]any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}
	// A literal `null` decodes into a nil map without error, which callers would
	// then read as "no document at all".
	if decoded == nil {
		return nil, fmt.Errorf("expected a JSON object, got null")
	}

	return decoded, nil
}

// jsonEqual reports whether two JSON documents carry the same data, ignoring
// key order and whitespace.
func jsonEqual(left, right []byte) bool {
	var leftDecoded, rightDecoded any
	if err := json.Unmarshal(left, &leftDecoded); err != nil {
		return false
	}
	if err := json.Unmarshal(right, &rightDecoded); err != nil {
		return false
	}

	return reflect.DeepEqual(leftDecoded, rightDecoded)
}

// isJSONNull reports whether a raw document is absent or the literal `null`,
// i.e. whether the attribute holding it has no value.
func isJSONNull(document json.RawMessage) bool {
	trimmed := bytes.TrimSpace(document)

	return len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null"))
}
