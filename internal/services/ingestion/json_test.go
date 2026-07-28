package ingestion

import (
	"strings"
	"testing"
)

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

// TestDecodeJSONObject_RejectsEmptyPayloads guards a regression found in review:
// encoding/json accepts a bare `null` as a no-op for struct targets and ignores
// anything after the first value, so either would have reached Algolia as an
// empty credentials object -- reintroducing the very bug that selecting the
// union variant from `type` was written to fix.
func TestDecodeJSONObject_RejectsEmptyPayloads(t *testing.T) {
	type payload struct {
		AppID  string `json:"appID"`
		APIKey string `json:"apiKey"`
	}

	tests := []struct {
		name    string
		raw     string
		strict  bool
		wantErr string
	}{
		{name: "bare null", raw: `null`, strict: true, wantErr: "got null"},
		{name: "bare null, lenient", raw: `null`, strict: false, wantErr: "got null"},
		{name: "null with whitespace", raw: "  null\n", strict: true, wantErr: "got null"},
		{name: "empty value", raw: ``, strict: true, wantErr: "empty value"},
		{name: "whitespace only", raw: "   ", strict: true, wantErr: "empty value"},
		{name: "second object", raw: `{"appID":"A","apiKey":"K"} {"appID":"other"}`, strict: true, wantErr: "trailing content"},
		{name: "trailing junk", raw: `{"appID":"A","apiKey":"K"}garbage`, strict: true, wantErr: "trailing content"},
		{name: "second object, lenient", raw: `{"appID":"A"} {"appID":"other"}`, strict: false, wantErr: "trailing content"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target payload
			err := decodeJSONObject([]byte(tt.raw), &target, tt.strict)
			if err == nil {
				t.Fatalf("decodeJSONObject(%q) succeeded and produced %+v, want an error containing %q", tt.raw, target, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("decodeJSONObject(%q) error = %q, want it to contain %q", tt.raw, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestDecodeJSONObject_AcceptsValidObject(t *testing.T) {
	type payload struct {
		AppID  string `json:"appID"`
		APIKey string `json:"apiKey"`
	}

	var target payload
	if err := decodeJSONObject([]byte(`  {"appID":"A","apiKey":"K"}  `), &target, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.AppID != "A" || target.APIKey != "K" {
		t.Errorf("decoded %+v, want {AppID:A APIKey:K}", target)
	}
}
