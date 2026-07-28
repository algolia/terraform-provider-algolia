package agent

import (
	"context"
	"strings"
	"testing"

	agentStudio "github.com/algolia/algoliasearch-client-go/v4/algolia/agent-studio"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/utils"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

// TestFlattenAgentResponse_withDisplayResultsTool covers the first of the two
// tool variants that used to fall off the end of flattenTools' switch: an
// algolia_display_results tool read from the API has to land in state, or the
// next update - which replaces the agent's whole tool list - deletes it.
func TestFlattenAgentResponse_withDisplayResultsTool(t *testing.T) {
	ctx := context.Background()
	resp := &agentStudio.AgentWithVersionResponse{
		Id:           "agent-display-tool",
		Name:         "display-agent",
		Status:       agentStudio.AGENT_STATUS_DRAFT,
		Instructions: "Show results",
		Tools: []agentStudio.ToolConfigInput{
			*agentStudio.AlgoliaDisplayResultsToolConfigAsToolConfigInput(&agentStudio.AlgoliaDisplayResultsToolConfig{
				Name:               strPtr("display_results"),
				Type:               "algolia_display_results",
				MinGroups:          int32Value(1),
				MaxGroups:          int32Value(3),
				MinResultsPerGroup: int32Value(2),
			}),
		},
		CreatedAt: "2026-01-01T00:00:00Z",
	}

	model := &AgentResourceModel{}
	diags := flattenAgentResponse(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if model.ToolAlgoliaDisplayResults.IsNull() {
		t.Fatal("expected non-null tool_algolia_display_results")
	}

	var tools []ToolAlgoliaDisplayResultsModel
	if d := model.ToolAlgoliaDisplayResults.ElementsAs(ctx, &tools, false); d.HasError() {
		t.Fatalf("unexpected errors reading display results tools: %v", d.Errors())
	}

	if len(tools) != 1 {
		t.Fatalf("expected 1 display results tool, got %d", len(tools))
	}
	if tools[0].Name.ValueString() != "display_results" {
		t.Errorf("expected tool name 'display_results', got %q", tools[0].Name.ValueString())
	}
	if tools[0].MinGroups.ValueInt64() != 1 {
		t.Errorf("expected min_groups 1, got %d", tools[0].MinGroups.ValueInt64())
	}
	if tools[0].MaxGroups.ValueInt64() != 3 {
		t.Errorf("expected max_groups 3, got %d", tools[0].MaxGroups.ValueInt64())
	}
	if tools[0].MinResultsPerGroup.ValueInt64() != 2 {
		t.Errorf("expected min_results_per_group 2, got %d", tools[0].MinResultsPerGroup.ValueInt64())
	}
	// Unset in the response: must be null rather than a zero value.
	if !tools[0].MaxResultsPerGroup.IsNull() {
		t.Errorf("expected null max_results_per_group, got %d", tools[0].MaxResultsPerGroup.ValueInt64())
	}
}

// TestFlattenAgentResponse_displayResultsToolRoundTrip proves the new variant
// survives a write: reading it into state is only half the fix, since
// expandTools rebuilds the full tool list sent on every update.
func TestFlattenAgentResponse_displayResultsToolRoundTrip(t *testing.T) {
	ctx := context.Background()

	displayToolList := mustList(t, ctx, types.ObjectType{AttrTypes: algoliaDisplayResultsToolAttrTypes},
		[]ToolAlgoliaDisplayResultsModel{
			{
				Name:               types.StringValue("display_results"),
				MinGroups:          types.Int64Value(1),
				MaxGroups:          types.Int64Value(3),
				MinResultsPerGroup: types.Int64Null(),
				MaxResultsPerGroup: types.Int64Value(9),
			},
		})

	model := &AgentResourceModel{
		Name:                      types.StringValue("display-roundtrip"),
		Instructions:              types.StringValue("Show results"),
		ToolAlgoliaDisplayResults: displayToolList,
	}

	cfg, diags := expandAgentConfigCreate(ctx, model)
	if diags.HasError() {
		t.Fatalf("expand: %v", diags.Errors())
	}
	if len(cfg.Tools) != 1 {
		t.Fatalf("expected 1 tool after expand, got %d", len(cfg.Tools))
	}
	if cfg.Tools[0].AlgoliaDisplayResultsToolConfig == nil {
		t.Fatalf("expected an AlgoliaDisplayResultsToolConfig, got %#v", cfg.Tools[0])
	}
	if got := cfg.Tools[0].AlgoliaDisplayResultsToolConfig.Type; got != "algolia_display_results" {
		t.Errorf("expected type 'algolia_display_results', got %q", got)
	}
	if cfg.Tools[0].AlgoliaDisplayResultsToolConfig.MinResultsPerGroup != nil {
		t.Errorf("expected a null min_results_per_group to be omitted, got %d",
			*cfg.Tools[0].AlgoliaDisplayResultsToolConfig.MinResultsPerGroup)
	}

	out := &AgentResourceModel{}
	diags = flattenAgentResponse(ctx, &agentStudio.AgentWithVersionResponse{
		Id:           "display-roundtrip-id",
		Name:         "display-roundtrip",
		Status:       agentStudio.AGENT_STATUS_DRAFT,
		Instructions: "Show results",
		Tools:        cfg.Tools,
		CreatedAt:    "2026-01-01T00:00:00Z",
	}, out)
	if diags.HasError() {
		t.Fatalf("flatten: %v", diags.Errors())
	}

	var tools []ToolAlgoliaDisplayResultsModel
	if d := out.ToolAlgoliaDisplayResults.ElementsAs(ctx, &tools, false); d.HasError() {
		t.Fatalf("read display results tools: %v", d.Errors())
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 display results tool after roundtrip, got %d", len(tools))
	}
	if tools[0].MaxResultsPerGroup.ValueInt64() != 9 || !tools[0].MinResultsPerGroup.IsNull() {
		t.Fatalf("unexpected display results tool after roundtrip: %#v", tools[0])
	}
}

// TestFlattenAgentResponse_unknownToolConfigErrors covers the second variant
// the switch used to drop. UnknownToolConfig is the client's placeholder for a
// tool type it does not recognise; the provider has no schema that could hold
// its arbitrary properties, so it must refuse the read instead of silently
// dropping the tool from the agent on the next write.
func TestFlattenAgentResponse_unknownToolConfigErrors(t *testing.T) {
	ctx := context.Background()
	resp := &agentStudio.AgentWithVersionResponse{
		Id:           "agent-unknown-tool",
		Name:         "unknown-tool-agent",
		Status:       agentStudio.AGENT_STATUS_DRAFT,
		Instructions: "Use tools",
		Tools: []agentStudio.ToolConfigInput{
			*agentStudio.UnknownToolConfigAsToolConfigInput(agentStudio.UnknownToolConfig{
				Name: "some_future_tool",
				Type: "future_tool",
			}),
		},
		CreatedAt: "2026-01-01T00:00:00Z",
	}

	model := &AgentResourceModel{}
	diags := flattenAgentResponse(ctx, resp, model)
	if !diags.HasError() {
		t.Fatal("expected an error for an unsupported tool type, got none")
	}

	detail := diags.Errors()[0].Detail()
	if !strings.Contains(detail, "some_future_tool") || !strings.Contains(detail, "future_tool") {
		t.Errorf("expected the diagnostic to name the tool and its type, got %q", detail)
	}
}

// TestFlattenAgentResponse_unhandledToolVariantErrors exercises the switch's
// default arm: a ToolConfigInput whose populated variant the provider does not
// handle must fail loudly. A future client release adding a seventh variant
// hits this path, and the whole point is that it cannot pass silently.
func TestFlattenAgentResponse_unhandledToolVariantErrors(t *testing.T) {
	ctx := context.Background()
	resp := &agentStudio.AgentWithVersionResponse{
		Id:           "agent-unhandled-tool",
		Name:         "unhandled-tool-agent",
		Status:       agentStudio.AGENT_STATUS_DRAFT,
		Instructions: "Use tools",
		// No variant populated: stands in for a variant this provider
		// version has no case for.
		Tools:     []agentStudio.ToolConfigInput{{}},
		CreatedAt: "2026-01-01T00:00:00Z",
	}

	model := &AgentResourceModel{}
	diags := flattenAgentResponse(ctx, resp, model)
	if !diags.HasError() {
		t.Fatal("expected an error for an unhandled tool variant, got none")
	}

	if summary := diags.Errors()[0].Summary(); summary != "Unsupported agent tool variant" {
		t.Errorf("unexpected diagnostic summary %q", summary)
	}
}

// TestFlattenMCPHeaders covers the null-vs-empty contract for
// tool_mcp.headers: an Optional, non-Computed attribute whose planned value is
// the configuration verbatim, so mapping an API-empty map to null regardless of
// the prior value aborts the apply of an explicitly configured `headers = {}`.
func TestFlattenMCPHeaders(t *testing.T) {
	emptyMap := types.MapValueMust(types.StringType, map[string]attr.Value{})
	configuredMap := types.MapValueMust(types.StringType, map[string]attr.Value{
		"Authorization": types.StringValue("Bearer token"),
	})

	tests := []struct {
		name    string
		headers map[string]string
		prior   types.Map
		want    types.Map
	}{
		{
			name:    "api empty and prior null stays null",
			headers: nil,
			prior:   types.MapNull(types.StringType),
			want:    types.MapNull(types.StringType),
		},
		{
			name:    "api empty and prior known empty stays known empty",
			headers: map[string]string{},
			prior:   emptyMap,
			want:    emptyMap,
		},
		{
			name:    "api non-empty wins",
			headers: map[string]string{"Authorization": "Bearer token"},
			prior:   types.MapNull(types.StringType),
			want:    configuredMap,
		},
		{
			name:    "api empty and prior with entries is drift and becomes null",
			headers: nil,
			prior:   configuredMap,
			want:    types.MapNull(types.StringType),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, diags := flattenMCPHeaders(test.headers, test.prior)
			if diags.HasError() {
				t.Fatalf("unexpected errors: %v", diags.Errors())
			}
			if !got.Equal(test.want) {
				t.Fatalf("headers = %v, want %v", got, test.want)
			}
		})
	}
}

// TestFlattenAgentResponse_preservesConfiguredEmptyHeaders is the end-to-end
// version of the case above: the prior value reaches flattenMCPHeaders through
// the model's existing tool_mcp list, matched by tool name.
func TestFlattenAgentResponse_preservesConfiguredEmptyHeaders(t *testing.T) {
	ctx := context.Background()

	emptyMap := types.MapValueMust(types.StringType, map[string]attr.Value{})
	priorMCP := mustList(t, ctx, types.ObjectType{AttrTypes: mcpToolAttrTypes}, []ToolMCPModel{
		{
			Name:         types.StringValue("my_mcp"),
			URL:          types.StringValue("https://mcp.example.com"),
			Transport:    types.StringValue("streamable_http"),
			Headers:      emptyMap,
			AllowedTools: types.ListNull(types.ObjectType{AttrTypes: mcpAllowedToolAttrTypes}),
		},
	})

	// The API echoes back an empty header map, as it does for a tool created
	// with `headers = {}` (expandMCPTool always sends a map).
	resp := &agentStudio.AgentWithVersionResponse{
		Id:           "agent-mcp",
		Name:         "mcp-agent",
		Status:       agentStudio.AGENT_STATUS_DRAFT,
		Instructions: "Use MCP",
		Tools: []agentStudio.ToolConfigInput{
			*agentStudio.McpServerToolConfigAsToolConfigInput(&agentStudio.McpServerToolConfig{
				Name:      "my_mcp",
				Type:      "mcp_tools",
				Url:       "https://mcp.example.com",
				Transport: strPtr("streamable_http"),
				Headers:   map[string]string{},
			}),
		},
		CreatedAt: "2026-01-01T00:00:00Z",
	}

	model := &AgentResourceModel{ToolMCP: priorMCP}
	if diags := flattenAgentResponse(ctx, resp, model); diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	var mcpTools []ToolMCPModel
	if d := model.ToolMCP.ElementsAs(ctx, &mcpTools, false); d.HasError() {
		t.Fatalf("read mcp tools: %v", d.Errors())
	}
	if len(mcpTools) != 1 {
		t.Fatalf("expected 1 mcp tool, got %d", len(mcpTools))
	}
	if mcpTools[0].Headers.IsNull() {
		t.Fatal("headers = null, want a known empty map (the configured value)")
	}
	if len(mcpTools[0].Headers.Elements()) != 0 {
		t.Fatalf("headers = %v, want a known empty map", mcpTools[0].Headers)
	}
}

// TestFlattenAgentResponse_nullHeadersStayNull is the counterpart: without a
// prior value (import, data source read, or a tool configured without headers)
// an empty API map is still null, so an unset attribute does not turn into a
// known empty map.
func TestFlattenAgentResponse_nullHeadersStayNull(t *testing.T) {
	ctx := context.Background()

	resp := &agentStudio.AgentWithVersionResponse{
		Id:           "agent-mcp-null-headers",
		Name:         "mcp-agent",
		Status:       agentStudio.AGENT_STATUS_DRAFT,
		Instructions: "Use MCP",
		Tools: []agentStudio.ToolConfigInput{
			*agentStudio.McpServerToolConfigAsToolConfigInput(&agentStudio.McpServerToolConfig{
				Name:    "my_mcp",
				Type:    "mcp_tools",
				Url:     "https://mcp.example.com",
				Headers: map[string]string{},
			}),
		},
		CreatedAt: "2026-01-01T00:00:00Z",
	}

	model := &AgentResourceModel{}
	if diags := flattenAgentResponse(ctx, resp, model); diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	var mcpTools []ToolMCPModel
	if d := model.ToolMCP.ElementsAs(ctx, &mcpTools, false); d.HasError() {
		t.Fatalf("read mcp tools: %v", d.Errors())
	}
	if len(mcpTools) != 1 {
		t.Fatalf("expected 1 mcp tool, got %d", len(mcpTools))
	}
	if !mcpTools[0].Headers.IsNull() {
		t.Fatalf("headers = %v, want null", mcpTools[0].Headers)
	}
}

// TestAgentResourceSchema_HeadersAreSensitive pins tool_mcp.headers as
// sensitive: values such as `Authorization: Bearer ...` are credentials, and
// Agent Studio returns them in full on read, so they must not be echoed in
// plan output.
func TestAgentResourceSchema_HeadersAreSensitive(t *testing.T) {
	block, ok := agentResourceSchema().Blocks["tool_mcp"].(schema.ListNestedBlock)
	if !ok {
		t.Fatalf("tool_mcp is not a ListNestedBlock: %T", agentResourceSchema().Blocks["tool_mcp"])
	}

	headers, ok := block.NestedObject.Attributes["headers"]
	if !ok {
		t.Fatal("tool_mcp has no headers attribute")
	}
	if !headers.IsSensitive() {
		t.Error("tool_mcp.headers must be marked sensitive: header values commonly carry credentials")
	}

	dsBlock, ok := agentDataSourceSchema().Blocks["tool_mcp"].(datasourceschema.ListNestedBlock)
	if !ok {
		t.Fatalf("data source tool_mcp is not a ListNestedBlock: %T", agentDataSourceSchema().Blocks["tool_mcp"])
	}
	dsHeaders, ok := dsBlock.NestedObject.Attributes["headers"]
	if !ok {
		t.Fatal("data source tool_mcp has no headers attribute")
	}
	if !dsHeaders.IsSensitive() {
		t.Error("data source tool_mcp.headers must be marked sensitive")
	}
}

func int32Value(v int32) *int32 {
	return &v
}

func mustList[T any](t *testing.T, ctx context.Context, elemType types.ObjectType, items []T) types.List {
	t.Helper()
	list, diags := types.ListValueFrom(ctx, elemType, items)
	if diags.HasError() {
		t.Fatalf("build list: %v", diags.Errors())
	}
	return list
}
