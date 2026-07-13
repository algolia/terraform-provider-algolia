package collection

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestCountFilters_SumsAcrossGroups(t *testing.T) {
	ctx := context.Background()

	groups, _ := types.ListValueFrom(ctx, filterGroupObjectType, []FilterGroupModel{
		{Filters: mustList(t, ctx, "a")},
		{Filters: mustList(t, ctx, "b", "c")},
		{Filters: mustList(t, ctx, "d", "e", "f")},
	})

	got := countFilters(ctx, groups)
	if got != 6 {
		t.Errorf("got %d, want 6", got)
	}
}

func TestCountFilters_NullOrUnknownIsZero(t *testing.T) {
	ctx := context.Background()
	if n := countFilters(ctx, types.ListNull(filterGroupObjectType)); n != 0 {
		t.Errorf("null: got %d", n)
	}
	if n := countFilters(ctx, types.ListUnknown(filterGroupObjectType)); n != 0 {
		t.Errorf("unknown: got %d", n)
	}
}

// buildModelWithFilterCounts creates a CollectionResourceModel whose conditions
// block contains `facetCount` + `numericCount` total filter entries, split
// across single-item groups (simplest shape exercising the counter).
func buildModelWithFilterCounts(t *testing.T, ctx context.Context, facetCount, numericCount int) CollectionResourceModel {
	t.Helper()

	buildGroups := func(n int, prefix string) types.List {
		if n == 0 {
			return types.ListNull(filterGroupObjectType)
		}
		groups := make([]FilterGroupModel, 0, n)
		for i := 0; i < n; i++ {
			filter := prefix + ":v" + itoa(i)
			groups = append(groups, FilterGroupModel{Filters: mustList(t, ctx, filter)})
		}
		list, diags := types.ListValueFrom(ctx, filterGroupObjectType, groups)
		if diags.HasError() {
			t.Fatalf("list setup: %v", diags.Errors())
		}
		return list
	}

	condObj, diags := types.ObjectValueFrom(ctx, conditionsAttrTypes, &ConditionsModel{
		FacetFilter:   buildGroups(facetCount, "brand"),
		NumericFilter: buildGroups(numericCount, "price"),
	})
	if diags.HasError() {
		t.Fatalf("conditions setup: %v", diags.Errors())
	}

	return CollectionResourceModel{
		Name:       types.StringValue("n"),
		IndexName:  types.StringValue("i"),
		Conditions: condObj,
	}
}

func TestMaxFiltersValidator_CountingBoundaries(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name    string
		facet   int
		numeric int
	}{
		{"under limit", 10, 5},
		{"at limit exactly", MaxFilters, 0},
		{"split evenly at limit", MaxFilters / 2, MaxFilters / 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := buildModelWithFilterCounts(t, ctx, tc.facet, tc.numeric)
			var cond ConditionsModel
			if diags := model.Conditions.As(ctx, &cond, basetypes.ObjectAsOptions{}); diags.HasError() {
				t.Fatalf("decode: %v", diags.Errors())
			}
			n := countFilters(ctx, cond.FacetFilter) + countFilters(ctx, cond.NumericFilter)
			if n > MaxFilters {
				t.Errorf("count %d should be ≤ %d for %q", n, MaxFilters, tc.name)
			}
		})
	}
}

func TestMaxFiltersValidator_OverLimitCountDetected(t *testing.T) {
	ctx := context.Background()
	model := buildModelWithFilterCounts(t, ctx, MaxFilters, 1) // 51 total
	var cond ConditionsModel
	if diags := model.Conditions.As(ctx, &cond, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatalf("decode: %v", diags.Errors())
	}
	n := countFilters(ctx, cond.FacetFilter) + countFilters(ctx, cond.NumericFilter)
	if n <= MaxFilters {
		t.Fatalf("expected count > %d, got %d", MaxFilters, n)
	}
}

func TestPreserveOnUnsetString_NoopOnCreate(t *testing.T) {
	// No prior state (create path) — modifier must not touch anything.
	req := planmodifier.StringRequest{
		Path:        path.Root("description"),
		ConfigValue: types.StringNull(),
		StateValue:  types.StringNull(),
	}
	resp := &planmodifier.StringResponse{PlanValue: types.StringNull()}

	PreserveOnUnsetString("reason").PlanModifyString(context.Background(), req, resp)

	if !resp.PlanValue.IsNull() {
		t.Errorf("expected plan to remain null, got %#v", resp.PlanValue)
	}
	if resp.Diagnostics.HasError() || len(resp.Diagnostics.Warnings()) > 0 {
		t.Errorf("expected no diagnostics, got %v", resp.Diagnostics)
	}
}

func TestPreserveOnUnsetString_NoopWhenConfigSet(t *testing.T) {
	// User explicitly set a value — honor it, whether it's a change or not.
	req := planmodifier.StringRequest{
		Path:        path.Root("description"),
		ConfigValue: types.StringValue("new-value"),
		StateValue:  types.StringValue("old-value"),
	}
	resp := &planmodifier.StringResponse{PlanValue: types.StringValue("new-value")}

	PreserveOnUnsetString("reason").PlanModifyString(context.Background(), req, resp)

	if resp.PlanValue.ValueString() != "new-value" {
		t.Errorf("expected plan to remain 'new-value', got %q", resp.PlanValue.ValueString())
	}
	if len(resp.Diagnostics.Warnings()) > 0 {
		t.Errorf("expected no warnings when user sets value, got %v", resp.Diagnostics.Warnings())
	}
}

func TestPreserveOnUnsetString_PreservesStateAndWarns(t *testing.T) {
	// User removed the attribute — state has a value — modifier must pin
	// plan to state and emit a warning.
	req := planmodifier.StringRequest{
		Path:        path.Root("description"),
		ConfigValue: types.StringNull(),
		StateValue:  types.StringValue("kept-value"),
	}
	resp := &planmodifier.StringResponse{PlanValue: types.StringNull()}

	PreserveOnUnsetString("reason for the limitation.").PlanModifyString(context.Background(), req, resp)

	if resp.PlanValue.ValueString() != "kept-value" {
		t.Errorf("expected plan pinned to state 'kept-value', got %q", resp.PlanValue.ValueString())
	}
	warnings := resp.Diagnostics.Warnings()
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if warnings[0].Summary() != "Attribute cannot be cleared via update" {
		t.Errorf("unexpected warning summary: %q", warnings[0].Summary())
	}
}

func TestMaxFiltersValidator_DescriptionMentionsCap(t *testing.T) {
	v := maxFiltersValidator{}
	got := v.Description(context.Background())
	if got == "" {
		t.Error("description must not be empty")
	}
	mdGot := v.MarkdownDescription(context.Background())
	if mdGot != got {
		t.Errorf("markdown description should match plain: %q vs %q", mdGot, got)
	}
}

// --- helpers ---

func mustList(t *testing.T, ctx context.Context, values ...string) types.List {
	t.Helper()
	list, diags := types.ListValueFrom(ctx, types.StringType, values)
	if diags.HasError() {
		t.Fatalf("mustList: %v", diags.Errors())
	}
	return list
}

func itoa(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = digits[i%10]
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
