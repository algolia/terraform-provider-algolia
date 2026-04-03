package agent

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExpandAgentRequest_basic(t *testing.T) {
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

	req, diags := expandAgentRequest(ctx, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if *req.Name != "test-agent" {
		t.Errorf("expected name 'test-agent', got %q", *req.Name)
	}
	if *req.Instructions != "Be helpful" {
		t.Errorf("expected instructions 'Be helpful', got %q", *req.Instructions)
	}
	if *req.Description != "A test agent" {
		t.Errorf("expected description 'A test agent', got %q", *req.Description)
	}
	if *req.SystemPrompt != "System rules" {
		t.Errorf("expected system_prompt 'System rules', got %q", *req.SystemPrompt)
	}
	if *req.ProviderID != "provider-uuid" {
		t.Errorf("expected provider_id 'provider-uuid', got %q", *req.ProviderID)
	}
	if *req.Model != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got %q", *req.Model)
	}
	if len(req.Tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(req.Tools))
	}

	configMap, ok := req.Config.(map[string]any)
	if !ok {
		t.Fatalf("expected config to be map[string]any, got %T", req.Config)
	}
	if temp, ok := configMap["temperature"].(float64); !ok || temp != 0.7 {
		t.Errorf("expected config.temperature=0.7, got %v", configMap["temperature"])
	}
}

func TestExpandAgentRequest_nullOptionals(t *testing.T) {
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

	req, diags := expandAgentRequest(ctx, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if req.Description != nil {
		t.Errorf("expected nil description, got %v", req.Description)
	}
	if req.SystemPrompt != nil {
		t.Errorf("expected nil system_prompt, got %v", req.SystemPrompt)
	}
	if req.Config != nil {
		t.Errorf("expected nil config, got %v", req.Config)
	}
}
