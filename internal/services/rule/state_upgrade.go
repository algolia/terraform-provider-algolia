package rule

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ resource.ResourceWithUpgradeState = &ruleResource{}

func (r *ruleResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {StateUpgrader: upgradeRuleStateV0},
	}
}

func upgradeRuleStateV0(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	// Sets and lists are both JSON arrays in stored state. Decode the complete
	// state against version 1 to change only object_ids' type. The old set has
	// already lost its order; the following Read restores Algolia's live order.
	state, err := req.RawState.Unmarshal(ruleResourceSchema().Type().TerraformType(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error upgrading rule state", err.Error())
		return
	}
	resp.State.Raw = state
}
