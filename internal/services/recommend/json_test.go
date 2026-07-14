package recommend

import "testing"

func TestJSONSemanticallyEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "identical",
			a:        `{"filters":"brand:apple"}`,
			b:        `{"filters":"brand:apple"}`,
			expected: true,
		},
		{
			name:     "different key order",
			a:        `{"filters":"brand:apple","context":"mobile"}`,
			b:        `{"context":"mobile","filters":"brand:apple"}`,
			expected: true,
		},
		{
			name:     "different array order",
			a:        `{"hide":["1","2"]}`,
			b:        `{"hide":["2","1"]}`,
			expected: true,
		},
		{
			name:     "different numeric array order",
			a:        `{"scores":[1,2,3]}`,
			b:        `{"scores":[3,1,2]}`,
			expected: true,
		},
		{
			name:     "arrays of objects keep order (not reordered)",
			a:        `{"promote":[{"objectID":"1"},{"objectID":"2"}]}`,
			b:        `{"promote":[{"objectID":"2"},{"objectID":"1"}]}`,
			expected: false,
		},
		{
			name:     "different whitespace",
			a:        "{\n  \"filters\": \"brand:apple\"\n}",
			b:        `{"filters":"brand:apple"}`,
			expected: true,
		},
		{
			name:     "different values",
			a:        `{"filters":"brand:apple"}`,
			b:        `{"filters":"brand:samsung"}`,
			expected: false,
		},
		{
			name:     "invalid json on left",
			a:        `{not valid`,
			b:        `{}`,
			expected: false,
		},
		{
			name:     "invalid json on right",
			a:        `{}`,
			b:        `{not valid`,
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
