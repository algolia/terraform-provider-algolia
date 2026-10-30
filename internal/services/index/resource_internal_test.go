package index

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
			expected: false,
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
			name:     "nested array order differs",
			a:        `{"a":{"b":["x","y"]}}`,
			b:        `{"a":{"b":["y","x"]}}`,
			expected: false,
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

func TestPreservePlannedValuesRenderingContentOrder(t *testing.T) {
	const planned = `{"facetOrdering":{"facets":{"order":["brand","category"]}}}`
	for _, tc := range []struct {
		name string
		api  string
		want string
	}{
		{
			name: "reordered facets remain visible",
			api:  `{"facetOrdering":{"facets":{"order":["category","brand"]}}}`,
			want: `{"facetOrdering":{"facets":{"order":["category","brand"]}}}`,
		},
		{
			name: "formatting alone preserves plan",
			api:  `{ "facetOrdering": {"facets": {"order": ["brand", "category"]}} }`,
			want: planned,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := IndexResourceModel{Advanced: blockObject(t, advancedAttrTypes, AdvancedModel{
				RenderingContent: types.StringValue(planned),
				ResponseFields:   types.ListNull(types.StringType),
				Replicas:         types.ListNull(types.StringType),
			})}
			state := IndexResourceModel{Advanced: blockObject(t, advancedAttrTypes, AdvancedModel{
				RenderingContent: types.StringValue(tc.api),
				ResponseFields:   types.ListNull(types.StringType),
				Replicas:         types.ListNull(types.StringType),
			})}
			if diags := preservePlannedValues(context.Background(), &plan, &state); diags.HasError() {
				t.Fatal(diags)
			}
			got := state.Advanced.Attributes()["rendering_content"].(types.String).ValueString()
			if got != tc.want {
				t.Fatalf("rendering_content = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestPreservePlannedValuesDecompoundedAttributes(t *testing.T) {
	const planned = `{"de":["name","description"],"nl":["title"]}`
	for _, tc := range []struct {
		name string
		api  string
		want string
	}{
		{
			name: "API reorders attribute names",
			api:  `{"nl":["title"],"de":["description","name"]}`,
			want: planned,
		},
		{
			name: "API changes attribute names",
			api:  `{"de":["name","title"],"nl":["title"]}`,
			want: `{"de":["name","title"],"nl":["title"]}`,
		},
		{
			name: "API removes a language",
			api:  `{"de":["name","description"]}`,
			want: `{"de":["name","description"]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			block := func(value string) types.Object {
				return blockObject(t, languagesAttrTypes, LanguagesModel{
					IndexLanguages:            types.ListNull(types.StringType),
					QueryLanguages:            types.ListNull(types.StringType),
					IgnorePluralsLanguages:    types.ListNull(types.StringType),
					RemoveStopWordsLanguages:  types.ListNull(types.StringType),
					AttributesToTransliterate: types.ListNull(types.StringType),
					CamelCaseAttributes:       types.ListNull(types.StringType),
					DecompoundedAttributes:    types.StringValue(value),
				})
			}
			plan := IndexResourceModel{Languages: block(planned)}
			state := IndexResourceModel{Languages: block(tc.api)}
			if diags := preservePlannedValues(context.Background(), &plan, &state); diags.HasError() {
				t.Fatal(diags)
			}
			got := state.Languages.Attributes()["decompounded_attributes"].(types.String).ValueString()
			if got != tc.want {
				t.Fatalf("decompounded_attributes = %s, want %s", got, tc.want)
			}
		})
	}
}
