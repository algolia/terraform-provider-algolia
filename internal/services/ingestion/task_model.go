package ingestion

import "github.com/hashicorp/terraform-plugin-framework/types"

// TaskResourceModel describes the algolia_ingestion_task resource: a
// scheduled or on-demand pipeline that reads records from a source,
// optionally transforms them, and writes them to a destination.
//
// Input, Notifications, and Policies hold JSON-encoded configuration
// matching the API's TaskInput/Notifications/Policies types respectively.
// Like Source/Destination/Transformation's `input`, GetTask returns all
// three in full (nothing is redacted), so Read refreshes them - but only
// adopts the API's encoding when it is not semantically equal to what's
// already configured (ignoring key/array order), to avoid a perpetual
// diff. See flattenTask/flattenTaskInput/flattenTaskNotifications/
// flattenTaskPolicies and the shared jsonSemanticallyEqual/normalizeJSON
// helpers in json.go.
//
// Cursor is the exception: TaskUpdate has no Cursor field at all - the
// Ingestion API gives no way to change it after creation - and its true
// value advances automatically as the task actually runs, which is
// runtime state out of this provider's scope (see runs/events, out of
// scope). So, unlike every other attribute here, Cursor is never
// refreshed from the API on Read: flattenTask leaves it untouched,
// preserving whatever was configured/already in state. This mirrors
// algolia_ingestion_authentication's write-only `input` handling, and
// avoids Terraform's "provider produced inconsistent result" error that
// would otherwise occur once a running task's cursor advances in the
// background between applies.
type TaskResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	TaskID             types.String `tfsdk:"task_id"`
	SourceID           types.String `tfsdk:"source_id"`
	DestinationID      types.String `tfsdk:"destination_id"`
	Action             types.String `tfsdk:"action"`
	SubscriptionAction types.String `tfsdk:"subscription_action"`
	Cron               types.String `tfsdk:"cron"`
	Enabled            types.Bool   `tfsdk:"enabled"`
	FailureThreshold   types.Int64  `tfsdk:"failure_threshold"`
	Input              types.String `tfsdk:"input"`
	Notifications      types.String `tfsdk:"notifications"`
	Policies           types.String `tfsdk:"policies"`
	Cursor             types.String `tfsdk:"cursor"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
	LastRun            types.String `tfsdk:"last_run"`
	NextRun            types.String `tfsdk:"next_run"`
}

// TaskDataSourceModel describes the algolia_ingestion_task data source.
// Unlike the resource, Cursor is surfaced directly from the API here:
// a data source has no prior configuration to preserve, and its Read
// always recomputes freely (like created_at/last_run/next_run).
type TaskDataSourceModel struct {
	TaskID             types.String `tfsdk:"task_id"`
	ID                 types.String `tfsdk:"id"`
	SourceID           types.String `tfsdk:"source_id"`
	DestinationID      types.String `tfsdk:"destination_id"`
	Action             types.String `tfsdk:"action"`
	SubscriptionAction types.String `tfsdk:"subscription_action"`
	Cron               types.String `tfsdk:"cron"`
	Enabled            types.Bool   `tfsdk:"enabled"`
	FailureThreshold   types.Int64  `tfsdk:"failure_threshold"`
	Input              types.String `tfsdk:"input"`
	Notifications      types.String `tfsdk:"notifications"`
	Policies           types.String `tfsdk:"policies"`
	Cursor             types.String `tfsdk:"cursor"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
	LastRun            types.String `tfsdk:"last_run"`
	NextRun            types.String `tfsdk:"next_run"`
}
