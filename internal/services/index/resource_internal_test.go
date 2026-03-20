package index

import "testing"

func TestJsonSemanticallyEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "identical JSON",
			a:        `{"a":"b"}`,
			b:        `{"a":"b"}`,
			expected: true,
		},
		{
			name:     "different key order",
			a:        `{"a":"1","b":"2"}`,
			b:        `{"b":"2","a":"1"}`,
			expected: true,
		},
		{
			name:     "different array order (strings)",
			a:        `{"de":["name","desc"]}`,
			b:        `{"de":["desc","name"]}`,
			expected: true,
		},
		{
			name:     "structurally different values",
			a:        `{"a":"1"}`,
			b:        `{"a":"2"}`,
			expected: false,
		},
		{
			name:     "different keys",
			a:        `{"a":"1"}`,
			b:        `{"b":"1"}`,
			expected: false,
		},
		{
			name:     "invalid JSON (a)",
			a:        `not json`,
			b:        `{}`,
			expected: false,
		},
		{
			name:     "invalid JSON (b)",
			a:        `{}`,
			b:        `not json`,
			expected: false,
		},
		{
			name:     "nested structures equal",
			a:        `{"a":{"b":["x","y"]}}`,
			b:        `{"a":{"b":["y","x"]}}`,
			expected: true,
		},
		{
			name:     "empty objects",
			a:        `{}`,
			b:        `{}`,
			expected: true,
		},
		{
			name:     "JSON primitives equal",
			a:        `"hello"`,
			b:        `"hello"`,
			expected: true,
		},
		{
			name:     "JSON primitives differ",
			a:        `"hello"`,
			b:        `"world"`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsonSemanticallyEqual(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("jsonSemanticallyEqual(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestNormalizeJSON(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want any
	}{
		{
			name: "nil",
			in:   nil,
			want: nil,
		},
		{
			name: "string",
			in:   "hello",
			want: "hello",
		},
		{
			name: "number",
			in:   float64(42),
			want: float64(42),
		},
		{
			name: "string slice sorted",
			in:   []any{"b", "a", "c"},
			want: []any{"a", "b", "c"},
		},
		{
			name: "mixed slice not sorted",
			in:   []any{"b", float64(1), "a"},
			want: []any{"b", float64(1), "a"},
		},
		{
			name: "nested map",
			in:   map[string]any{"k": map[string]any{"z": "1"}},
			want: map[string]any{"k": map[string]any{"z": "1"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeJSON(tt.in)
			if !equalAny(got, tt.want) {
				t.Errorf("normalizeJSON(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// equalAny is a simple deep-equal helper for test assertions.
func equalAny(a, b any) bool {
	switch va := a.(type) {
	case map[string]any:
		vb, ok := b.(map[string]any)
		if !ok || len(va) != len(vb) {
			return false
		}
		for k, v := range va {
			if !equalAny(v, vb[k]) {
				return false
			}
		}
		return true
	case []any:
		vb, ok := b.([]any)
		if !ok || len(va) != len(vb) {
			return false
		}
		for i := range va {
			if !equalAny(va[i], vb[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}
