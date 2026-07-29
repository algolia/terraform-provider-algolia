package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExpandAgentConfigCreate_basic(t *testing.T) {
	ctx := context.Background()
	model := &AgentResourceModel{
		Name:         types.StringValue("test-agent"),
		Instructions: types.StringValue("Be helpful"),
		Description:  types.StringValue("A test agent"),
		SystemPrompt: types.StringValue("System rules"),
		ProviderID:   types.StringValue("provider-uuid"),
		Model:        types.StringValue("gpt-4o"),
		TemplateType: types.StringValue("support"),
		Config:       types.StringValue(`{"temperature":0.7}`),
		// No tools
		ToolAlgoliaSearch:    types.ListNull(types.ObjectType{AttrTypes: algoliaSearchToolAttrTypes}),
		ToolAlgoliaRecommend: types.ListNull(types.ObjectType{AttrTypes: algoliaRecommendToolAttrTypes}),
		ToolClientSide:       types.ListNull(types.ObjectType{AttrTypes: clientSideToolAttrTypes}),
		ToolMCP:              types.ListNull(types.ObjectType{AttrTypes: mcpToolAttrTypes}),
	}

	cfg, diags := expandAgentConfigCreate(ctx, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if cfg.Name != "test-agent" {
		t.Errorf("expected name 'test-agent', got %q", cfg.Name)
	}
	if cfg.Instructions != "Be helpful" {
		t.Errorf("expected instructions 'Be helpful', got %q", cfg.Instructions)
	}
	if cfg.Description == nil || *cfg.Description != "A test agent" {
		t.Errorf("expected description 'A test agent', got %v", cfg.Description)
	}
	if cfg.SystemPrompt == nil || *cfg.SystemPrompt != "System rules" {
		t.Errorf("expected system_prompt 'System rules', got %v", cfg.SystemPrompt)
	}
	if cfg.ProviderId == nil || *cfg.ProviderId != "provider-uuid" {
		t.Errorf("expected provider_id 'provider-uuid', got %v", cfg.ProviderId)
	}
	if cfg.Model == nil || *cfg.Model != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got %v", cfg.Model)
	}
	if cfg.TemplateType == nil || *cfg.TemplateType != "support" {
		t.Errorf("expected template_type 'support', got %v", cfg.TemplateType)
	}
	if len(cfg.Tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(cfg.Tools))
	}

	// Numbers stay json.Number rather than becoming float64, so re-encoding the
	// config for the request cannot reformat a literal the user wrote.
	if temp, ok := cfg.Config["temperature"].(json.Number); !ok || temp.String() != "0.7" {
		t.Errorf("expected config.temperature=0.7 as a json.Number, got %#v", cfg.Config["temperature"])
	}
}

func TestExpandAgentConfigCreate_nullOptionals(t *testing.T) {
	ctx := context.Background()
	model := &AgentResourceModel{
		Name:                 types.StringValue("minimal-agent"),
		Instructions:         types.StringValue("Do things"),
		Description:          types.StringNull(),
		SystemPrompt:         types.StringNull(),
		ProviderID:           types.StringNull(),
		Model:                types.StringNull(),
		TemplateType:         types.StringNull(),
		Config:               types.StringNull(),
		ToolAlgoliaSearch:    types.ListNull(types.ObjectType{AttrTypes: algoliaSearchToolAttrTypes}),
		ToolAlgoliaRecommend: types.ListNull(types.ObjectType{AttrTypes: algoliaRecommendToolAttrTypes}),
		ToolClientSide:       types.ListNull(types.ObjectType{AttrTypes: clientSideToolAttrTypes}),
		ToolMCP:              types.ListNull(types.ObjectType{AttrTypes: mcpToolAttrTypes}),
	}

	cfg, diags := expandAgentConfigCreate(ctx, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if cfg.Description != nil {
		t.Errorf("expected nil description, got %v", cfg.Description)
	}
	if cfg.SystemPrompt != nil {
		t.Errorf("expected nil system_prompt, got %v", cfg.SystemPrompt)
	}
	if cfg.Config != nil {
		t.Errorf("expected nil config, got %v", cfg.Config)
	}
}

func TestExpandAgentConfigUpdate_setsKnownFieldsOnly(t *testing.T) {
	ctx := context.Background()
	model := &AgentResourceModel{
		Name:                 types.StringValue("updated-agent"),
		Instructions:         types.StringValue("New instructions"),
		Description:          types.StringNull(),
		SystemPrompt:         types.StringNull(),
		ProviderID:           types.StringNull(),
		Model:                types.StringNull(),
		TemplateType:         types.StringNull(),
		Config:               types.StringNull(),
		ToolAlgoliaSearch:    types.ListNull(types.ObjectType{AttrTypes: algoliaSearchToolAttrTypes}),
		ToolAlgoliaRecommend: types.ListNull(types.ObjectType{AttrTypes: algoliaRecommendToolAttrTypes}),
		ToolClientSide:       types.ListNull(types.ObjectType{AttrTypes: clientSideToolAttrTypes}),
		ToolMCP:              types.ListNull(types.ObjectType{AttrTypes: mcpToolAttrTypes}),
	}

	cfg, diags := expandAgentConfigUpdate(ctx, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if !cfg.Name.IsSet() || cfg.Name.Get() == nil || *cfg.Name.Get() != "updated-agent" {
		t.Errorf("expected name to be set to 'updated-agent', got %v", cfg.Name)
	}
	if !cfg.Instructions.IsSet() || cfg.Instructions.Get() == nil || *cfg.Instructions.Get() != "New instructions" {
		t.Errorf("expected instructions to be set, got %v", cfg.Instructions)
	}
	if cfg.Description.IsSet() {
		t.Errorf("expected description to be unset, got %v", cfg.Description)
	}
	if cfg.Config != nil {
		t.Errorf("expected nil config, got %v", cfg.Config)
	}
}
