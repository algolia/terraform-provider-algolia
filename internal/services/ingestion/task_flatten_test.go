package ingestion

import (
	"testing"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenTask_PopulatesFields(t *testing.T) {
	action := ingestionapi.ACTION_TYPE_REPLACE
	subscriptionAction := ingestionapi.ACTION_TYPE_PARTIAL
	cron := "0 0 * * *"
	lastRun := "2024-01-03T00:00:00Z"
	nextRun := "2024-01-04T00:00:00Z"
	failureThreshold := int32(50)

	task := &ingestionapi.Task{
		TaskID:             "task-1",
		SourceID:           "source-1",
		DestinationID:      "destination-1",
		Cron:               &cron,
		LastRun:            &lastRun,
		NextRun:            &nextRun,
		Enabled:            true,
		FailureThreshold:   &failureThreshold,
		Action:             &action,
		SubscriptionAction: &subscriptionAction,
		CreatedAt:          "2024-01-01T00:00:00Z",
		UpdatedAt:          "2024-01-02T00:00:00Z",
	}

	var model TaskResourceModel
	diags := flattenTask(task, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.ID.ValueString() != "task-1" {
		t.Fatalf("id = %v, want task-1", model.ID.ValueString())
	}
	if model.TaskID.ValueString() != "task-1" {
		t.Fatalf("task_id = %v, want task-1", model.TaskID.ValueString())
	}
	if model.SourceID.ValueString() != "source-1" {
		t.Fatalf("source_id = %v, want source-1", model.SourceID.ValueString())
	}
	if model.DestinationID.ValueString() != "destination-1" {
		t.Fatalf("destination_id = %v, want destination-1", model.DestinationID.ValueString())
	}
	if model.Cron.ValueString() != "0 0 * * *" {
		t.Fatalf("cron = %v, want 0 0 * * *", model.Cron.ValueString())
	}
	if !model.Enabled.ValueBool() {
		t.Fatalf("enabled = %v, want true", model.Enabled.ValueBool())
	}
	if model.FailureThreshold.ValueInt64() != 50 {
		t.Fatalf("failure_threshold = %v, want 50", model.FailureThreshold.ValueInt64())
	}
	if model.Action.ValueString() != "replace" {
		t.Fatalf("action = %v, want replace", model.Action.ValueString())
	}
	if model.SubscriptionAction.ValueString() != "partial" {
		t.Fatalf("subscription_action = %v, want partial", model.SubscriptionAction.ValueString())
	}
	if model.LastRun.ValueString() != lastRun {
		t.Fatalf("last_run = %v, want %v", model.LastRun.ValueString(), lastRun)
	}
	if model.NextRun.ValueString() != nextRun {
		t.Fatalf("next_run = %v, want %v", model.NextRun.ValueString(), nextRun)
	}
	if model.CreatedAt.ValueString() != "2024-01-01T00:00:00Z" {
		t.Fatalf("created_at = %v, want 2024-01-01T00:00:00Z", model.CreatedAt.ValueString())
	}
	if model.UpdatedAt.ValueString() != "2024-01-02T00:00:00Z" {
		t.Fatalf("updated_at = %v, want 2024-01-02T00:00:00Z", model.UpdatedAt.ValueString())
	}
}

func TestFlattenTask_NoOptionalFieldsAreNull(t *testing.T) {
	task := &ingestionapi.Task{
		TaskID:        "task-2",
		SourceID:      "source-1",
		DestinationID: "destination-1",
		Enabled:       false,
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
	}

	var model TaskResourceModel
	diags := flattenTask(task, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.Cron.IsNull() {
		t.Fatalf("cron = %#v, want null", model.Cron)
	}
	if !model.LastRun.IsNull() {
		t.Fatalf("last_run = %#v, want null", model.LastRun)
	}
	if !model.NextRun.IsNull() {
		t.Fatalf("next_run = %#v, want null", model.NextRun)
	}
	if !model.FailureThreshold.IsNull() {
		t.Fatalf("failure_threshold = %#v, want null", model.FailureThreshold)
	}
	if !model.SubscriptionAction.IsNull() {
		t.Fatalf("subscription_action = %#v, want null", model.SubscriptionAction)
	}
	if !model.Action.IsNull() {
		t.Fatalf("action = %#v, want null when the API omits it", model.Action)
	}
	if model.Enabled.ValueBool() {
		t.Fatalf("enabled = %v, want false", model.Enabled.ValueBool())
	}
}

// TestFlattenTask_CursorNeverTouched is the core regression test for
// Cursor's write-once handling: TaskUpdate has no way to change it, and its
// true value advances automatically as the task runs in the background, so
// flattenTask must never overwrite whatever was already in the model -
// regardless of what GetTask returns - to avoid Terraform's "provider
// produced inconsistent result" error on a later Update.
func TestFlattenTask_CursorNeverTouched(t *testing.T) {
	task := &ingestionapi.Task{
		TaskID:        "task-3",
		SourceID:      "source-1",
		DestinationID: "destination-1",
		Enabled:       true,
		Cursor:        stringPtr("2024-06-01T00:00:00Z"), // simulates a live, advanced cursor
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
	}

	t.Run("configured value preserved", func(t *testing.T) {
		model := TaskResourceModel{Cursor: types.StringValue("2024-01-01T00:00:00Z")}
		diags := flattenTask(task, &model)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if model.Cursor.ValueString() != "2024-01-01T00:00:00Z" {
			t.Fatalf("cursor = %v, want unchanged configured value", model.Cursor.ValueString())
		}
	})

	t.Run("null value preserved", func(t *testing.T) {
		model := TaskResourceModel{Cursor: types.StringNull()}
		diags := flattenTask(task, &model)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if !model.Cursor.IsNull() {
			t.Fatalf("cursor = %#v, want null even though the API returned a value", model.Cursor)
		}
	})
}

func stringPtr(s string) *string { return &s }

// TestFlattenTask_InputPreservedWhenSemanticallyEqual is the core
// regression test for the semantic-equality refresh behavior: like
// Source/Destination/Transformation's input, GetTask returns a task's
// input in full, so naively overwriting the configured JSON string with
// the API's encoding on every Read would create a perpetual diff whenever
// the API echoes back the same configuration with different key/array
// ordering.
func TestFlattenTask_InputPreservedWhenSemanticallyEqual(t *testing.T) {
	const configuredInput = `{"streams": [{"name": "orders", "syncMode": "fullTable"}]}`

	model := TaskResourceModel{Input: types.StringValue(configuredInput)}

	// Build the API's echoed input independently via expandTaskInput (the
	// same decode path used elsewhere), with keys in a different order but
	// semantically identical data.
	apiInput, expandDiags := expandTaskInput(types.StringValue(`{"streams": [{"syncMode": "fullTable", "name": "orders"}]}`))
	if expandDiags.HasError() {
		t.Fatalf("unexpected diagnostics building fixture: %v", expandDiags)
	}

	task := &ingestionapi.Task{
		TaskID:        "task-4",
		SourceID:      "source-1",
		DestinationID: "destination-1",
		Enabled:       true,
		Input:         apiInput,
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
	}

	diags := flattenTask(task, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Input.ValueString() != configuredInput {
		t.Fatalf("input = %v, want unchanged configured value %v", model.Input.ValueString(), configuredInput)
	}
}

// TestFlattenTask_InputAdoptsAPIValueWhenDifferent covers the opposite
// case: when the API's input is not semantically equal to what's
// configured (e.g. drift introduced out-of-band), flattenTask must adopt
// the API's value so Terraform actually reports the difference.
func TestFlattenTask_InputAdoptsAPIValueWhenDifferent(t *testing.T) {
	model := TaskResourceModel{Input: types.StringValue(`{"streams": [{"name": "old", "syncMode": "fullTable"}]}`)}

	apiInput, expandDiags := expandTaskInput(types.StringValue(`{"streams": [{"name": "new", "syncMode": "fullTable"}]}`))
	if expandDiags.HasError() {
		t.Fatalf("unexpected diagnostics building fixture: %v", expandDiags)
	}

	task := &ingestionapi.Task{
		TaskID:        "task-5",
		SourceID:      "source-1",
		DestinationID: "destination-1",
		Enabled:       true,
		Input:         apiInput,
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
	}

	diags := flattenTask(task, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Input.ValueString() == `{"streams": [{"name": "old", "syncMode": "fullTable"}]}` {
		t.Fatalf("input = %v, want the API's new value adopted", model.Input.ValueString())
	}
}

func TestFlattenTask_NilAPIInputKeepsConfiguredValue(t *testing.T) {
	model := TaskResourceModel{Input: types.StringValue(`{"some": "config"}`)}

	task := &ingestionapi.Task{
		TaskID:        "task-6",
		SourceID:      "source-1",
		DestinationID: "destination-1",
		Enabled:       true,
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
	}

	diags := flattenTask(task, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Input.ValueString() != `{"some": "config"}` {
		t.Fatalf("input = %v, want the previously configured value preserved", model.Input.ValueString())
	}
}

func TestFlattenTask_NilAPIInputAndUnconfiguredIsNull(t *testing.T) {
	model := TaskResourceModel{Input: types.StringNull()}

	task := &ingestionapi.Task{
		TaskID:        "task-7",
		SourceID:      "source-1",
		DestinationID: "destination-1",
		Enabled:       true,
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
	}

	diags := flattenTask(task, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.Input.IsNull() {
		t.Fatalf("input = %#v, want null", model.Input)
	}
}

// TestFlattenTask_NotificationsPreservedWhenSemanticallyEqual mirrors
// TestFlattenTask_InputPreservedWhenSemanticallyEqual for `notifications`.
func TestFlattenTask_NotificationsPreservedWhenSemanticallyEqual(t *testing.T) {
	const configured = `{"email": {"enabled": true}}`

	model := TaskResourceModel{Notifications: types.StringValue(configured)}

	enabled := true
	task := &ingestionapi.Task{
		TaskID:        "task-8",
		SourceID:      "source-1",
		DestinationID: "destination-1",
		Enabled:       true,
		Notifications: &ingestionapi.Notifications{Email: ingestionapi.EmailNotifications{Enabled: &enabled}},
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
	}

	diags := flattenTask(task, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Notifications.ValueString() != configured {
		t.Fatalf("notifications = %v, want unchanged configured value %v", model.Notifications.ValueString(), configured)
	}
}

func TestFlattenTask_NotificationsAdoptsAPIValueWhenDifferent(t *testing.T) {
	model := TaskResourceModel{Notifications: types.StringValue(`{"email": {"enabled": true}}`)}

	enabled := false
	task := &ingestionapi.Task{
		TaskID:        "task-9",
		SourceID:      "source-1",
		DestinationID: "destination-1",
		Enabled:       true,
		Notifications: &ingestionapi.Notifications{Email: ingestionapi.EmailNotifications{Enabled: &enabled}},
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
	}

	diags := flattenTask(task, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Notifications.ValueString() != `{"email":{"enabled":false}}` {
		t.Fatalf("notifications = %v, want the API's new value", model.Notifications.ValueString())
	}
}

// TestFlattenTask_PoliciesPreservedWhenSemanticallyEqual mirrors
// TestFlattenTask_InputPreservedWhenSemanticallyEqual for `policies`.
func TestFlattenTask_PoliciesPreservedWhenSemanticallyEqual(t *testing.T) {
	const configured = `{"criticalThreshold": 50}`

	model := TaskResourceModel{Policies: types.StringValue(configured)}

	threshold := int32(50)
	task := &ingestionapi.Task{
		TaskID:        "task-10",
		SourceID:      "source-1",
		DestinationID: "destination-1",
		Enabled:       true,
		Policies:      &ingestionapi.Policies{CriticalThreshold: &threshold},
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
	}

	diags := flattenTask(task, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Policies.ValueString() != configured {
		t.Fatalf("policies = %v, want unchanged configured value %v", model.Policies.ValueString(), configured)
	}
}

func TestFlattenTaskDataSource_PopulatesInputNotificationsPoliciesAndCursor(t *testing.T) {
	enabled := true
	threshold := int32(50)
	cursor := "2024-06-01T00:00:00Z"

	apiInput, expandDiags := expandTaskInput(types.StringValue(`{"streams": [{"name": "orders", "syncMode": "fullTable"}]}`))
	if expandDiags.HasError() {
		t.Fatalf("unexpected diagnostics building fixture: %v", expandDiags)
	}

	task := &ingestionapi.Task{
		TaskID:        "task-11",
		SourceID:      "source-1",
		DestinationID: "destination-1",
		Enabled:       true,
		Cursor:        &cursor,
		Input:         apiInput,
		Notifications: &ingestionapi.Notifications{Email: ingestionapi.EmailNotifications{Enabled: &enabled}},
		Policies:      &ingestionapi.Policies{CriticalThreshold: &threshold},
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
	}

	var model TaskDataSourceModel
	diags := flattenTaskDataSource(task, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Cursor.ValueString() != cursor {
		t.Fatalf("cursor = %v, want %v (data source surfaces it directly)", model.Cursor.ValueString(), cursor)
	}
	if model.Input.ValueString() != `{"streams":[{"name":"orders","syncMode":"fullTable"}]}` {
		t.Fatalf("input = %v, want the encoded API value", model.Input.ValueString())
	}
	if model.Notifications.ValueString() != `{"email":{"enabled":true}}` {
		t.Fatalf("notifications = %v, want the encoded API value", model.Notifications.ValueString())
	}
	if model.Policies.ValueString() != `{"criticalThreshold":50}` {
		t.Fatalf("policies = %v, want the encoded API value", model.Policies.ValueString())
	}
}

func TestFlattenTaskDataSource_NilInputNotificationsPoliciesAreNull(t *testing.T) {
	task := &ingestionapi.Task{
		TaskID:        "task-12",
		SourceID:      "source-1",
		DestinationID: "destination-1",
		Enabled:       true,
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
	}

	var model TaskDataSourceModel
	diags := flattenTaskDataSource(task, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.Input.IsNull() {
		t.Fatalf("input = %#v, want null", model.Input)
	}
	if !model.Notifications.IsNull() {
		t.Fatalf("notifications = %#v, want null", model.Notifications)
	}
	if !model.Policies.IsNull() {
		t.Fatalf("policies = %#v, want null", model.Policies)
	}
	if !model.Cursor.IsNull() {
		t.Fatalf("cursor = %#v, want null", model.Cursor)
	}
}
