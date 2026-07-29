package abtest

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// suppressEquivalentJSON returns a plan modifier that keeps the prior state value
// when the configured one carries the same JSON data, ignoring key order and
// whitespace.
//
// `variants`, `metrics` and `configuration` are JSON documents held in strings,
// and all three are RequiresReplace because the A/B Testing API has no update
// endpoint. That combination is unforgiving: reformatting a document, or listing
// the same keys in another order, would otherwise plan the destruction of a
// running experiment and discard the statistics it has accumulated. The risk is
// most acute right after `terraform import`, where the document in state was
// rendered by the provider rather than typed by the user.
//
// This must be listed BEFORE RequiresReplace in a schema's PlanModifiers. The
// framework runs them in order, so by aligning the planned value with state first
// the replace check sees no change at all. Reversing the order would leave the
// replace already decided by the time this ran.
//
// A genuine change still plans a replace: the two documents only compare equal
// when they decode to the same data.
func suppressEquivalentJSON() planmodifier.String {
	return suppressEquivalentJSONModifier{}
}

type suppressEquivalentJSONModifier struct{}

func (m suppressEquivalentJSONModifier) Description(_ context.Context) string {
	return "Keep the prior value when the configured JSON carries the same data, ignoring key order and whitespace."
}

func (m suppressEquivalentJSONModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m suppressEquivalentJSONModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Leave creation and destruction alone: there is nothing to compare against,
	// and an unknown plan value belongs to whichever modifier produces it.
	if req.StateValue.IsNull() || req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	if jsonEquivalent(req.StateValue.ValueString(), req.PlanValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}

// jsonEquivalent reports whether two strings are both valid JSON carrying the
// same data. A string that does not parse is never equivalent to anything, so a
// malformed document still reaches the schema's validator rather than being
// quietly treated as unchanged.
func jsonEquivalent(left, right string) bool {
	var leftDecoded, rightDecoded any
	if err := json.Unmarshal([]byte(left), &leftDecoded); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(right), &rightDecoded); err != nil {
		return false
	}

	return reflect.DeepEqual(leftDecoded, rightDecoded)
}
