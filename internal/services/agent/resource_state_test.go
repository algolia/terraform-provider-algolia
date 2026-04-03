package agent

import (
	"context"
	"testing"

	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestHydrateAgentResourceState_UsesRemotePublishStateAndPreservesDeletionProtection(t *testing.T) {
	ctx := context.Background()
	model := &AgentResourceModel{
		Publish:            types.BoolValue(true),
		DeletionProtection: types.BoolValue(false),
	}

	resp := &AgentResponse{
		ID:           "agent-123",
		Name:         "support-bot",
		Status:       "draft",
		Instructions: "Be helpful.",
		Tools:        []any{},
		CreatedAt:    "2026-01-01T00:00:00Z",
	}

	diags := hydrateAgentResourceState(ctx, resp, model.DeletionProtection, model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags.Errors())
	}

	if model.Publish.IsNull() || model.Publish.ValueBool() {
		t.Fatalf("expected publish to reflect remote draft status, got %#v", model.Publish)
	}

	if model.DeletionProtection.IsNull() || model.DeletionProtection.ValueBool() {
		t.Fatalf("expected deletion protection to remain false, got %#v", model.DeletionProtection)
	}
}

func TestHydrateImportedAgentResourceState_DefaultsDeletionProtection(t *testing.T) {
	ctx := context.Background()
	model := &AgentResourceModel{}

	resp := &AgentResponse{
		ID:           "agent-456",
		Name:         "support-bot",
		Status:       "published",
		Instructions: "Be helpful.",
		Tools:        []any{},
		CreatedAt:    "2026-01-01T00:00:00Z",
	}

	diags := hydrateImportedAgentResourceState(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags.Errors())
	}

	if model.Publish.IsNull() || !model.Publish.ValueBool() {
		t.Fatalf("expected publish to reflect remote published status, got %#v", model.Publish)
	}

	if model.DeletionProtection.IsNull() || !model.DeletionProtection.ValueBool() {
		t.Fatalf("expected import to default deletion protection to true, got %#v", model.DeletionProtection)
	}
}

func TestValidatePublishTransition_BlocksUnpublish(t *testing.T) {
	err := validatePublishTransition(
		AgentResourceModel{Status: types.StringValue("published")},
		AgentResourceModel{Publish: types.BoolValue(false)},
	)
	if err == nil {
		t.Fatal("expected unpublish validation to fail")
	}
}

func TestValidatePublishTransition_AllowsDraftUpdates(t *testing.T) {
	err := validatePublishTransition(
		AgentResourceModel{Status: types.StringValue("draft")},
		AgentResourceModel{Publish: types.BoolValue(false)},
	)
	if err != nil {
		t.Fatalf("expected draft update to pass validation, got %v", err)
	}
}

func TestAgentResourceSchema_LocalFlagsSupportDefaults(t *testing.T) {
	schema := agentResourceSchema()

	publishAttr, ok := schema.Attributes["publish"].(resourceschema.BoolAttribute)
	if !ok {
		t.Fatal("expected publish to be a bool attribute")
	}

	if !publishAttr.Optional || !publishAttr.Computed {
		t.Fatal("expected publish to remain optional and computed")
	}

	deletionProtectionAttr, ok := schema.Attributes["deletion_protection"].(resourceschema.BoolAttribute)
	if !ok {
		t.Fatal("expected deletion_protection to be a bool attribute")
	}

	if !deletionProtectionAttr.Optional || !deletionProtectionAttr.Computed {
		t.Fatal("expected deletion_protection to remain optional and computed")
	}
}

func TestAgentDataSourceSchema_DoesNotExposeDeletionProtection(t *testing.T) {
	schema := agentDataSourceSchema()

	if _, exists := schema.Attributes["deletion_protection"]; exists {
		t.Fatal("expected deletion_protection to be absent from the data source schema")
	}

	publishAttr, ok := schema.Attributes["publish"].(datasourceschema.BoolAttribute)
	if !ok {
		t.Fatal("expected publish to remain a bool attribute on the data source")
	}

	if !publishAttr.Computed {
		t.Fatal("expected data source publish to be computed")
	}
}
