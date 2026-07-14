package ingestion

import (
	"encoding/json"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// expandTaskCreate converts the Terraform plan into a TaskCreate request
// body for CreateTask.
func expandTaskCreate(model *TaskResourceModel) (*ingestionapi.TaskCreate, diag.Diagnostics) {
	var diags diag.Diagnostics

	input, inputDiags := expandTaskInput(model.Input)
	diags.Append(inputDiags...)
	notifications, notificationsDiags := expandTaskNotifications(model.Notifications)
	diags.Append(notificationsDiags...)
	policies, policiesDiags := expandTaskPolicies(model.Policies)
	diags.Append(policiesDiags...)
	if diags.HasError() {
		return nil, diags
	}

	create := ingestionapi.NewTaskCreate(
		model.SourceID.ValueString(),
		model.DestinationID.ValueString(),
		ingestionapi.ActionType(model.Action.ValueString()),
	)
	create.Cron = model.Cron.ValueStringPointer()
	create.Enabled = model.Enabled.ValueBoolPointer()
	create.Input = input
	create.Cursor = model.Cursor.ValueStringPointer()
	create.Notifications = notifications
	create.Policies = policies
	create.SubscriptionAction = expandActionTypePointer(model.SubscriptionAction)
	create.FailureThreshold = expandFailureThreshold(model.FailureThreshold)

	return create, diags
}

// expandTaskUpdate converts the Terraform plan into a TaskUpdate request
// body for UpdateTask.
//
// TaskUpdate has no SourceID and no Action field at all: the Ingestion
// API gives no way to change a task's source or action after creation,
// which is why both are RequiresReplace in the resource schema. It also
// has no Cursor field - see the `cursor` attribute's schema description
// and TaskResourceModel's doc comment for why that attribute is never
// sent on update at all.
func expandTaskUpdate(model *TaskResourceModel) (*ingestionapi.TaskUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics

	input, inputDiags := expandTaskInput(model.Input)
	diags.Append(inputDiags...)
	notifications, notificationsDiags := expandTaskNotifications(model.Notifications)
	diags.Append(notificationsDiags...)
	policies, policiesDiags := expandTaskPolicies(model.Policies)
	diags.Append(policiesDiags...)
	if diags.HasError() {
		return nil, diags
	}

	update := ingestionapi.NewTaskUpdate()
	update.DestinationID = model.DestinationID.ValueStringPointer()
	update.Cron = model.Cron.ValueStringPointer()
	update.Enabled = model.Enabled.ValueBoolPointer()
	update.Input = input
	update.Notifications = notifications
	update.Policies = policies
	update.SubscriptionAction = expandActionTypePointer(model.SubscriptionAction)
	update.FailureThreshold = expandFailureThreshold(model.FailureThreshold)

	return update, diags
}

// expandActionTypePointer converts the optional `subscription_action`
// attribute into an *ActionType, or nil if unconfigured.
func expandActionTypePointer(value types.String) *ingestionapi.ActionType {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil
	}

	actionType := ingestionapi.ActionType(value.ValueString())

	return &actionType
}

// expandFailureThreshold converts the optional `failure_threshold`
// attribute into an *int32, or nil if unconfigured.
func expandFailureThreshold(value types.Int64) *int32 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	failureThreshold := int32(value.ValueInt64())

	return &failureThreshold
}

// expandTaskInput JSON-decodes the `input` attribute into the TaskInput
// union type expected by TaskCreate/TaskUpdate. `input` is Optional: not
// every task needs one (e.g. a task on a "push" source has none), so a
// null/unknown/empty value decodes to a nil *TaskInput, which
// TaskCreate/TaskUpdate's MarshalJSON simply omits from the request body.
func expandTaskInput(input types.String) (*ingestionapi.TaskInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	if input.IsNull() || input.IsUnknown() || input.ValueString() == "" {
		return nil, diags
	}

	var taskInput ingestionapi.TaskInput
	if err := json.Unmarshal([]byte(input.ValueString()), &taskInput); err != nil {
		diags.AddError(
			"Invalid input JSON",
			"The `input` attribute must be JSON-encoded configuration matching the task's source "+
				"(e.g. jsonencode({ streams = [...] })). Failed to parse: "+err.Error(),
		)
		return nil, diags
	}

	return &taskInput, diags
}

// expandTaskNotifications JSON-decodes the `notifications` attribute into
// the Notifications type expected by TaskCreate/TaskUpdate.
func expandTaskNotifications(notifications types.String) (*ingestionapi.Notifications, diag.Diagnostics) {
	var diags diag.Diagnostics

	if notifications.IsNull() || notifications.IsUnknown() || notifications.ValueString() == "" {
		return nil, diags
	}

	var decoded ingestionapi.Notifications
	if err := json.Unmarshal([]byte(notifications.ValueString()), &decoded); err != nil {
		diags.AddError(
			"Invalid notifications JSON",
			"The `notifications` attribute must be JSON-encoded configuration (e.g. "+
				"jsonencode({ email = { enabled = true } })). Failed to parse: "+err.Error(),
		)
		return nil, diags
	}

	return &decoded, diags
}

// expandTaskPolicies JSON-decodes the `policies` attribute into the
// Policies type expected by TaskCreate/TaskUpdate.
func expandTaskPolicies(policies types.String) (*ingestionapi.Policies, diag.Diagnostics) {
	var diags diag.Diagnostics

	if policies.IsNull() || policies.IsUnknown() || policies.ValueString() == "" {
		return nil, diags
	}

	var decoded ingestionapi.Policies
	if err := json.Unmarshal([]byte(policies.ValueString()), &decoded); err != nil {
		diags.AddError(
			"Invalid policies JSON",
			"The `policies` attribute must be JSON-encoded configuration (e.g. "+
				"jsonencode({ criticalThreshold = 50 })). Failed to parse: "+err.Error(),
		)
		return nil, diags
	}

	return &decoded, diags
}
