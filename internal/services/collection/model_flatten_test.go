package collection

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestFlattenCollectionResponse_PopulatesScalarsAndRecords(t *testing.T) {
	ctx := context.Background()
	status := "COMMITTED"
	updatedAt := "2026-04-01T10:00:00Z"
	description := "Curated summer picks"

	resp := &CollectionResponse{
		ID:          "coll-42",
		Name:        "Summer",
		IndexName:   "products",
		Description: &description,
		CreatedAt:   "2026-03-01T00:00:00Z",
		UpdatedAt:   &updatedAt,
		Status:      &status,
		Records:     []CollectionRecord{{ObjectID: "a"}, {ObjectID: "b"}},
	}

	model := &CollectionResourceModel{}
	diags := flattenCollectionResponse(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}

	if model.ID.ValueString() != "coll-42" {
		t.Errorf("id: got %q", model.ID.ValueString())
	}
	if model.Name.ValueString() != "Summer" {
		t.Errorf("name: got %q", model.Name.ValueString())
	}
	if model.Description.ValueString() != description {
		t.Errorf("description: got %q", model.Description.ValueString())
	}
	if model.Records.IsNull() {
		t.Fatal("expected records to be populated")
	}

	var got []string
	diags.Append(model.Records.ElementsAs(ctx, &got, false)...)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("records: got %v", got)
	}
}

func TestFlattenCollectionResponse_EmptyRecordsBecomeNullList(t *testing.T) {
	ctx := context.Background()
	resp := &CollectionResponse{
		ID:        "coll-empty",
		Name:      "Empty",
		IndexName: "products",
		CreatedAt: "2026-04-01T00:00:00Z",
	}

	model := &CollectionResourceModel{}
	diags := flattenCollectionResponse(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}

	if !model.Records.IsNull() {
		t.Errorf("expected records to be null, got %#v", model.Records)
	}
	if !model.Description.IsNull() {
		t.Errorf("expected description to be null, got %#v", model.Description)
	}
	if !model.Status.IsNull() {
		t.Errorf("expected status to be null, got %#v", model.Status)
	}
}

// readConditionsModel extracts the nested FacetFilter/NumericFilter groups
// from a flattened conditions Object so tests can assert against them.
func readConditionsModel(t *testing.T, ctx context.Context, obj types.Object) (facet, numeric [][]string) {
	t.Helper()
	if obj.IsNull() {
		return nil, nil
	}
	var cond ConditionsModel
	diags := obj.As(ctx, &cond, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		t.Fatalf("conditions.As: %v", diags.Errors())
	}

	read := func(list types.List) [][]string {
		if list.IsNull() || list.IsUnknown() {
			return nil
		}
		var items []FilterGroupModel
		diags := list.ElementsAs(ctx, &items, false)
		if diags.HasError() {
			t.Fatalf("list.ElementsAs: %v", diags.Errors())
		}
		out := make([][]string, 0, len(items))
		for _, g := range items {
			var s []string
			diags := g.Filters.ElementsAs(ctx, &s, false)
			if diags.HasError() {
				t.Fatalf("filters.ElementsAs: %v", diags.Errors())
			}
			out = append(out, s)
		}
		return out
	}

	return read(cond.FacetFilter), read(cond.NumericFilter)
}

func TestFlattenConditions_StringsBecomeSingleFilterGroups(t *testing.T) {
	ctx := context.Background()
	resp := &CollectionResponse{
		ID:         "c",
		Name:       "n",
		IndexName:  "i",
		CreatedAt:  "t",
		Conditions: &Conditions{FacetFilters: []any{"brand:Apple"}},
	}
	model := &CollectionResourceModel{}
	diags := flattenCollectionResponse(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}
	facet, numeric := readConditionsModel(t, ctx, model.Conditions)
	if len(facet) != 1 || len(facet[0]) != 1 || facet[0][0] != "brand:Apple" {
		t.Errorf("facet: got %v", facet)
	}
	if numeric != nil {
		t.Errorf("numeric should be nil, got %v", numeric)
	}
}

func TestFlattenConditions_ArraysBecomeOrGroups(t *testing.T) {
	ctx := context.Background()
	resp := &CollectionResponse{
		ID:         "c",
		Name:       "n",
		IndexName:  "i",
		CreatedAt:  "t",
		Conditions: &Conditions{FacetFilters: []any{[]any{"category:Phone", "category:Tablet"}}},
	}
	model := &CollectionResourceModel{}
	diags := flattenCollectionResponse(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}
	facet, _ := readConditionsModel(t, ctx, model.Conditions)
	if len(facet) != 1 {
		t.Fatalf("expected 1 group, got %d", len(facet))
	}
	if len(facet[0]) != 2 || facet[0][0] != "category:Phone" || facet[0][1] != "category:Tablet" {
		t.Errorf("or group: got %v", facet[0])
	}
}

func TestFlattenConditions_MixedShape(t *testing.T) {
	ctx := context.Background()
	resp := &CollectionResponse{
		ID:        "c",
		Name:      "n",
		IndexName: "i",
		CreatedAt: "t",
		Conditions: &Conditions{
			FacetFilters:   []any{"brand:Apple", []any{"category:Phone", "category:Tablet"}},
			NumericFilters: []any{[]any{"price<100", "price>1000"}},
		},
	}
	model := &CollectionResourceModel{}
	diags := flattenCollectionResponse(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}
	facet, numeric := readConditionsModel(t, ctx, model.Conditions)
	if len(facet) != 2 || facet[0][0] != "brand:Apple" || facet[1][0] != "category:Phone" {
		t.Errorf("facet shape wrong: %v", facet)
	}
	if len(numeric) != 1 || numeric[0][0] != "price<100" || numeric[0][1] != "price>1000" {
		t.Errorf("numeric shape wrong: %v", numeric)
	}
}

func TestFlattenConditions_NilProducesNullObject(t *testing.T) {
	ctx := context.Background()
	obj, diags := flattenConditions(ctx, nil)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}
	if !obj.IsNull() {
		t.Errorf("expected null object, got %#v", obj)
	}
}

func TestFlattenConditions_DefensivelySkipsUnknownShapes(t *testing.T) {
	ctx := context.Background()
	resp := &CollectionResponse{
		ID:        "c",
		Name:      "n",
		IndexName: "i",
		CreatedAt: "t",
		Conditions: &Conditions{
			// Mix a string, an OR-group, a non-string element in an inner
			// group, and an unrecognized top-level shape (int). Only the
			// valid string + OR-group should survive.
			FacetFilters: []any{
				"brand:Apple",
				[]any{"good", 42, "also-good"},
				42,
			},
		},
	}
	model := &CollectionResourceModel{}
	diags := flattenCollectionResponse(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}
	facet, _ := readConditionsModel(t, ctx, model.Conditions)
	if len(facet) != 2 {
		t.Fatalf("expected 2 groups (string + filtered OR), got %d: %v", len(facet), facet)
	}
	if facet[0][0] != "brand:Apple" {
		t.Errorf("first group: got %v", facet[0])
	}
	if len(facet[1]) != 2 || facet[1][0] != "good" || facet[1][1] != "also-good" {
		t.Errorf("second group should drop the int: got %v", facet[1])
	}
}

func TestFlattenStringList_EmptyIsNull(t *testing.T) {
	ctx := context.Background()
	if !flattenStringList(ctx, nil).IsNull() {
		t.Error("expected nil slice to produce null list")
	}
	if !flattenStringList(ctx, []string{}).IsNull() {
		t.Error("expected empty slice to produce null list")
	}

	list := flattenStringList(ctx, []string{"x"})
	if list.IsNull() {
		t.Error("expected non-empty slice to produce a list value")
	}
	if list.ElementType(ctx) != types.StringType {
		t.Errorf("unexpected element type %v", list.ElementType(ctx))
	}
}
