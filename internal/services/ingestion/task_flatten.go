package ingestion

import (
	"encoding/json"

	"github.com/algolia/terraform-provider-algolia/internal/deletionprotection"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// flattenTask copies a GetTask response into the Terraform resource
// model.
//
// Input/Notifications/Policies are refreshed the same way as
// Source/Destination/Transformation's `input`: GetTask returns them in
// full (nothing is redacted), but naively overwriting the configured JSON
// string with the API's encoding on every Create/Read/Update would cause
// a perpetual diff whenever the API echoes back semantically identical
// JSON in a different form (key order, array order). So the corresponding
// flatten helpers below keep the model's existing string as-is when it is
// semantically equal to what the API returned, and only adopt the API's
// encoding when it actually differs.
//
// Cursor is the exception: per the `cursor` attribute's schema
// description and TaskResourceModel's doc comment, it is never touched
// here - flattenTask leaves model.Cursor exactly as it already was
// (the plan's configured value on Create/Update, or the prior state on
// Read).
func flattenTask(task *ingestionapi.Task, model *TaskResourceModel) diag.Diagnostics {
	// Algolia does not store this flag, so it survives only by being carried through
	// every rebuild of the model. Resolving it here also seeds an import, which
	// arrives with no value at all.
	model.DeletionProtection = deletionprotection.Value(model.DeletionProtection)

	var diags diag.Diagnostics

	model.ID = types.StringValue(task.TaskID)
	model.TaskID = types.StringValue(task.TaskID)
	model.SourceID = types.StringValue(task.SourceID)
	model.DestinationID = types.StringValue(task.DestinationID)
	model.Cron = types.StringPointerValue(task.Cron)
	model.Enabled = types.BoolValue(task.Enabled)
	model.CreatedAt = types.StringValue(task.CreatedAt)
	model.UpdatedAt = types.StringValue(task.UpdatedAt)
	model.LastRun = types.StringPointerValue(task.LastRun)
	model.NextRun = types.StringPointerValue(task.NextRun)
	model.Action = flattenActionTypeOrPrior(task.Action, model.Action)
	model.SubscriptionAction = flattenActionTypeOrPrior(task.SubscriptionAction, model.SubscriptionAction)
	model.FailureThreshold = flattenFailureThreshold(task.FailureThreshold)

	inputValue, inputDiags := flattenTaskInput(task.Input, model.Input)
	diags.Append(inputDiags...)
	model.Input = inputValue

	notificationsValue, notificationsDiags := flattenTaskNotifications(task.Notifications, model.Notifications)
	diags.Append(notificationsDiags...)
	model.Notifications = notificationsValue

	policiesValue, policiesDiags := flattenTaskPolicies(task.Policies, model.Policies)
	diags.Append(policiesDiags...)
	model.Policies = policiesValue

	return diags
}

// flattenActionTypeOrPrior converts an *ActionType into a types.String, keeping
// the prior value when the API omits the field.
//
// Task.Action and Task.SubscriptionAction are both `omitempty` on the client's
// read model and the Ingestion API does not echo them back, so reading them as
// null would report `action` as removed on every apply - and since `action` is
// Required, that surfaced as "Provider produced inconsistent result after apply:
// .action: was cty.StringVal(...), but now null". Both attributes are
// RequiresReplace, so a configured value cannot drift and preserving it is safe.
func flattenActionTypeOrPrior(action *ingestionapi.ActionType, prior types.String) types.String {
	if action == nil {
		return prior
	}

	return types.StringValue(string(*action))
}

// flattenActionType converts an *ActionType into a types.String, null if
// unset.
func flattenActionType(action *ingestionapi.ActionType) types.String {
	if action == nil {
		return types.StringNull()
	}

	return types.StringValue(string(*action))
}

// flattenFailureThreshold converts a *int32 into a types.Int64, null if
// unset.
func flattenFailureThreshold(failureThreshold *int32) types.Int64 {
	if failureThreshold == nil {
		return types.Int64Null()
	}

	return types.Int64Value(int64(*failureThreshold))
}

// flattenTaskInput JSON-encodes the API's *TaskInput and decides whether
// to adopt it into state or keep the value already configured. previous
// is model.Input's value before this Create/Read/Update call - i.e. the
// plan's configured value (Create/Update) or the prior state (Read).
// Mirrors flattenSourceInput/flattenTransformationInput.
func flattenTaskInput(input *ingestionapi.TaskInput, previous types.String) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if input == nil {
		if previous.IsNull() || previous.IsUnknown() {
			return types.StringNull(), diags
		}

		return previous, diags
	}

	encoded, err := json.Marshal(input)
	if err != nil {
		diags.AddError("Error encoding task input", "Could not JSON-encode the task's input: "+err.Error())
		return previous, diags
	}
	apiValue := string(encoded)

	if !previous.IsNull() && !previous.IsUnknown() && jsonSemanticallyEqual(previous.ValueString(), apiValue) {
		return previous, diags
	}

	return types.StringValue(apiValue), diags
}

// flattenTaskNotifications is the Notifications counterpart of
// flattenTaskInput.
func flattenTaskNotifications(notifications *ingestionapi.Notifications, previous types.String) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if notifications == nil {
		if previous.IsNull() || previous.IsUnknown() {
			return types.StringNull(), diags
		}

		return previous, diags
	}

	encoded, err := json.Marshal(notifications)
	if err != nil {
		diags.AddError("Error encoding task notifications", "Could not JSON-encode the task's notifications: "+err.Error())
		return previous, diags
	}
	apiValue := string(encoded)

	if !previous.IsNull() && !previous.IsUnknown() && jsonSemanticallyEqual(previous.ValueString(), apiValue) {
		return previous, diags
	}

	return types.StringValue(apiValue), diags
}

// flattenTaskPolicies is the Policies counterpart of flattenTaskInput.
func flattenTaskPolicies(policies *ingestionapi.Policies, previous types.String) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if policies == nil {
		if previous.IsNull() || previous.IsUnknown() {
			return types.StringNull(), diags
		}

		return previous, diags
	}

	encoded, err := json.Marshal(policies)
	if err != nil {
		diags.AddError("Error encoding task policies", "Could not JSON-encode the task's policies: "+err.Error())
		return previous, diags
	}
	apiValue := string(encoded)

	if !previous.IsNull() && !previous.IsUnknown() && jsonSemanticallyEqual(previous.ValueString(), apiValue) {
		return previous, diags
	}

	return types.StringValue(apiValue), diags
}

// flattenTaskDataSource is the data source counterpart of flattenTask.
// The data source has no prior configuration to preserve, so it always
// surfaces the API's JSON encoding of input/notifications/policies
// verbatim, and - unlike the resource - surfaces cursor directly too.
func flattenTaskDataSource(task *ingestionapi.Task, model *TaskDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(task.TaskID)
	model.TaskID = types.StringValue(task.TaskID)
	model.SourceID = types.StringValue(task.SourceID)
	model.DestinationID = types.StringValue(task.DestinationID)
	model.Cron = types.StringPointerValue(task.Cron)
	model.Enabled = types.BoolValue(task.Enabled)
	model.Cursor = types.StringPointerValue(task.Cursor)
	model.CreatedAt = types.StringValue(task.CreatedAt)
	model.UpdatedAt = types.StringValue(task.UpdatedAt)
	model.LastRun = types.StringPointerValue(task.LastRun)
	model.NextRun = types.StringPointerValue(task.NextRun)
	model.Action = flattenActionType(task.Action)
	model.SubscriptionAction = flattenActionType(task.SubscriptionAction)
	model.FailureThreshold = flattenFailureThreshold(task.FailureThreshold)

	if task.Input == nil {
		model.Input = types.StringNull()
	} else {
		encoded, err := json.Marshal(task.Input)
		if err != nil {
			diags.AddError("Error encoding task input", "Could not JSON-encode the task's input: "+err.Error())
			return diags
		}
		model.Input = types.StringValue(string(encoded))
	}

	if task.Notifications == nil {
		model.Notifications = types.StringNull()
	} else {
		encoded, err := json.Marshal(task.Notifications)
		if err != nil {
			diags.AddError("Error encoding task notifications", "Could not JSON-encode the task's notifications: "+err.Error())
			return diags
		}
		model.Notifications = types.StringValue(string(encoded))
	}

	if task.Policies == nil {
		model.Policies = types.StringNull()
	} else {
		encoded, err := json.Marshal(task.Policies)
		if err != nil {
			diags.AddError("Error encoding task policies", "Could not JSON-encode the task's policies: "+err.Error())
			return diags
		}
		model.Policies = types.StringValue(string(encoded))
	}

	return diags
}
