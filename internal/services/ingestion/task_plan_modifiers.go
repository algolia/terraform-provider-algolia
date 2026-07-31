package ingestion

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// requiresReplaceOnRemoval forces the task to be replaced when an attribute is
// removed from configuration, for fields the Ingestion API can set but not clear.
//
// Task updates are a PATCH, and the client models these fields as pointers with
// `omitempty`, so there is no way to send "unset this" - dropping the attribute
// sends nothing at all. The server keeps its value, the read restores it, and the
// plan proposes the same removal on every run without ever converging.
//
// Replacement is the only operation that does converge: a task created without
// the field simply does not have it. Changing the value, by contrast, works fine
// over PATCH, so this deliberately triggers on removal alone rather than on any
// change - forcing a task to be destroyed and recreated just to move a schedule
// would be gratuitous.
//
// Confirmed against the live API: sending an empty string is rejected as an
// invalid cron expression, and sending an explicit JSON null returns 200 while
// leaving the schedule untouched, so the API genuinely has no representation for
// removing a schedule.
//
// Only `cron` uses this. `subscription_action` shares the same request shape but
// not the same read shape - flattenTask keeps the prior value for it when the API
// omits it, so its removal already produced a consistent apply, just one that
// silently left the server's value in place. Replacing the task to fix that would
// be worse than the divergence: Terraform destroys before it creates, a task
// carrying a subscription action sits on a platform source the API validates at
// create time, and there is no deletion protection on this resource - so a failed
// recreate leaves nothing at all. That field is handled by
// errorOnUnclearableRemoval instead.
func requiresReplaceOnRemoval() planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
			// The framework has already ruled out create, destroy and no-op
			// changes before calling this, so a null configuration value here
			// means the attribute was genuinely removed from a resource that had
			// one.
			resp.RequiresReplace = req.ConfigValue.IsNull() && !req.StateValue.IsNull()
		},
		"Removing this attribute requires the task to be replaced.",
		"Removing this attribute requires the task to be replaced.",
	)
}

// errorOnUnclearableRemoval refuses an update that would drop a field the
// Ingestion API cannot clear.
//
// Such an update is a no-op on the field - the request omits it and the server
// keeps what it had - so letting it through reports success for an apply that
// changed nothing, and leaves state claiming otherwise. Failing is better, and the
// message says which command resolves it.
//
// The two fields arrive here by different routes:
//
//   - `subscription_action` always does, on every removal, because it has no
//     replace-on-removal modifier. Automatically replacing the task would be the
//     more dangerous choice: see requiresReplaceOnRemoval.
//   - `cron` only does when the configured value was still unknown at plan time -
//     an expression referring to another resource's computed attribute, say - since
//     an unknown is neither null nor equal to prior state, so the plan is an
//     ordinary update and by the time it resolves to null the decision is made. A
//     known removal is planned as a replacement and never reaches Update.
func errorOnUnclearableRemoval(state, plan TaskResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	removals := []struct {
		name   string
		state  types.String
		plan   types.String
		detail string
	}{
		{
			name:  "cron",
			state: state.Cron,
			plan:  plan.Cron,
			detail: "Terraform planned an in-place update rather than a replacement because the configured " +
				"value was not yet known when the plan was made - it comes from an expression that only " +
				"resolved during apply.\n\nRe-run the apply: with the value now known, the next plan " +
				"replaces the task.",
		},
		{
			name:  "subscription_action",
			state: state.SubscriptionAction,
			plan:  plan.SubscriptionAction,
			detail: "The provider does not replace the task automatically for this field. A task carrying a " +
				"subscription action sits on a platform source the API validates when a task is created, " +
				"and a replacement destroys the existing task first - so a recreate that then fails would " +
				"leave nothing behind.",
		},
	}

	for _, removal := range removals {
		if removal.state.IsNull() || removal.state.IsUnknown() || !removal.plan.IsNull() {
			continue
		}

		diags.AddError(
			"Cannot remove "+removal.name+" without replacing the task",
			"The configuration removes "+removal.name+" from this task, but the Ingestion API has no way to "+
				"clear it: a task can only be without it by being created that way. "+removal.detail+
				"\n\nTo replace the task in one step, run terraform apply with -replace and this resource's "+
				"address.",
		)
	}

	return diags
}
