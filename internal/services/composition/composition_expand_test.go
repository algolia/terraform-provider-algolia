package composition

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExpandComposition(t *testing.T) {
	model := &CompositionResourceModel{
		Name:        types.StringValue("Featured products"),
		Description: types.StringValue("Blends featured and organic results"),
		Behavior: types.StringValue(
			`{"injection":{"main":{"source":{"search":{"index":"products"}}}}}`,
		),
		SortingStrategy: types.StringValue(`{"Price (asc)":"products_price_asc"}`),
	}

	comp, diags := expandComposition("my-composition", model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := comp.GetObjectID(); got != "my-composition" {
		t.Fatalf("objectID = %q, want %q", got, "my-composition")
	}
	if got := comp.GetName(); got != "Featured products" {
		t.Fatalf("name = %q, want %q", got, "Featured products")
	}
	if got := comp.GetDescription(); got != "Blends featured and organic results" {
		t.Fatalf("description = %q, want %q", got, "Blends featured and organic results")
	}
	if comp.Behavior.CompositionInjectionBehavior == nil {
		t.Fatal("expected behavior to decode as an injection behavior")
	}
	if got := comp.Behavior.CompositionInjectionBehavior.Injection.Main.GetSource().InjectionMainSearchSource.Search.Index; got != "products" {
		t.Fatalf("behavior.injection.main.source.search.index = %q, want %q", got, "products")
	}
	if comp.SortingStrategy == nil {
		t.Fatal("expected sorting_strategy to be set")
	}
	if got := (*comp.SortingStrategy)["Price (asc)"]; got != "products_price_asc" {
		t.Fatalf("sorting_strategy[Price (asc)] = %q, want %q", got, "products_price_asc")
	}
}

func TestExpandCompositionMinimal(t *testing.T) {
	model := &CompositionResourceModel{
		Name:            types.StringValue("Minimal"),
		Description:     types.StringNull(),
		Behavior:        types.StringValue(`{"multifeed":{"feeds":{}}}`),
		SortingStrategy: types.StringNull(),
	}

	comp, diags := expandComposition("minimal", model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if comp.Description != nil {
		t.Fatalf("description = %#v, want nil", comp.Description)
	}
	if comp.SortingStrategy != nil {
		t.Fatalf("sorting_strategy = %#v, want nil", comp.SortingStrategy)
	}
	if comp.Behavior.CompositionMultifeedBehavior == nil {
		t.Fatal("expected behavior to decode as a multifeed behavior")
	}
}

func TestExpandCompositionMissingBehavior(t *testing.T) {
	model := &CompositionResourceModel{
		Name:     types.StringValue("Missing behavior"),
		Behavior: types.StringNull(),
	}

	if _, diags := expandComposition("my-composition", model); !diags.HasError() {
		t.Fatal("expected an error for a missing behavior")
	}
}

func TestExpandCompositionInvalidBehaviorJSON(t *testing.T) {
	model := &CompositionResourceModel{
		Name:     types.StringValue("Invalid"),
		Behavior: types.StringValue(`not valid json`),
	}

	if _, diags := expandComposition("my-composition", model); !diags.HasError() {
		t.Fatal("expected an error for invalid behavior JSON")
	}
}

func TestExpandCompositionInvalidSortingStrategyJSON(t *testing.T) {
	model := &CompositionResourceModel{
		Name:            types.StringValue("Invalid"),
		Behavior:        types.StringValue(`{"injection":{"main":{"source":{"search":{"index":"products"}}}}}`),
		SortingStrategy: types.StringValue(`not valid json`),
	}

	if _, diags := expandComposition("my-composition", model); !diags.HasError() {
		t.Fatal("expected an error for invalid sorting_strategy JSON")
	}
}
