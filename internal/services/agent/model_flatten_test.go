package agent

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenAgentResponse_basic(t *testing.T) {
	ctx := context.Background()
	desc := "A test agent"
	sysPrompt := "Be safe"
	providerID := "prov-uuid"
	modelName := "gpt-4o"
	updatedAt := "2026-01-01T00:00:00Z"

	resp := &AgentResponse{
		ID:           "agent-uuid-123",
		Name:         "test-agent",
		Description:  &desc,
		Status:       "draft",
		ProviderID:   &providerID,
		Model:        &modelName,
		Instructions: "Be helpful",
		SystemPrompt: &sysPrompt,
		Config:       map[string]any{"temperature": 0.7},
		Tools:        []any{},
		CreatedAt:    "2026-01-01T00:00:00Z",
		UpdatedAt:    &updatedAt,
	}

	model := &AgentResourceModel{}
	diags := flattenAgentResponse(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if model.ID.ValueString() != "agent-uuid-123" {
		t.Errorf("expected id 'agent-uuid-123', got %q", model.ID.ValueString())
	}
	if model.Name.ValueString() != "test-agent" {
		t.Errorf("expected name 'test-agent', got %q", model.Name.ValueString())
	}
	if model.Description.ValueString() != "A test agent" {
		t.Errorf("expected description 'A test agent', got %q", model.Description.ValueString())
	}
	if model.Status.ValueString() != "draft" {
		t.Errorf("expected status 'draft', got %q", model.Status.ValueString())
	}
	if model.Config.ValueString() != `{"temperature":0.7}` {
		t.Errorf("expected config JSON, got %q", model.Config.ValueString())
	}
}

func TestFlattenAgentResponse_nullOptionals(t *testing.T) {
	ctx := context.Background()
	resp := &AgentResponse{
		ID:           "agent-uuid-456",
		Name:         "minimal-agent",
		Status:       "published",
		Instructions: "Do things",
		Config:       nil,
		Tools:        nil,
		CreatedAt:    "2026-01-01T00:00:00Z",
	}

	model := &AgentResourceModel{}
	diags := flattenAgentResponse(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if !model.Description.IsNull() {
		t.Errorf("expected null description, got %q", model.Description.ValueString())
	}
	if !model.SystemPrompt.IsNull() {
		t.Errorf("expected null system_prompt, got %q", model.SystemPrompt.ValueString())
	}
	if !model.Config.IsNull() {
		t.Errorf("expected null config, got %q", model.Config.ValueString())
	}
}

func TestFlattenAgentResponse_withClientSideTool(t *testing.T) {
	ctx := context.Background()
	resp := &AgentResponse{
		ID:           "agent-uuid-789",
		Name:         "tool-agent",
		Status:       "draft",
		Instructions: "Use tools",
		Config:       map[string]any{},
		Tools: []any{
			map[string]any{
				"type":        "client_side",
				"name":        "get_order",
				"description": "Get order status",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"order_id": map[string]any{"type": "string"},
					},
				},
			},
		},
		CreatedAt: "2026-01-01T00:00:00Z",
	}

	model := &AgentResourceModel{}
	diags := flattenAgentResponse(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if model.ToolClientSide.IsNull() {
		t.Fatal("expected non-null tool_client_side")
	}

	var clientTools []ToolClientSideModel
	diags = model.ToolClientSide.ElementsAs(ctx, &clientTools, false)
	if diags.HasError() {
		t.Fatalf("unexpected errors reading client tools: %v", diags.Errors())
	}

	if len(clientTools) != 1 {
		t.Fatalf("expected 1 client tool, got %d", len(clientTools))
	}
	if clientTools[0].Name.ValueString() != "get_order" {
		t.Errorf("expected tool name 'get_order', got %q", clientTools[0].Name.ValueString())
	}
	if clientTools[0].Description.ValueString() != "Get order status" {
		t.Errorf("expected tool description 'Get order status', got %q", clientTools[0].Description.ValueString())
	}
}

func TestFlattenAgentResponse_withAlgoliaSearchTool(t *testing.T) {
	ctx := context.Background()
	resp := &AgentResponse{
		ID:           "agent-search-tool",
		Name:         "search-agent",
		Status:       "draft",
		Instructions: "Search things",
		Config:       map[string]any{},
		Tools: []any{
			map[string]any{
				"type": "algolia_search_index",
				"name": "search_products",
				"indices": []any{
					map[string]any{
						"index":       "products",
						"description": "Product catalog",
					},
				},
			},
		},
		CreatedAt: "2026-01-01T00:00:00Z",
	}

	model := &AgentResourceModel{}
	diags := flattenAgentResponse(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if model.ToolAlgoliaSearch.IsNull() {
		t.Fatal("expected non-null tool_algolia_search")
	}

	var searchTools []ToolAlgoliaSearchModel
	diags = model.ToolAlgoliaSearch.ElementsAs(ctx, &searchTools, false)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if len(searchTools) != 1 {
		t.Fatalf("expected 1 search tool, got %d", len(searchTools))
	}
	if searchTools[0].Name.ValueString() != "search_products" {
		t.Errorf("expected tool name 'search_products', got %q", searchTools[0].Name.ValueString())
	}
}

func TestExpandFlattenRoundTrip(t *testing.T) {
	ctx := context.Background()

	// Build a model with all tool types.
	clientToolType := types.ObjectType{AttrTypes: clientSideToolAttrTypes}
	clientToolList, diags := types.ListValueFrom(ctx, clientToolType, []ToolClientSideModel{
		{
			Name:        types.StringValue("my_tool"),
			Description: types.StringValue("A custom tool"),
			InputSchema: types.StringValue(`{"type":"object","properties":{"id":{"type":"string"}}}`),
		},
	})
	if diags.HasError() {
		t.Fatalf("setup: %v", diags.Errors())
	}

	model := &AgentResourceModel{
		Name:                 types.StringValue("roundtrip-agent"),
		Instructions:         types.StringValue("Test roundtrip"),
		Description:          types.StringValue("Roundtrip test"),
		SystemPrompt:         types.StringNull(),
		ProviderID:           types.StringNull(),
		Model:                types.StringNull(),
		TemplateType:         types.StringNull(),
		Config:               types.StringValue(`{"temperature":0.5}`),
		ToolAlgoliaSearch:    types.ListNull(types.ObjectType{AttrTypes: algoliaSearchToolAttrTypes}),
		ToolAlgoliaRecommend: types.ListNull(types.ObjectType{AttrTypes: algoliaRecommendToolAttrTypes}),
		ToolClientSide:       clientToolList,
		ToolMCP:              types.ListNull(types.ObjectType{AttrTypes: mcpToolAttrTypes}),
	}

	// Expand
	req, diags := expandAgentRequest(ctx, model)
	if diags.HasError() {
		t.Fatalf("expand: %v", diags.Errors())
	}

	if len(req.Tools) != 1 {
		t.Fatalf("expected 1 tool after expand, got %d", len(req.Tools))
	}

	toolMap, ok := req.Tools[0].(map[string]any)
	if !ok {
		t.Fatalf("expected tool to be map[string]any, got %T", req.Tools[0])
	}
	if toolMap["type"] != "client_side" {
		t.Errorf("expected type 'client_side', got %v", toolMap["type"])
	}
	if toolMap["name"] != "my_tool" {
		t.Errorf("expected name 'my_tool', got %v", toolMap["name"])
	}
}
