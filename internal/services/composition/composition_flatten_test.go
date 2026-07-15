package composition

import (
	"encoding/json"
	"testing"

	compositionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/composition"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func newTestComposition() *compositionapi.Composition {
	comp := compositionapi.NewEmptyComposition()
	comp.ObjectID = "my-composition"
	comp.Name = "Featured products"
	description := "Blends featured and organic results"
	comp.Description = &description
	comp.Behavior = *compositionapi.CompositionInjectionBehaviorAsCompositionBehavior(
		compositionapi.NewCompositionInjectionBehavior(
			*compositionapi.NewInjection(
				*compositionapi.NewInjectionMain(
					compositionapi.WithInjectionMainSource(
						*compositionapi.InjectionMainSearchSourceAsInjectionMainSource(
							compositionapi.NewInjectionMainSearchSource(*compositionapi.NewMainSearch("products")),
						),
					),
				),
			),
		),
	)
	sortingStrategy := map[string]string{"Price (asc)": "products_price_asc"}
	comp.SortingStrategy = &sortingStrategy

	return comp
}

func TestFlattenComposition_AdoptsAPIEncodingWhenNoPrior(t *testing.T) {
	comp := newTestComposition()

	var model CompositionResourceModel
	diags := flattenComposition(comp, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.ID.ValueString(); got != "my-composition" {
		t.Fatalf("id = %q, want %q", got, "my-composition")
	}
	if got := model.ObjectID.ValueString(); got != "my-composition" {
		t.Fatalf("object_id = %q, want %q", got, "my-composition")
	}
	if got := model.Name.ValueString(); got != "Featured products" {
		t.Fatalf("name = %q, want %q", got, "Featured products")
	}
	if got := model.Description.ValueString(); got != "Blends featured and organic results" {
		t.Fatalf("description = %q, want %q", got, "Blends featured and organic results")
	}
	if model.Behavior.IsNull() {
		t.Fatal("behavior should be set")
	}
	if model.SortingStrategy.IsNull() {
		t.Fatal("sorting_strategy should be set")
	}

	var decodedBehavior map[string]any
	if err := json.Unmarshal([]byte(model.Behavior.ValueString()), &decodedBehavior); err != nil {
		t.Fatalf("behavior is not valid JSON: %v", err)
	}
	if _, ok := decodedBehavior["injection"]; !ok {
		t.Fatalf("behavior = %v, want an injection key", decodedBehavior)
	}
}

func TestFlattenComposition_PreservesSemanticallyEqualPrior(t *testing.T) {
	comp := newTestComposition()

	priorBehavior := `{
		"injection": {
			"main": {
				"source": {
					"search": {
						"index": "products"
					}
				}
			}
		}
	}`

	model := CompositionResourceModel{
		Behavior: types.StringValue(priorBehavior),
	}

	diags := flattenComposition(comp, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.Behavior.ValueString(); got != priorBehavior {
		t.Fatalf("behavior = %q, want the preserved prior value %q", got, priorBehavior)
	}
}

func TestFlattenComposition_AdoptsAPIEncodingWhenPriorDiffers(t *testing.T) {
	comp := newTestComposition()

	model := CompositionResourceModel{
		Behavior: types.StringValue(`{"injection":{"main":{"source":{"search":{"index":"other"}}}}}`),
	}

	diags := flattenComposition(comp, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.Behavior.ValueString(); got == `{"injection":{"main":{"source":{"search":{"index":"other"}}}}}` {
		t.Fatalf("behavior = %q, want it to be replaced by the API's value", got)
	}
}

func TestFlattenComposition_NilDescriptionAndSortingStrategy(t *testing.T) {
	comp := newTestComposition()
	comp.Description = nil
	comp.SortingStrategy = nil

	t.Run("no prior configuration yields null", func(t *testing.T) {
		var model CompositionResourceModel
		diags := flattenComposition(comp, &model)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if !model.Description.IsNull() {
			t.Fatalf("description = %v, want null", model.Description.ValueString())
		}
		if !model.SortingStrategy.IsNull() {
			t.Fatalf("sorting_strategy = %v, want null", model.SortingStrategy.ValueString())
		}
	})

	t.Run("a configured prior sorting_strategy is preserved even when the API returns nothing", func(t *testing.T) {
		model := CompositionResourceModel{
			SortingStrategy: types.StringValue(`{"Price (asc)":"products_price_asc"}`),
		}
		diags := flattenComposition(comp, &model)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if got := model.SortingStrategy.ValueString(); got != `{"Price (asc)":"products_price_asc"}` {
			t.Fatalf("sorting_strategy = %q, want the preserved prior value", got)
		}
	})
}
