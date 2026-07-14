package ingestion

import (
	"testing"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExpandTaskCreate_Basic(t *testing.T) {
	model := &TaskResourceModel{
		SourceID:      types.StringValue("source-1"),
		DestinationID: types.StringValue("destination-1"),
		Action:        types.StringValue(string(ingestionapi.ACTION_TYPE_REPLACE)),
		Enabled:       types.BoolValue(true),
	}

	create, diags := expandTaskCreate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if create.SourceID != "source-1" {
		t.Fatalf("sourceID = %v, want source-1", create.SourceID)
	}
	if create.DestinationID != "destination-1" {
		t.Fatalf("destinationID = %v, want destination-1", create.DestinationID)
	}
	if create.Action != ingestionapi.ACTION_TYPE_REPLACE {
		t.Fatalf("action = %v, want replace", create.Action)
	}
	if create.Enabled == nil || !*create.Enabled {
		t.Fatalf("enabled = %#v, want true", create.Enabled)
	}
	if create.Cron != nil {
		t.Fatalf("expected cron to be nil, got %#v", create.Cron)
	}
	if create.Input != nil {
		t.Fatalf("expected input to be nil, got %#v", create.Input)
	}
	if create.Notifications != nil {
		t.Fatalf("expected notifications to be nil, got %#v", create.Notifications)
	}
	if create.Policies != nil {
		t.Fatalf("expected policies to be nil, got %#v", create.Policies)
	}
	if create.SubscriptionAction != nil {
		t.Fatalf("expected subscriptionAction to be nil, got %#v", create.SubscriptionAction)
	}
	if create.FailureThreshold != nil {
		t.Fatalf("expected failureThreshold to be nil, got %#v", create.FailureThreshold)
	}
}

func TestExpandTaskCreate_WithAllOptionalFields(t *testing.T) {
	model := &TaskResourceModel{
		SourceID:           types.StringValue("source-1"),
		DestinationID:      types.StringValue("destination-1"),
		Action:             types.StringValue(string(ingestionapi.ACTION_TYPE_SAVE)),
		SubscriptionAction: types.StringValue(string(ingestionapi.ACTION_TYPE_PARTIAL)),
		Cron:               types.StringValue("0 0 * * *"),
		Enabled:            types.BoolValue(false),
		FailureThreshold:   types.Int64Value(50),
		Cursor:             types.StringValue("2024-01-01T00:00:00Z"),
		Input:              types.StringValue(`{"streams": [{"name": "orders", "syncMode": "fullTable"}]}`),
		Notifications:      types.StringValue(`{"email": {"enabled": true}}`),
		Policies:           types.StringValue(`{"criticalThreshold": 25}`),
	}

	create, diags := expandTaskCreate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if create.Cron == nil || *create.Cron != "0 0 * * *" {
		t.Fatalf("cron = %#v, want 0 0 * * *", create.Cron)
	}
	if create.Enabled == nil || *create.Enabled {
		t.Fatalf("enabled = %#v, want false", create.Enabled)
	}
	if create.FailureThreshold == nil || *create.FailureThreshold != 50 {
		t.Fatalf("failureThreshold = %#v, want 50", create.FailureThreshold)
	}
	if create.Cursor == nil || *create.Cursor != "2024-01-01T00:00:00Z" {
		t.Fatalf("cursor = %#v, want 2024-01-01T00:00:00Z", create.Cursor)
	}
	if create.SubscriptionAction == nil || *create.SubscriptionAction != ingestionapi.ACTION_TYPE_PARTIAL {
		t.Fatalf("subscriptionAction = %#v, want partial", create.SubscriptionAction)
	}
	if create.Input == nil || create.Input.DockerStreamsInput == nil {
		t.Fatalf("expected input to decode into DockerStreamsInput, got %#v", create.Input)
	}
	if create.Notifications == nil || create.Notifications.Email.Enabled == nil || !*create.Notifications.Email.Enabled {
		t.Fatalf("notifications = %#v, want email.enabled = true", create.Notifications)
	}
	if create.Policies == nil || create.Policies.CriticalThreshold == nil || *create.Policies.CriticalThreshold != 25 {
		t.Fatalf("policies = %#v, want criticalThreshold = 25", create.Policies)
	}
}

func TestExpandTaskCreate_InvalidInputJSON(t *testing.T) {
	model := &TaskResourceModel{
		SourceID:      types.StringValue("source-1"),
		DestinationID: types.StringValue("destination-1"),
		Action:        types.StringValue(string(ingestionapi.ACTION_TYPE_REPLACE)),
		Input:         types.StringValue(`{not valid json`),
	}

	_, diags := expandTaskCreate(model)
	if !diags.HasError() {
		t.Fatal("expected a diagnostic error for invalid input JSON")
	}
}

func TestExpandTaskCreate_InvalidNotificationsJSON(t *testing.T) {
	model := &TaskResourceModel{
		SourceID:      types.StringValue("source-1"),
		DestinationID: types.StringValue("destination-1"),
		Action:        types.StringValue(string(ingestionapi.ACTION_TYPE_REPLACE)),
		Notifications: types.StringValue(`not json at all`),
	}

	_, diags := expandTaskCreate(model)
	if !diags.HasError() {
		t.Fatal("expected a diagnostic error for invalid notifications JSON")
	}
}

func TestExpandTaskCreate_InvalidPoliciesJSON(t *testing.T) {
	model := &TaskResourceModel{
		SourceID:      types.StringValue("source-1"),
		DestinationID: types.StringValue("destination-1"),
		Action:        types.StringValue(string(ingestionapi.ACTION_TYPE_REPLACE)),
		Policies:      types.StringValue(`not json at all`),
	}

	_, diags := expandTaskCreate(model)
	if !diags.HasError() {
		t.Fatal("expected a diagnostic error for invalid policies JSON")
	}
}

func TestExpandTaskUpdate_HasNoSourceIDOrActionOrCursor(t *testing.T) {
	model := &TaskResourceModel{
		SourceID:      types.StringValue("source-1"),
		DestinationID: types.StringValue("destination-2"),
		Action:        types.StringValue(string(ingestionapi.ACTION_TYPE_REPLACE)),
		Enabled:       types.BoolValue(true),
		Cursor:        types.StringValue("some-cursor-value"),
	}

	update, diags := expandTaskUpdate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if update.DestinationID == nil || *update.DestinationID != "destination-2" {
		t.Fatalf("destinationID = %#v, want destination-2", update.DestinationID)
	}
	if update.Enabled == nil || !*update.Enabled {
		t.Fatalf("enabled = %#v, want true", update.Enabled)
	}
	// TaskUpdate (the Go client type) has no SourceID, Action, or Cursor
	// fields at all - this test exists mainly to document that, and to
	// confirm expandTaskUpdate compiles/works without them.
}

func TestExpandTaskUpdate_WithNotificationsAndPolicies(t *testing.T) {
	model := &TaskResourceModel{
		DestinationID: types.StringValue("destination-1"),
		Notifications: types.StringValue(`{"email": {"enabled": false}}`),
		Policies:      types.StringValue(`{"criticalThreshold": 10}`),
	}

	update, diags := expandTaskUpdate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if update.Notifications == nil || update.Notifications.Email.Enabled == nil || *update.Notifications.Email.Enabled {
		t.Fatalf("notifications = %#v, want email.enabled = false", update.Notifications)
	}
	if update.Policies == nil || update.Policies.CriticalThreshold == nil || *update.Policies.CriticalThreshold != 10 {
		t.Fatalf("policies = %#v, want criticalThreshold = 10", update.Policies)
	}
}

func TestExpandTaskUpdate_InvalidInputJSON(t *testing.T) {
	model := &TaskResourceModel{
		DestinationID: types.StringValue("destination-1"),
		Input:         types.StringValue(`not json`),
	}

	_, diags := expandTaskUpdate(model)
	if !diags.HasError() {
		t.Fatal("expected a diagnostic error for invalid input JSON")
	}
}

func TestExpandTaskInput_NullAndEmptyReturnNil(t *testing.T) {
	t.Run("null", func(t *testing.T) {
		input, diags := expandTaskInput(types.StringNull())
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if input != nil {
			t.Fatalf("expected nil input, got %#v", input)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		input, diags := expandTaskInput(types.StringValue(""))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if input != nil {
			t.Fatalf("expected nil input, got %#v", input)
		}
	})
}

func TestExpandTaskNotifications_NullAndEmptyReturnNil(t *testing.T) {
	t.Run("null", func(t *testing.T) {
		notifications, diags := expandTaskNotifications(types.StringNull())
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if notifications != nil {
			t.Fatalf("expected nil notifications, got %#v", notifications)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		notifications, diags := expandTaskNotifications(types.StringValue(""))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if notifications != nil {
			t.Fatalf("expected nil notifications, got %#v", notifications)
		}
	})
}

func TestExpandTaskPolicies_NullAndEmptyReturnNil(t *testing.T) {
	t.Run("null", func(t *testing.T) {
		policies, diags := expandTaskPolicies(types.StringNull())
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if policies != nil {
			t.Fatalf("expected nil policies, got %#v", policies)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		policies, diags := expandTaskPolicies(types.StringValue(""))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if policies != nil {
			t.Fatalf("expected nil policies, got %#v", policies)
		}
	})
}

func TestExpandFailureThreshold_NullAndUnknownReturnNil(t *testing.T) {
	if got := expandFailureThreshold(types.Int64Null()); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
	if got := expandFailureThreshold(types.Int64Unknown()); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
	if got := expandFailureThreshold(types.Int64Value(42)); got == nil || *got != 42 {
		t.Fatalf("expected 42, got %#v", got)
	}
}

func TestExpandActionTypePointer_NullAndEmptyReturnNil(t *testing.T) {
	if got := expandActionTypePointer(types.StringNull()); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
	if got := expandActionTypePointer(types.StringValue("")); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
	if got := expandActionTypePointer(types.StringValue("append")); got == nil || *got != ingestionapi.ACTION_TYPE_APPEND {
		t.Fatalf("expected append, got %#v", got)
	}
}
