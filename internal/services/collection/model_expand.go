package collection

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// expandCreate builds the UpsertRequest for Create: every record in the plan is an addition.
func expandCreate(ctx context.Context, plan *CollectionResourceModel) (*UpsertRequest, diag.Diagnostics) {
	req, diags := baseUpsertRequest(ctx, plan)
	if diags.HasError() {
		return nil, diags
	}

	adds, d := stringListValues(ctx, plan.Records)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	if len(adds) > 0 {
		req.Add = adds
	}

	return req, diags
}

// expandUpdate builds the UpsertRequest for Update: records diff into add/remove deltas.
func expandUpdate(ctx context.Context, state, plan *CollectionResourceModel) (*UpsertRequest, diag.Diagnostics) {
	req, diags := baseUpsertRequest(ctx, plan)
	if diags.HasError() {
		return nil, diags
	}

	stateRecords, d := stringListValues(ctx, state.Records)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	planRecords, d := stringListValues(ctx, plan.Records)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	add, remove := diffRecords(stateRecords, planRecords)
	if len(add) > 0 {
		req.Add = add
	}
	if len(remove) > 0 {
		req.Remove = remove
	}

	// Carry the existing ID so the server updates in place.
	if isKnown(plan.ID) {
		id := plan.ID.ValueString()
		req.ID = &id
	} else if isKnown(state.ID) {
		id := state.ID.ValueString()
		req.ID = &id
	}

	return req, diags
}

// baseUpsertRequest populates the shared scalar + conditions fields.
func baseUpsertRequest(ctx context.Context, plan *CollectionResourceModel) (*UpsertRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	req := &UpsertRequest{}

	if isKnown(plan.Name) {
		v := plan.Name.ValueString()
		req.Name = &v
	}
	if isKnown(plan.IndexName) {
		v := plan.IndexName.ValueString()
		req.IndexName = &v
	}
	if isKnown(plan.Description) {
		v := plan.Description.ValueString()
		req.Description = &v
	}
	if isKnown(plan.Commit) {
		v := plan.Commit.ValueBool()
		req.Commit = &v
	}

	conditions, d := expandConditions(ctx, plan.Conditions)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	req.Conditions = conditions

	return req, diags
}

// expandConditions converts the nested block into a Conditions payload. Returns
// nil when no filter groups are present so we don't send an empty conditions
// object on the wire.
func expandConditions(ctx context.Context, obj types.Object) (*Conditions, diag.Diagnostics) {
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return nil, diags
	}

	var model ConditionsModel
	diags.Append(obj.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	facet, d := expandFilterGroups(ctx, model.FacetFilter)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	numeric, d := expandFilterGroups(ctx, model.NumericFilter)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	if facet == nil && numeric == nil {
		return nil, diags
	}
	c := &Conditions{}
	// Assign conditionally so a typed-nil slice never becomes a non-nil
	// interface value (which would defeat `omitempty` and fail equality
	// checks against untyped nil).
	if facet != nil {
		c.FacetFilters = facet
	}
	if numeric != nil {
		c.NumericFilters = numeric
	}
	return c, diags
}

// expandFilterGroups converts a list of FilterGroupModel into Algolia's wire
// shape: a top-level AND array whose elements are either bare filter strings
// (1 item) or nested OR arrays (N items).
func expandFilterGroups(ctx context.Context, groups types.List) ([]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	if groups.IsNull() || groups.IsUnknown() {
		return nil, diags
	}

	var items []FilterGroupModel
	diags.Append(groups.ElementsAs(ctx, &items, false)...)
	if diags.HasError() {
		return nil, diags
	}

	var result []any
	for _, g := range items {
		filters, d := stringListValues(ctx, g.Filters)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		switch len(filters) {
		case 0:
			// Defensive: the SizeAtLeast(1) validator should prevent this,
			// but skip to avoid emitting an empty OR-group on the wire.
			continue
		case 1:
			result = append(result, filters[0])
		default:
			result = append(result, filters)
		}
	}

	if len(result) == 0 {
		return nil, diags
	}
	return result, diags
}

// diffRecords computes the set-difference add/remove lists between prior and
// desired record IDs. Both returned slices are sorted for stable wire output.
func diffRecords(prior, desired []string) (add, remove []string) {
	priorSet := make(map[string]struct{}, len(prior))
	for _, id := range prior {
		priorSet[id] = struct{}{}
	}
	desiredSet := make(map[string]struct{}, len(desired))
	for _, id := range desired {
		desiredSet[id] = struct{}{}
	}

	for id := range desiredSet {
		if _, ok := priorSet[id]; !ok {
			add = append(add, id)
		}
	}
	for id := range priorSet {
		if _, ok := desiredSet[id]; !ok {
			remove = append(remove, id)
		}
	}

	sort.Strings(add)
	sort.Strings(remove)
	return add, remove
}

// stringListValues extracts a Go []string from a types.List, treating null/unknown as empty.
func stringListValues(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}
	var out []string
	diags.Append(list.ElementsAs(ctx, &out, false)...)
	return out, diags
}

// isKnown returns true if a Terraform value is neither null nor unknown.
func isKnown(v interface{ IsNull() bool; IsUnknown() bool }) bool {
	return !v.IsNull() && !v.IsUnknown()
}
