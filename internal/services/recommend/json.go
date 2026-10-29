package recommend

import (
	"encoding/json"
	"reflect"
)

// jsonSemanticallyEqual ignores object key order and whitespace, but preserves
// array order so a reordered API response is not mistaken for the prior state.
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
