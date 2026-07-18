package agent

import (
	"context"
	"testing"

	agentStudio "github.com/algolia/algoliasearch-client-go/v4/algolia/agent-studio"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/utils"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenAgentResponse_basic(t *testing.T) {
	ctx := context.Background()

	resp := &agentStudio.AgentWithVersionResponse{
		Id:           "agent-uuid-123",
		Name:         "test-agent",
		Description:  *utils.NewNullable(strPtr("A test agent")),
		Status:       agentStudio.AGENT_STATUS_DRAFT,
		ProviderId:   *utils.NewNullable(strPtr("prov-uuid")),
		Model:        *utils.NewNullable(strPtr("gpt-4o")),
		Instructions: "Be helpful",
		SystemPrompt: *utils.NewNullable(strPtr("Be safe")),
		Config:       map[string]any{"temperature": 0.7},
		Tools:        []agentStudio.ToolConfigInput{},
		CreatedAt:    "2026-01-01T00:00:00Z",
		UpdatedAt:    *utils.NewNullable(strPtr("2026-01-01T00:00:00Z")),
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
	resp := &agentStudio.AgentWithVersionResponse{
		Id:           "agent-uuid-456",
		Name:         "minimal-agent",
		Status:       agentStudio.AGENT_STATUS_PUBLISHED,
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
	resp := &agentStudio.AgentWithVersionResponse{
		Id:           "agent-uuid-789",
		Name:         "tool-agent",
		Status:       agentStudio.AGENT_STATUS_DRAFT,
		Instructions: "Use tools",
		Config:       map[string]any{},
		Tools: []agentStudio.ToolConfigInput{
			*agentStudio.ClientSideToolConfigAsToolConfigInput(&agentStudio.ClientSideToolConfig{
				Name:        "get_order",
				Type:        "client_side",
				Description: "Get order status",
				InputSchema: agentStudio.ClientToolsArgsSchema{
					Type: strPtr("object"),
					Properties: map[string]any{
						"order_id": map[string]any{"type": "string"},
					},
				},
			}),
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
	resp := &agentStudio.AgentWithVersionResponse{
		Id:           "agent-search-tool",
		Name:         "search-agent",
		Status:       agentStudio.AGENT_STATUS_DRAFT,
		Instructions: "Search things",
		Config:       map[string]any{},
		Tools: []agentStudio.ToolConfigInput{
			*agentStudio.AlgoliaSearchToolConfigAsToolConfigInput(&agentStudio.AlgoliaSearchToolConfig{
				Name: "search_products",
				Type: "algolia_search_index",
				Indices: []agentStudio.AlgoliaSearchToolIndexConfig{
					{Index: "products", Description: "Product catalog"},
				},
			}),
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

	// Build a model with a client-side tool and search parameters.
	clientToolType := types.ObjectType{AttrTypes: clientSideToolAttrTypes}
	clientToolList, diags := types.ListValueFrom(ctx, clientToolType, []ToolClientSideModel{
		{
			Name:        types.StringValue("my_tool"),
			Description: types.StringValue("A custom tool"),
			InputSchema: types.StringValue(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`),
		},
	})
	if diags.HasError() {
		t.Fatalf("setup: %v", diags.Errors())
	}

	searchToolType := types.ObjectType{AttrTypes: algoliaSearchToolAttrTypes}
	searchToolList, diags := types.ListValueFrom(ctx, searchToolType, []ToolAlgoliaSearchModel{
		{
			Name: types.StringValue("search_products"),
			Indices: mustList(t, ctx, types.ObjectType{AttrTypes: algoliaSearchIndexAttrTypes}, []AlgoliaSearchIndexModel{
				{
					Name:                types.StringValue("products"),
					Description:         types.StringValue("Product catalog"),
					EnhancedDescription: types.StringNull(),
					SearchParameters:    types.StringValue(`{"hitsPerPage":10}`),
				},
			}),
		},
	})
	if diags.HasError() {
		t.Fatalf("setup search tool: %v", diags.Errors())
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
		ToolAlgoliaSearch:    searchToolList,
		ToolAlgoliaRecommend: types.ListNull(types.ObjectType{AttrTypes: algoliaRecommendToolAttrTypes}),
		ToolClientSide:       clientToolList,
		ToolMCP:              types.ListNull(types.ObjectType{AttrTypes: mcpToolAttrTypes}),
	}

	// Expand
	cfg, diags := expandAgentConfigCreate(ctx, model)
	if diags.HasError() {
		t.Fatalf("expand: %v", diags.Errors())
	}

	if len(cfg.Tools) != 2 {
		t.Fatalf("expected 2 tools after expand, got %d", len(cfg.Tools))
	}

	// Flatten back into a response and confirm the client-side + search tools round-trip.
	resp := &agentStudio.AgentWithVersionResponse{
		Id:           "roundtrip-id",
		Name:         "roundtrip-agent",
		Status:       agentStudio.AGENT_STATUS_DRAFT,
		Instructions: "Test roundtrip",
		Config:       map[string]any{"temperature": 0.5},
		Tools:        cfg.Tools,
		CreatedAt:    "2026-01-01T00:00:00Z",
	}

	out := &AgentResourceModel{}
	diags = flattenAgentResponse(ctx, resp, out)
	if diags.HasError() {
		t.Fatalf("flatten: %v", diags.Errors())
	}

	var clientTools []ToolClientSideModel
	if d := out.ToolClientSide.ElementsAs(ctx, &clientTools, false); d.HasError() {
		t.Fatalf("read client tools: %v", d.Errors())
	}
	if len(clientTools) != 1 || clientTools[0].Name.ValueString() != "my_tool" {
		t.Fatalf("unexpected client tools after roundtrip: %#v", clientTools)
	}

	var searchTools []ToolAlgoliaSearchModel
	if d := out.ToolAlgoliaSearch.ElementsAs(ctx, &searchTools, false); d.HasError() {
		t.Fatalf("read search tools: %v", d.Errors())
	}
	if len(searchTools) != 1 {
		t.Fatalf("expected 1 search tool after roundtrip, got %d", len(searchTools))
	}

	var indices []AlgoliaSearchIndexModel
	if d := searchTools[0].Indices.ElementsAs(ctx, &indices, false); d.HasError() {
		t.Fatalf("read indices: %v", d.Errors())
	}
	if len(indices) != 1 || indices[0].SearchParameters.ValueString() != `{"hitsPerPage":10}` {
		t.Fatalf("unexpected search parameters after roundtrip: %#v", indices)
	}
}

func mustList[T any](t *testing.T, ctx context.Context, elemType types.ObjectType, items []T) types.List {
	t.Helper()
	list, diags := types.ListValueFrom(ctx, elemType, items)
	if diags.HasError() {
		t.Fatalf("build list: %v", diags.Errors())
	}
	return list
}
