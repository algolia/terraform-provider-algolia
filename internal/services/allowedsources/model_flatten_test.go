package allowedsources

import (
	"context"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenSources_Empty(t *testing.T) {
	var model AllowedSourcesResourceModel

	diags := flattenSources(context.Background(), nil, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Source.IsNull() {
		t.Fatal("expected a non-null source set; allowed sources always exist as a singleton")
	}
	if len(model.Source.Elements()) != 0 {
		t.Fatalf("expected an empty set, got %#v", model.Source.Elements())
	}
}

func TestFlattenSources_WithoutDescription(t *testing.T) {
	var model AllowedSourcesResourceModel

	diags := flattenSources(context.Background(), []search.Source{
		*search.NewSource("1.2.3.4"),
	}, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	elements := model.Source.Elements()
	if len(elements) != 1 {
		t.Fatalf("expected 1 element, got %#v", elements)
	}

	obj, ok := elements[0].(types.Object)
	if !ok {
		t.Fatalf("expected element to be an object, got %#v", elements[0])
	}
	attrs := obj.Attributes()
	if v, ok := attrs["source"].(types.String); !ok || v.ValueString() != "1.2.3.4" {
		t.Fatalf("source = %#v, want 1.2.3.4", attrs["source"])
	}
	if d, ok := attrs["description"].(types.String); !ok || !d.IsNull() {
		t.Fatalf("description = %#v, want null", attrs["description"])
	}
}

func TestFlattenSources_WithDescription(t *testing.T) {
	var model AllowedSourcesResourceModel

	diags := flattenSources(context.Background(), []search.Source{
		*search.NewSource("10.0.0.0/24", search.WithSourceDescription("office network")),
	}, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	elements := model.Source.Elements()
	if len(elements) != 1 {
		t.Fatalf("expected 1 element, got %#v", elements)
	}

	obj, ok := elements[0].(types.Object)
	if !ok {
		t.Fatalf("expected element to be an object, got %#v", elements[0])
	}
	attrs := obj.Attributes()
	if v, ok := attrs["description"].(types.String); !ok || v.ValueString() != "office network" {
		t.Fatalf("description = %#v, want %q", attrs["description"], "office network")
	}
}

func TestFlattenSources_Multiple(t *testing.T) {
	var model AllowedSourcesResourceModel

	diags := flattenSources(context.Background(), []search.Source{
		*search.NewSource("1.2.3.4"),
		*search.NewSource("10.0.0.0/24", search.WithSourceDescription("office network")),
	}, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if len(model.Source.Elements()) != 2 {
		t.Fatalf("expected 2 elements, got %#v", model.Source.Elements())
	}
}

// priorSourceSet builds the set a prior plan or state would hold, keyed by
// source value. A null description stands for one the configuration omitted.
func priorSourceSet(descriptions map[string]types.String) types.Set {
	values := make([]attr.Value, 0, len(descriptions))
	for source, description := range descriptions {
		values = append(values, types.ObjectValueMust(sourceAttrTypes, map[string]attr.Value{
			"source":      types.StringValue(source),
			"description": description,
		}))
	}

	return types.SetValueMust(types.ObjectType{AttrTypes: sourceAttrTypes}, values)
}

func flattenedDescription(t *testing.T, set types.Set, source string) types.String {
	t.Helper()

	for _, element := range set.Elements() {
		attrs := element.(types.Object).Attributes()
		if attrs["source"].(types.String).ValueString() == source {
			return attrs["description"].(types.String)
		}
	}

	t.Fatalf("source %q is missing from %s", source, set)

	return types.StringNull()
}

func TestFlattenSources_Descriptions(t *testing.T) {
	tests := []struct {
		name     string
		prior    types.String
		response []search.Source
		want     types.String
	}{
		{
			name: "an empty configured description stays a known empty string",
			// expandSources sends no description for an empty one, so it comes
			// back absent; returning null here would not match the planned value.
			prior:    types.StringValue(""),
			response: []search.Source{*search.NewSource("1.2.3.4")},
			want:     types.StringValue(""),
		},
		{
			name:     "an omitted description stays null",
			prior:    types.StringNull(),
			response: []search.Source{*search.NewSource("1.2.3.4")},
			want:     types.StringNull(),
		},
		{
			name:     "a configured description round trips",
			prior:    types.StringValue("office network"),
			response: []search.Source{*search.NewSource("1.2.3.4", search.WithSourceDescription("office network"))},
			want:     types.StringValue("office network"),
		},
		{
			name:     "drift replaces the configured description",
			prior:    types.StringValue("office network"),
			response: []search.Source{*search.NewSource("1.2.3.4", search.WithSourceDescription("home network"))},
			want:     types.StringValue("home network"),
		},
		{
			name:     "a description that appeared out of band is adopted",
			prior:    types.StringValue(""),
			response: []search.Source{*search.NewSource("1.2.3.4", search.WithSourceDescription("added in the dashboard"))},
			want:     types.StringValue("added in the dashboard"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := AllowedSourcesResourceModel{
				Source: priorSourceSet(map[string]types.String{"1.2.3.4": test.prior}),
			}

			if diags := flattenSources(context.Background(), test.response, &model); diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}

			if got := flattenedDescription(t, model.Source, "1.2.3.4"); !got.Equal(test.want) {
				t.Errorf("description = %s, want %s", got, test.want)
			}
		})
	}
}

func TestFlattenSources_DescriptionsMatchPriorBySourceValue(t *testing.T) {
	// The set is unordered, so a response entry can only be paired with its
	// configured counterpart through the source value. Here the response order is
	// the reverse of the prior's.
	model := AllowedSourcesResourceModel{
		Source: priorSourceSet(map[string]types.String{
			"1.2.3.4":     types.StringValue(""),
			"10.0.0.0/24": types.StringValue("office network"),
		}),
	}

	response := []search.Source{
		*search.NewSource("10.0.0.0/24", search.WithSourceDescription("office network")),
		*search.NewSource("1.2.3.4"),
	}

	if diags := flattenSources(context.Background(), response, &model); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := flattenedDescription(t, model.Source, "1.2.3.4"); !got.Equal(types.StringValue("")) {
		t.Errorf("1.2.3.4 description = %s, want an empty string", got)
	}
	if got := flattenedDescription(t, model.Source, "10.0.0.0/24"); !got.Equal(types.StringValue("office network")) {
		t.Errorf("10.0.0.0/24 description = %s, want %q", got, "office network")
	}
}

func TestFlattenSources_NewEntryWithoutPrior(t *testing.T) {
	// An entry the prior set never had - a fresh source, or any entry at all on
	// import - has no configured description to preserve.
	model := AllowedSourcesResourceModel{
		Source: priorSourceSet(map[string]types.String{"1.2.3.4": types.StringValue("")}),
	}

	response := []search.Source{
		*search.NewSource("1.2.3.4"),
		*search.NewSource("10.0.0.0/24"),
	}

	if diags := flattenSources(context.Background(), response, &model); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := flattenedDescription(t, model.Source, "10.0.0.0/24"); !got.IsNull() {
		t.Errorf("10.0.0.0/24 description = %s, want null", got)
	}
}

func TestExpandFlattenSources_RoundTrip(t *testing.T) {
	sources := []search.Source{
		*search.NewSource("1.2.3.4", search.WithSourceDescription("a")),
		*search.NewSource("10.0.0.0/24"),
	}

	var model AllowedSourcesResourceModel
	if diags := flattenSources(context.Background(), sources, &model); diags.HasError() {
		t.Fatalf("unexpected diagnostics flattening: %v", diags)
	}

	roundTripped, diags := expandSources(context.Background(), model.Source)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics expanding: %v", diags)
	}

	if len(roundTripped) != 2 {
		t.Fatalf("expected 2 sources, got %#v", roundTripped)
	}
	if roundTripped[0].GetSource() != "1.2.3.4" || !roundTripped[0].HasDescription() || roundTripped[0].GetDescription() != "a" {
		t.Fatalf("roundTripped[0] = %#v, want source=1.2.3.4 description=a", roundTripped[0])
	}
	if roundTripped[1].GetSource() != "10.0.0.0/24" || roundTripped[1].HasDescription() {
		t.Fatalf("roundTripped[1] = %#v, want source=10.0.0.0/24 with no description", roundTripped[1])
	}
}
