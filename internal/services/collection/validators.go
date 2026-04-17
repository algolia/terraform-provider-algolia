package collection

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// MaxFilters caps the combined count of facet + numeric filter entries across a
// single collection. Mirrors the server-side MAX_FILTERS constant.
const MaxFilters = 50

// facetFilterRegex matches Algolia's <attribute>:<value> facet filter grammar.
// The [^\\] lookahead-ish clause keeps a trailing backslash out of the
// attribute side so negation escapes like `category:\-Movie` remain legal.
var facetFilterRegex = regexp.MustCompile(`^[^:]+[^\\]*:.+$`)

// numericFilterRegex matches either `<attribute>:<n> TO <n>` range syntax or
// `<attribute><op><number>` comparison syntax with op in {<, <=, =, >=, >}.
var numericFilterRegex = regexp.MustCompile(`^[^:]+[^\\]*:\d*\.?\d* TO \d*\.?\d*$|^[^:]+[^\\]*(<|<=|=|>=|>)\d*\.?\d*$`)

// maxFiltersValidator enforces MaxFilters across the sum of facet + numeric
// filter entries nested inside the `conditions` block. The server rejects
// configs that exceed this cap; surfacing it at plan time gives users a
// faster feedback loop.
type maxFiltersValidator struct{}

func (maxFiltersValidator) Description(_ context.Context) string {
	return fmt.Sprintf("conditions may contain at most %d filters total (facet + numeric combined)", MaxFilters)
}

func (maxFiltersValidator) MarkdownDescription(ctx context.Context) string {
	return (maxFiltersValidator{}).Description(ctx)
}

func (maxFiltersValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model CollectionResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !isKnown(model.Conditions) {
		return
	}

	var cond ConditionsModel
	resp.Diagnostics.Append(model.Conditions.As(ctx, &cond, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	n := countFilters(ctx, cond.FacetFilter) + countFilters(ctx, cond.NumericFilter)
	if n > MaxFilters {
		resp.Diagnostics.AddAttributeError(
			path.Root("conditions"),
			"Too many filters",
			fmt.Sprintf("conditions may contain at most %d filters total; got %d.", MaxFilters, n),
		)
	}
}

// preserveOnUnsetString is a plan modifier for string attributes backed by
// server state that the API cannot clear via update. When the user removes
// the attribute from HCL on an already-set resource, this modifier pins the
// plan value to the prior state (matching the server's actual behavior) and
// raises a warning so the user isn't silently misled.
type preserveOnUnsetString struct {
	reason string
}

// PreserveOnUnsetString returns a plan modifier that keeps the prior state
// value when the config is null and state has a value. `reason` is surfaced
// to the user in the warning diagnostic.
func PreserveOnUnsetString(reason string) planmodifier.String {
	return preserveOnUnsetString{reason: reason}
}

func (m preserveOnUnsetString) Description(_ context.Context) string {
	return "Preserves the prior state value when the config is unset. Reason: " + m.reason
}

func (m preserveOnUnsetString) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m preserveOnUnsetString) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// No state on create — nothing to preserve.
	if req.StateValue.IsNull() {
		return
	}
	// Config explicitly set — honor the user's choice (including a change).
	if !req.ConfigValue.IsNull() {
		return
	}
	// Config is null AND state has a value → user tried to clear something
	// the API won't let us clear. Keep state and warn.
	resp.PlanValue = req.StateValue
	resp.Diagnostics.AddAttributeWarning(
		req.Path,
		"Attribute cannot be cleared via update",
		m.reason+" Terraform will keep the current value ("+req.StateValue.ValueString()+"). To fully remove it, destroy and recreate the resource.",
	)
}

// countFilters sums the sizes of every FilterGroup's `filters` list within the
// given block list. Unknown groups or unknown filter lists are skipped so
// partially-known configs don't trigger spurious errors.
func countFilters(ctx context.Context, groups types.List) int {
	if groups.IsNull() || groups.IsUnknown() {
		return 0
	}
	var items []FilterGroupModel
	if diags := groups.ElementsAs(ctx, &items, false); diags.HasError() {
		return 0
	}
	total := 0
	for _, g := range items {
		if g.Filters.IsNull() || g.Filters.IsUnknown() {
			continue
		}
		total += len(g.Filters.Elements())
	}
	return total
}
