package collection

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDiffRecords(t *testing.T) {
	testCases := []struct {
		name       string
		prior      []string
		desired    []string
		wantAdd    []string
		wantRemove []string
	}{
		{
			name:       "initial create — all adds",
			prior:      nil,
			desired:    []string{"b", "a"},
			wantAdd:    []string{"a", "b"},
			wantRemove: nil,
		},
		{
			name:       "full removal",
			prior:      []string{"a", "b"},
			desired:    nil,
			wantAdd:    nil,
			wantRemove: []string{"a", "b"},
		},
		{
			name:       "identical sets produce no deltas",
			prior:      []string{"a", "b"},
			desired:    []string{"a", "b"},
			wantAdd:    nil,
			wantRemove: nil,
		},
		{
			name:       "mixed add and remove",
			prior:      []string{"a", "b", "c"},
			desired:    []string{"b", "c", "d"},
			wantAdd:    []string{"d"},
			wantRemove: []string{"a"},
		},
		{
			name:       "ordering is deterministic regardless of input order",
			prior:      []string{"z", "y"},
			desired:    []string{"b", "a", "c"},
			wantAdd:    []string{"a", "b", "c"},
			wantRemove: []string{"y", "z"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			add, remove := diffRecords(tc.prior, tc.desired)
			if !reflect.DeepEqual(add, tc.wantAdd) {
				t.Errorf("add: got %v, want %v", add, tc.wantAdd)
			}
			if !reflect.DeepEqual(remove, tc.wantRemove) {
				t.Errorf("remove: got %v, want %v", remove, tc.wantRemove)
			}
		})
	}
}

func TestExpandCreate_PopulatesFields(t *testing.T) {
	ctx := context.Background()

	records, _ := types.ListValueFrom(ctx, types.StringType, []string{"obj-1", "obj-2"})

	plan := &CollectionResourceModel{
		Name:       types.StringValue("Summer"),
		IndexName:  types.StringValue("products"),
		Records:    records,
		Commit:     types.BoolValue(true),
		Conditions: types.ObjectNull(conditionsAttrTypes),
	}

	req, diags := expandCreate(ctx, plan)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}

	if req.Name == nil || *req.Name != "Summer" {
		t.Errorf("name: got %v, want Summer", req.Name)
	}
	if req.IndexName == nil || *req.IndexName != "products" {
		t.Errorf("index_name: got %v, want products", req.IndexName)
	}
	if req.Commit == nil || *req.Commit != true {
		t.Errorf("commit: got %v, want true", req.Commit)
	}
	if !reflect.DeepEqual(req.Add, []string{"obj-1", "obj-2"}) {
		t.Errorf("add: got %v, want [obj-1 obj-2]", req.Add)
	}
	if req.Remove != nil {
		t.Errorf("remove: expected nil on create, got %v", req.Remove)
	}
	if req.Conditions != nil {
		t.Errorf("conditions: expected nil when block absent, got %#v", req.Conditions)
	}
}

func TestExpandUpdate_DiffsRecordsAndCarriesID(t *testing.T) {
	ctx := context.Background()

	stateList, _ := types.ListValueFrom(ctx, types.StringType, []string{"a", "b", "c"})
	planList, _ := types.ListValueFrom(ctx, types.StringType, []string{"b", "c", "d"})

	state := &CollectionResourceModel{
		ID:         types.StringValue("coll-123"),
		Records:    stateList,
		Conditions: types.ObjectNull(conditionsAttrTypes),
	}
	plan := &CollectionResourceModel{
		ID:         types.StringValue("coll-123"),
		Name:       types.StringValue("Updated"),
		IndexName:  types.StringValue("products"),
		Records:    planList,
		Commit:     types.BoolValue(true),
		Conditions: types.ObjectNull(conditionsAttrTypes),
	}

	req, diags := expandUpdate(ctx, state, plan)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}

	if req.ID == nil || *req.ID != "coll-123" {
		t.Errorf("id: got %v, want coll-123", req.ID)
	}
	if !reflect.DeepEqual(req.Add, []string{"d"}) {
		t.Errorf("add: got %v, want [d]", req.Add)
	}
	if !reflect.DeepEqual(req.Remove, []string{"a"}) {
		t.Errorf("remove: got %v, want [a]", req.Remove)
	}
}

// buildConditionsObject constructs a ConditionsModel object with the given
// filter-group slices for testing. Empty slices become null lists.
func buildConditionsObject(t *testing.T, ctx context.Context, facet, numeric [][]string) types.Object {
	t.Helper()

	toList := func(groups [][]string) types.List {
		if len(groups) == 0 {
			return types.ListNull(filterGroupObjectType)
		}
		models := make([]FilterGroupModel, 0, len(groups))
		for _, g := range groups {
			lst, diags := types.ListValueFrom(ctx, types.StringType, g)
			if diags.HasError() {
				t.Fatalf("inner list setup: %v", diags.Errors())
			}
			models = append(models, FilterGroupModel{Filters: lst})
		}
		list, diags := types.ListValueFrom(ctx, filterGroupObjectType, models)
		if diags.HasError() {
			t.Fatalf("group list setup: %v", diags.Errors())
		}
		return list
	}

	obj, diags := types.ObjectValueFrom(ctx, conditionsAttrTypes, &ConditionsModel{
		FacetFilter:   toList(facet),
		NumericFilter: toList(numeric),
	})
	if diags.HasError() {
		t.Fatalf("conditions object setup: %v", diags.Errors())
	}
	return obj
}

func TestExpandConditions_SingleAndTerm(t *testing.T) {
	ctx := context.Background()
	obj := buildConditionsObject(t, ctx, [][]string{{"brand:Apple"}}, nil)

	got, diags := expandConditions(ctx, obj)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}
	if got == nil {
		t.Fatal("expected non-nil conditions")
	}
	want := []any{"brand:Apple"}
	if !reflect.DeepEqual(got.FacetFilters, want) {
		t.Errorf("facet: got %#v, want %#v", got.FacetFilters, want)
	}
	if got.NumericFilters != nil {
		t.Errorf("numeric: expected nil, got %#v", got.NumericFilters)
	}
}

func TestExpandConditions_OrGroup(t *testing.T) {
	ctx := context.Background()
	obj := buildConditionsObject(t, ctx, [][]string{{"category:Phone", "category:Tablet"}}, nil)

	got, diags := expandConditions(ctx, obj)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}
	want := []any{[]string{"category:Phone", "category:Tablet"}}
	if !reflect.DeepEqual(got.FacetFilters, want) {
		t.Errorf("facet: got %#v, want %#v", got.FacetFilters, want)
	}
}

func TestExpandConditions_MixedAndOr(t *testing.T) {
	ctx := context.Background()
	obj := buildConditionsObject(
		t, ctx,
		[][]string{
			{"brand:Apple"},
			{"category:Phone", "category:Tablet"},
			{"color:red", "color:blue"},
		},
		nil,
	)

	got, diags := expandConditions(ctx, obj)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}
	want := []any{
		"brand:Apple",
		[]string{"category:Phone", "category:Tablet"},
		[]string{"color:red", "color:blue"},
	}
	if !reflect.DeepEqual(got.FacetFilters, want) {
		t.Errorf("facet: got %#v, want %#v", got.FacetFilters, want)
	}
}

func TestExpandConditions_BothFilterTypes(t *testing.T) {
	ctx := context.Background()
	obj := buildConditionsObject(
		t, ctx,
		[][]string{{"brand:Apple"}},
		[][]string{{"price<100", "price>1000"}},
	)

	got, diags := expandConditions(ctx, obj)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}
	if !reflect.DeepEqual(got.FacetFilters, []any{"brand:Apple"}) {
		t.Errorf("facet mismatch: %#v", got.FacetFilters)
	}
	if !reflect.DeepEqual(got.NumericFilters, []any{[]string{"price<100", "price>1000"}}) {
		t.Errorf("numeric mismatch: %#v", got.NumericFilters)
	}
}

func TestExpandConditions_EmptyBlocksProduceNil(t *testing.T) {
	ctx := context.Background()
	obj := buildConditionsObject(t, ctx, nil, nil)

	got, diags := expandConditions(ctx, obj)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}
	if got != nil {
		t.Errorf("expected nil, got %#v", got)
	}
}

func TestExpandConditions_NullObjectProducesNil(t *testing.T) {
	ctx := context.Background()
	got, diags := expandConditions(ctx, types.ObjectNull(conditionsAttrTypes))
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}
	if got != nil {
		t.Errorf("expected nil, got %#v", got)
	}
}
