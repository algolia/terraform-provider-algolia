package allowedsources

import (
	"context"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
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
