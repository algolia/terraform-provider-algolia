package composition

import (
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
			a:        `{"tags":["1","2"]}`,
			b:        `{"tags":["2","1"]}`,
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
			a:        `{"conditions":[{"filters":"a"},{"filters":"b"}]}`,
			b:        `{"conditions":[{"filters":"b"},{"filters":"a"}]}`,
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

func TestFlattenRequiredJSONField(t *testing.T) {
	t.Run("adopts API encoding when no prior", func(t *testing.T) {
		value, diags := flattenRequiredJSONField(map[string]string{"a": "1"}, types.StringNull(), "behavior")
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if !jsonSemanticallyEqual(value.ValueString(), `{"a":"1"}`) {
			t.Fatalf("value = %q, want semantically equal to {\"a\":\"1\"}", value.ValueString())
		}
	})

	t.Run("preserves semantically equal prior", func(t *testing.T) {
		prior := `{
			"a": "1"
		}`
		value, diags := flattenRequiredJSONField(map[string]string{"a": "1"}, types.StringValue(prior), "behavior")
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if value.ValueString() != prior {
			t.Fatalf("value = %q, want preserved prior %q", value.ValueString(), prior)
		}
	})

	t.Run("adopts API encoding when prior differs", func(t *testing.T) {
		value, diags := flattenRequiredJSONField(map[string]string{"a": "1"}, types.StringValue(`{"a":"2"}`), "behavior")
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if value.ValueString() == `{"a":"2"}` {
			t.Fatalf("value = %q, want it to be replaced by the API's value", value.ValueString())
		}
	})
}

func TestFlattenOptionalJSONField(t *testing.T) {
	t.Run("nil value with no prior yields null", func(t *testing.T) {
		var value *map[string]string
		got, diags := flattenOptionalJSONField(value, types.StringNull(), "sorting_strategy")
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if !got.IsNull() {
			t.Fatalf("value = %v, want null", got)
		}
	})

	t.Run("nil value preserves a configured prior", func(t *testing.T) {
		var value *map[string]string
		prior := `{"Price (asc)":"products_price_asc"}`
		got, diags := flattenOptionalJSONField(value, types.StringValue(prior), "sorting_strategy")
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if got.ValueString() != prior {
			t.Fatalf("value = %q, want preserved prior %q", got.ValueString(), prior)
		}
	})

	t.Run("non-nil value adopts API encoding when no prior", func(t *testing.T) {
		value := map[string]string{"Price (asc)": "products_price_asc"}
		got, diags := flattenOptionalJSONField(&value, types.StringNull(), "sorting_strategy")
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if !jsonSemanticallyEqual(got.ValueString(), `{"Price (asc)":"products_price_asc"}`) {
			t.Fatalf("value = %q, want semantically equal encoding", got.ValueString())
		}
	})
}

func TestFlattenSliceJSONField(t *testing.T) {
	t.Run("empty slice with no prior yields null", func(t *testing.T) {
		got, diags := flattenSliceJSONField([]string{}, types.StringNull(), "validity")
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if !got.IsNull() {
			t.Fatalf("value = %v, want null", got)
		}
	})

	t.Run("empty slice preserves a configured prior", func(t *testing.T) {
		prior := `[{"from":1,"until":2}]`
		got, diags := flattenSliceJSONField([]string(nil), types.StringValue(prior), "validity")
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if got.ValueString() != prior {
			t.Fatalf("value = %q, want preserved prior %q", got.ValueString(), prior)
		}
	})

	t.Run("non-empty slice adopts API encoding when no prior", func(t *testing.T) {
		got, diags := flattenSliceJSONField([]string{"a", "b"}, types.StringNull(), "validity")
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if !jsonSemanticallyEqual(got.ValueString(), `["a","b"]`) {
			t.Fatalf("value = %q, want semantically equal encoding", got.ValueString())
		}
	})
}
