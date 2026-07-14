package ingestion

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
			a:        `{"url":"https://example.com/data.csv"}`,
			b:        `{"url":"https://example.com/data.csv"}`,
			expected: true,
		},
		{
			name:     "different key order",
			a:        `{"url":"https://example.com/data.csv","uniqueIDColumn":"id"}`,
			b:        `{"uniqueIDColumn":"id","url":"https://example.com/data.csv"}`,
			expected: true,
		},
		{
			name:     "different array order",
			a:        `{"languages":["en","fr"]}`,
			b:        `{"languages":["fr","en"]}`,
			expected: true,
		},
		{
			name:     "different numeric array order",
			a:        `{"ports":[443,80,8080]}`,
			b:        `{"ports":[80,8080,443]}`,
			expected: true,
		},
		{
			name:     "arrays of objects keep order (not reordered)",
			a:        `{"cols":[{"n":"a"},{"n":"b"}]}`,
			b:        `{"cols":[{"n":"b"},{"n":"a"}]}`,
			expected: false,
		},
		{
			name:     "different whitespace",
			a:        "{\n  \"url\": \"https://example.com/data.csv\"\n}",
			b:        `{"url":"https://example.com/data.csv"}`,
			expected: true,
		},
		{
			name:     "different values",
			a:        `{"url":"https://example.com/old.csv"}`,
			b:        `{"url":"https://example.com/new.csv"}`,
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
