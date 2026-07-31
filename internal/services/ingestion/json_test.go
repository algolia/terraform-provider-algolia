package ingestion

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
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
			// Strict about nulls on purpose: expandSourceUpdate uses this to decide
			// whether to send `input` in a PATCH at all, so a difference dismissed
			// here is a write that never happens. The looser comparison lives in
			// TestJSONEqualIgnoringAPINulls.
			name:     "key present with a null differs from the key being absent",
			a:        `{"url":"https://example.com/data.csv"}`,
			b:        `{"url":"https://example.com/data.csv","description":null}`,
			expected: false,
		},
		{
			name:     "null against a real value differs",
			a:        `{"condition":null}`,
			b:        `{"condition":"record.price > 0"}`,
			expected: false,
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

// TestSourceInputUnchanged_TreatsNullKeysAsAChange is the regression guard for the
// most damaging way to get this wrong. Unlike every other use of a JSON
// comparison in this package, this one decides whether `input` is included in the
// PATCH at all - so a difference dismissed here is a write that never reaches
// Algolia, while state records the new value and no later refresh disagrees. A
// source's `input` holds free-form, operator-authored configuration (a docker
// source's `configuration` map, for one), where a null is the operator's and not
// the API's.
func TestSourceInputUnchanged_TreatsNullKeysAsAChange(t *testing.T) {
	cases := []struct {
		name    string
		planned string
		prior   string
		want    bool
	}{
		{
			name:    "operator removed a null-valued key",
			planned: `{"start_date":"2024-01-01"}`,
			prior:   `{"start_date":"2024-01-01","cursor_field":null}`,
			want:    false,
		},
		{
			name:    "operator added a key set to null",
			planned: `{"start_date":"2024-01-01","cursor_field":null}`,
			prior:   `{"start_date":"2024-01-01"}`,
			want:    false,
		},
		{
			name:    "genuinely unchanged, re-encoded",
			planned: `{"start_date":"2024-01-01","cursor_field":"updated_at"}`,
			prior:   `{"cursor_field":"updated_at","start_date":"2024-01-01"}`,
			want:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sourceInputUnchanged(types.StringValue(tc.planned), types.StringValue(tc.prior))
			if got != tc.want {
				t.Errorf("sourceInputUnchanged() = %v, want %v - a false positive here silently skips the update", got, tc.want)
			}
		})
	}
}

// TestJSONEqualIgnoringAPINulls covers the one-directional tolerance used when
// deciding whether to adopt an API response into state. It must accept a null the
// API added, and nothing else - in particular it must never make two different
// configurations look the same, and it must never be reachable from a decision
// about whether to send a write.
func TestJSONEqualIgnoringAPINulls(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		apiValue   string
		expected   bool
	}{
		{
			// The case this exists for: the API adds a `condition` to every no-code
			// step, which nobody configured.
			name:       "api added a null key",
			configured: `{"steps":[{"id":"s1","configuration":{"action":{"kind":"addAttribute"}}}]}`,
			apiValue:   `{"steps":[{"id":"s1","configuration":{"action":{"kind":"addAttribute"},"condition":null}}]}`,
			expected:   true,
		},
		{
			name:       "api added a null key at the top level",
			configured: `{"url":"https://example.com/data.csv"}`,
			apiValue:   `{"url":"https://example.com/data.csv","description":null}`,
			expected:   true,
		},
		{
			// Two different documents. Collapsing both sides would make these equal,
			// which is the failure mode this direction is designed to avoid.
			name:       "different keys, both null",
			configured: `{"a":null}`,
			apiValue:   `{"b":null}`,
			expected:   false,
		},
		{
			// The operator wrote a null and the API dropped it. That is drift, and
			// stripping the configured side would have hidden it.
			name:       "configured null the api does not report",
			configured: `{"cursorField":null}`,
			apiValue:   `{}`,
			expected:   false,
		},
		{
			name:       "api replaced a value with null",
			configured: `{"cursorField":"updated_at"}`,
			apiValue:   `{"cursorField":null}`,
			expected:   false,
		},
		{
			name:       "genuinely different values",
			configured: `{"url":"https://example.com/old.csv"}`,
			apiValue:   `{"url":"https://example.com/new.csv"}`,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsonEqualIgnoringAPINulls(tt.configured, tt.apiValue)
			if got != tt.expected {
				t.Errorf("jsonEqualIgnoringAPINulls(%q, %q) = %v, want %v", tt.configured, tt.apiValue, got, tt.expected)
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
