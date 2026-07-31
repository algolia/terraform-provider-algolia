package ingestion

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
// Confirmed against the live API for `cron`: sending an empty string is rejected
// as an invalid cron expression, and sending an explicit JSON null returns 200
// while leaving the schedule untouched, so the API genuinely has no
// representation for removing one. `subscription_action` could not be probed the
// same way - creating a task that accepts one requires a reachable platform
// source, which the API validates on create - but it has the identical
// `omitempty` pointer shape in the same request struct, and the perpetual diff
// follows from the provider being unable to send a removal at all, whatever the
// API would accept.
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
