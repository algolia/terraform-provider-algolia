package ingestion

import (
	"testing"

	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestTaskResourceSchema_SourceIDAndActionAreRequiredWithReplace(t *testing.T) {
	s := taskResourceSchema()

	sourceIDAttr, ok := s.Attributes["source_id"].(resourceschema.StringAttribute)
	if !ok || !sourceIDAttr.Required {
		t.Fatal("expected source_id to be a required string attribute")
	}
	if len(sourceIDAttr.PlanModifiers) == 0 {
		t.Fatal("expected source_id to have a RequiresReplace plan modifier: TaskUpdate has no sourceID field")
	}

	actionAttr, ok := s.Attributes["action"].(resourceschema.StringAttribute)
	if !ok || !actionAttr.Required {
		t.Fatal("expected action to be a required string attribute")
	}
	if len(actionAttr.PlanModifiers) == 0 {
		t.Fatal("expected action to have a RequiresReplace plan modifier: TaskUpdate has no action field")
	}
}

func TestTaskResourceSchema_DestinationIDIsRequiredWithoutReplace(t *testing.T) {
	s := taskResourceSchema()

	destinationIDAttr, ok := s.Attributes["destination_id"].(resourceschema.StringAttribute)
	if !ok || !destinationIDAttr.Required {
		t.Fatal("expected destination_id to be a required string attribute")
	}
	if len(destinationIDAttr.PlanModifiers) != 0 {
		t.Fatal("expected destination_id to have no RequiresReplace plan modifier: TaskUpdate can change destinationID")
	}
}

func TestTaskResourceSchema_SubscriptionActionIsOptional(t *testing.T) {
	s := taskResourceSchema()

	subscriptionActionAttr, ok := s.Attributes["subscription_action"].(resourceschema.StringAttribute)
	if !ok || !subscriptionActionAttr.Optional {
		t.Fatal("expected subscription_action to be an optional string attribute")
	}
	if subscriptionActionAttr.Required || subscriptionActionAttr.Computed {
		t.Fatal("expected subscription_action to be neither required nor computed")
	}
	if len(subscriptionActionAttr.Validators) == 0 {
		t.Fatal("expected subscription_action to have a OneOf validator")
	}
}

func TestTaskResourceSchema_EnabledIsOptionalComputedWithDefault(t *testing.T) {
	s := taskResourceSchema()

	enabledAttr, ok := s.Attributes["enabled"].(resourceschema.BoolAttribute)
	if !ok {
		t.Fatal("expected enabled to be a bool attribute")
	}
	if !enabledAttr.Optional || !enabledAttr.Computed {
		t.Fatal("expected enabled to be optional and computed")
	}
	if enabledAttr.Default == nil {
		t.Fatal("expected enabled to have a static default")
	}
}

func TestTaskResourceSchema_FailureThresholdIsOptionalInt64(t *testing.T) {
	s := taskResourceSchema()

	failureThresholdAttr, ok := s.Attributes["failure_threshold"].(resourceschema.Int64Attribute)
	if !ok || !failureThresholdAttr.Optional {
		t.Fatal("expected failure_threshold to be an optional int64 attribute")
	}
	if failureThresholdAttr.Required || failureThresholdAttr.Computed {
		t.Fatal("expected failure_threshold to be neither required nor computed")
	}
}

func TestTaskResourceSchema_InputNotificationsPoliciesAreOptionalAndNotSensitive(t *testing.T) {
	s := taskResourceSchema()

	for _, name := range []string{"input", "notifications", "policies"} {
		attr, ok := s.Attributes[name].(resourceschema.StringAttribute)
		if !ok {
			t.Fatalf("expected %s to be a string attribute", name)
		}
		if !attr.Optional {
			t.Fatalf("expected %s to be optional", name)
		}
		if attr.Required || attr.Computed {
			t.Fatalf("expected %s to be neither required nor computed", name)
		}
		if attr.Sensitive {
			t.Fatalf("expected %s to not be sensitive: it is configuration, not a secret", name)
		}
	}
}

func TestTaskResourceSchema_CursorIsOptionalWithoutComputed(t *testing.T) {
	s := taskResourceSchema()

	cursorAttr, ok := s.Attributes["cursor"].(resourceschema.StringAttribute)
	if !ok || !cursorAttr.Optional {
		t.Fatal("expected cursor to be an optional string attribute")
	}
	if cursorAttr.Required || cursorAttr.Computed {
		t.Fatal("expected cursor to be neither required nor computed: it is never refreshed from the API, see flattenTask")
	}
}

func TestTaskResourceSchema_IDAndTaskIDAreComputed(t *testing.T) {
	s := taskResourceSchema()

	idAttr, ok := s.Attributes["id"].(resourceschema.StringAttribute)
	if !ok || !idAttr.Computed {
		t.Fatal("expected id to be a computed string attribute")
	}

	taskIDAttr, ok := s.Attributes["task_id"].(resourceschema.StringAttribute)
	if !ok || !taskIDAttr.Computed {
		t.Fatal("expected task_id to be a computed string attribute")
	}
}

func TestTaskResourceSchema_LastRunAndNextRunAreComputed(t *testing.T) {
	s := taskResourceSchema()

	for _, name := range []string{"last_run", "next_run", "created_at", "updated_at"} {
		attr, ok := s.Attributes[name].(resourceschema.StringAttribute)
		if !ok || !attr.Computed {
			t.Fatalf("expected %s to be a computed string attribute", name)
		}
		if attr.Required || attr.Optional {
			t.Fatalf("expected %s to be neither required nor optional", name)
		}
	}
}

func TestTaskDataSourceSchema_TaskIDIsRequired(t *testing.T) {
	s := taskDataSourceSchema()

	taskIDAttr, ok := s.Attributes["task_id"].(datasourceschema.StringAttribute)
	if !ok || !taskIDAttr.Required {
		t.Fatal("expected task_id to be a required string attribute")
	}

	inputAttr, ok := s.Attributes["input"].(datasourceschema.StringAttribute)
	if !ok || !inputAttr.Computed {
		t.Fatal("expected input to be a computed string attribute")
	}

	cursorAttr, ok := s.Attributes["cursor"].(datasourceschema.StringAttribute)
	if !ok || !cursorAttr.Computed {
		t.Fatal("expected cursor to be a computed string attribute in the data source")
	}

	enabledAttr, ok := s.Attributes["enabled"].(datasourceschema.BoolAttribute)
	if !ok || !enabledAttr.Computed {
		t.Fatal("expected enabled to be a computed bool attribute")
	}

	failureThresholdAttr, ok := s.Attributes["failure_threshold"].(datasourceschema.Int64Attribute)
	if !ok || !failureThresholdAttr.Computed {
		t.Fatal("expected failure_threshold to be a computed int64 attribute")
	}
}

func TestAllowedActionTypeStrings_MatchesEnum(t *testing.T) {
	// Assert the known baseline values are present rather than an exact
	// count, so adding a new action type upstream doesn't break this test.
	assertContains(t, "action types", allowedActionTypeStrings(),
		"replace", "save", "partial", "partialNoCreate", "append")
}
