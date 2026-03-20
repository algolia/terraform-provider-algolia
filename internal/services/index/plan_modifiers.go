package index

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// useStateForKnown returns a plan modifier that copies the prior state value
// into the plan only when the state value is not null. Unlike UseStateForUnknown,
// this avoids copying a null state into the plan when an enclosing block is
// newly created, which would cause "was null, but now <value>" errors when the
// API returns a default value.

func useStateForKnownString() planmodifier.String {
	return useStateForKnownStringModifier{}
}

type useStateForKnownStringModifier struct{}

func (m useStateForKnownStringModifier) Description(_ context.Context) string {
	return "Use prior state for unknown values, except when state is null."
}

func (m useStateForKnownStringModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m useStateForKnownStringModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.PlanValue.IsUnknown() || req.StateValue.IsNull() {
		return
	}
	resp.PlanValue = req.StateValue
}

func useStateForKnownBool() planmodifier.Bool {
	return useStateForKnownBoolModifier{}
}

type useStateForKnownBoolModifier struct{}

func (m useStateForKnownBoolModifier) Description(_ context.Context) string {
	return "Use prior state for unknown values, except when state is null."
}

func (m useStateForKnownBoolModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m useStateForKnownBoolModifier) PlanModifyBool(_ context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if !req.PlanValue.IsUnknown() || req.StateValue.IsNull() {
		return
	}
	resp.PlanValue = req.StateValue
}

func useStateForKnownList() planmodifier.List {
	return useStateForKnownListModifier{}
}

type useStateForKnownListModifier struct{}

func (m useStateForKnownListModifier) Description(_ context.Context) string {
	return "Use prior state for unknown values, except when state is null."
}

func (m useStateForKnownListModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m useStateForKnownListModifier) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if !req.PlanValue.IsUnknown() || req.StateValue.IsNull() {
		return
	}
	resp.PlanValue = req.StateValue
}

func useStateForKnownInt64() planmodifier.Int64 {
	return useStateForKnownInt64Modifier{}
}

type useStateForKnownInt64Modifier struct{}

func (m useStateForKnownInt64Modifier) Description(_ context.Context) string {
	return "Use prior state for unknown values, except when state is null."
}

func (m useStateForKnownInt64Modifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m useStateForKnownInt64Modifier) PlanModifyInt64(_ context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if !req.PlanValue.IsUnknown() || req.StateValue.IsNull() {
		return
	}
	resp.PlanValue = req.StateValue
}
