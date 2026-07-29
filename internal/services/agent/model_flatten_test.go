package agent

import (
	"context"
	"encoding/json"
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
	diags := flattenAgentResponse(ctx, agentDocumentOf(t, resp), model)
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
	diags := flattenAgentResponse(ctx, agentDocumentOf(t, resp), model)
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
	diags := flattenAgentResponse(ctx, agentDocumentOf(t, resp), model)
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
	diags := flattenAgentResponse(ctx, agentDocumentOf(t, resp), model)
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

	// Flatten the echoed request back and confirm the client-side and search
	// tools round-trip.
	out := &AgentResourceModel{}
	diags = flattenAgentResponse(ctx, agentDocumentFromJSON(t, echoAgentResponse(t, cfg)), out)
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

// realisticInputSchema is a JSON Schema of the shape users actually write:
// pretty-printed, and carrying keywords agentStudio.ClientToolsArgsSchema does
// not model. input_schema is Required, so it has no Computed escape hatch: the
// applied value must equal these bytes exactly.
const realisticInputSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "order_id": {
      "type": "string",
      "description": "The order to look up."
    },
    "channels": {
      "type": "array",
      "items": {
        "type": "string",
        "enum": ["email", "sms"]
      }
    }
  },
  "required": ["order_id"]
}`

// TestExpandFlattenRoundTrip_preservesInputSchema pins the whole contract for
// tool_client_side.input_schema: every keyword reaches Algolia, and the
// configured bytes are what land in state.
//
// Both halves are needed. Encoding the schema through
// agentStudio.ClientToolsArgsSchema, which models only type/properties/required,
// dropped every other top-level keyword - `$schema` and `additionalProperties`
// here - before the request was even sent; re-encoding the response then also
// collapsed the user's formatting. Either way the applied value differed from the
// planned one and Terraform aborted the apply with "Provider produced
// inconsistent result after apply".
func TestExpandFlattenRoundTrip_preservesInputSchema(t *testing.T) {
	ctx := context.Background()

	model := &AgentResourceModel{
		Name:         types.StringValue("schema-agent"),
		Instructions: types.StringValue("Use the tool"),
		ToolClientSide: mustList(t, ctx, types.ObjectType{AttrTypes: clientSideToolAttrTypes}, []ToolClientSideModel{
			{
				Name:        types.StringValue("get_order"),
				Description: types.StringValue("Get order status"),
				InputSchema: types.StringValue(realisticInputSchema),
			},
		}),
	}

	out, request := roundTripAgentModel(t, ctx, model)

	// Expand side: the keywords the client does not model have to be in the
	// outbound request, or they never reach Algolia at all.
	for _, keyword := range []string{"$schema", "additionalProperties", "The order to look up.", "enum"} {
		if !strings.Contains(request, keyword) {
			t.Errorf("outbound request dropped %q:\n%s", keyword, request)
		}
	}

	var clientTools []ToolClientSideModel
	if d := out.ToolClientSide.ElementsAs(ctx, &clientTools, false); d.HasError() {
		t.Fatalf("read client tools: %v", d.Errors())
	}
	if len(clientTools) != 1 {
		t.Fatalf("expected 1 client tool, got %d", len(clientTools))
	}

	// Flatten side: byte-for-byte, including the indentation.
	if got := clientTools[0].InputSchema.ValueString(); got != realisticInputSchema {
		t.Errorf("input_schema was not preserved verbatim:\n got: %s\nwant: %s", got, realisticInputSchema)
	}
}

// TestExpandFlattenRoundTrip_preservesUnmodelledSearchParameters is the same
// story for tool_algolia_search.index.search_parameters, which used to be
// round-tripped through the typed agentStudio.SearchParameters: any parameter
// Algolia ships ahead of a client regeneration was dropped from the request and
// from state.
func TestExpandFlattenRoundTrip_preservesUnmodelledSearchParameters(t *testing.T) {
	ctx := context.Background()

	const searchParameters = `{"hitsPerPage":10,"aFutureParameter":{"enabled":true}}`

	model := &AgentResourceModel{
		Name:         types.StringValue("search-agent"),
		Instructions: types.StringValue("Search things"),
		ToolAlgoliaSearch: mustList(t, ctx, types.ObjectType{AttrTypes: algoliaSearchToolAttrTypes}, []ToolAlgoliaSearchModel{
			{
				Name: types.StringValue("search_products"),
				Indices: mustList(t, ctx, types.ObjectType{AttrTypes: algoliaSearchIndexAttrTypes}, []AlgoliaSearchIndexModel{
					{
						Name:                types.StringValue("products"),
						Description:         types.StringValue("Product catalog"),
						EnhancedDescription: types.StringNull(),
						SearchParameters:    types.StringValue(searchParameters),
					},
				}),
			},
		}),
	}

	out, request := roundTripAgentModel(t, ctx, model)

	if !strings.Contains(request, "aFutureParameter") {
		t.Errorf("outbound request dropped an unmodelled search parameter:\n%s", request)
	}

	var searchTools []ToolAlgoliaSearchModel
	if d := out.ToolAlgoliaSearch.ElementsAs(ctx, &searchTools, false); d.HasError() {
		t.Fatalf("read search tools: %v", d.Errors())
	}
	var indices []AlgoliaSearchIndexModel
	if d := searchTools[0].Indices.ElementsAs(ctx, &indices, false); d.HasError() {
		t.Fatalf("read indices: %v", d.Errors())
	}
	if len(indices) != 1 {
		t.Fatalf("expected 1 index, got %d", len(indices))
	}
	if got := indices[0].SearchParameters.ValueString(); got != searchParameters {
		t.Errorf("search_parameters = %s, want the configured %s", got, searchParameters)
	}
}

// TestExpandFlattenRoundTrip_preservesEmptyJSONObjects covers `config` and
// `predefined_recommend_parameters`, which both used to turn a configured empty
// object into null - a planned "{}" applying as null aborts the apply.
func TestExpandFlattenRoundTrip_preservesEmptyJSONObjects(t *testing.T) {
	ctx := context.Background()

	model := &AgentResourceModel{
		Name:         types.StringValue("empty-objects-agent"),
		Instructions: types.StringValue("Recommend things"),
		Config:       types.StringValue("{}"),
		ToolAlgoliaRecommend: mustList(t, ctx, types.ObjectType{AttrTypes: algoliaRecommendToolAttrTypes}, []ToolAlgoliaRecommendModel{
			{
				Name: types.StringValue("recommend_products"),
				AllowedConfigs: mustList(t, ctx, types.ObjectType{AttrTypes: algoliaRecommendConfigAttrTypes}, []AlgoliaRecommendConfigModel{
					{
						Index:       types.StringValue("products"),
						ModelName:   types.StringValue("bought-together"),
						Description: types.StringNull(),
					},
				}),
				PredefinedRecommendParameters: types.StringValue("{}"),
			},
		}),
	}

	out, _ := roundTripAgentModel(t, ctx, model)

	if got := out.Config.ValueString(); got != "{}" {
		t.Errorf("config = %q, want the configured %q", got, "{}")
	}

	var recommendTools []ToolAlgoliaRecommendModel
	if d := out.ToolAlgoliaRecommend.ElementsAs(ctx, &recommendTools, false); d.HasError() {
		t.Fatalf("read recommend tools: %v", d.Errors())
	}
	if len(recommendTools) != 1 {
		t.Fatalf("expected 1 recommend tool, got %d", len(recommendTools))
	}
	if got := recommendTools[0].PredefinedRecommendParameters.ValueString(); got != "{}" {
		t.Errorf("predefined_recommend_parameters = %q, want the configured %q", got, "{}")
	}
}

// TestFlattenAgentResponse_keepsConfiguredJSONFormatting is the end-to-end half
// of TestFlattenJSONDocument: an API that stores the same data but answers with
// its own formatting and key order must not disturb what the user wrote. It also
// pins the wiring that carries the prior values in, which matches tools by name
// and indices by index name rather than by position.
func TestFlattenAgentResponse_keepsConfiguredJSONFormatting(t *testing.T) {
	ctx := context.Background()

	const configuredSchema = "{\n  \"properties\": {\n    \"order_id\": { \"type\": \"string\" }\n  },\n  \"type\": \"object\"\n}"
	const configuredParameters = "{\n  \"hitsPerPage\": 10\n}"

	model := &AgentResourceModel{
		ToolClientSide: mustList(t, ctx, types.ObjectType{AttrTypes: clientSideToolAttrTypes}, []ToolClientSideModel{
			{
				Name:        types.StringValue("get_order"),
				Description: types.StringValue("Get order status"),
				InputSchema: types.StringValue(configuredSchema),
			},
		}),
		ToolAlgoliaSearch: mustList(t, ctx, types.ObjectType{AttrTypes: algoliaSearchToolAttrTypes}, []ToolAlgoliaSearchModel{
			{
				Name: types.StringValue("search_products"),
				Indices: mustList(t, ctx, types.ObjectType{AttrTypes: algoliaSearchIndexAttrTypes}, []AlgoliaSearchIndexModel{
					{
						Name:                types.StringValue("products"),
						Description:         types.StringValue("Product catalog"),
						EnhancedDescription: types.StringNull(),
						SearchParameters:    types.StringValue(configuredParameters),
					},
				}),
			},
		}),
	}

	// The same data, compacted and with reordered keys, and with the null-valued
	// parameters Algolia fills the search parameter schema with.
	body := `{
	  "id": "format-agent",
	  "name": "format-agent",
	  "status": "draft",
	  "instructions": "Use the tools",
	  "createdAt": "2026-01-01T00:00:00Z",
	  "tools": [
	    {
	      "name": "get_order",
	      "type": "client_side",
	      "description": "Get order status",
	      "inputSchema": {"type":"object","properties":{"order_id":{"type":"string"}}}
	    },
	    {
	      "name": "search_products",
	      "type": "algolia_search_index",
	      "indices": [
	        {
	          "index": "products",
	          "description": "Product catalog",
	          "searchParameters": {"hitsPerPage":10,"query":null,"filters":null}
	        }
	      ]
	    }
	  ]
	}`

	if diags := flattenAgentResponse(ctx, agentDocumentFromJSON(t, body), model); diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	var clientTools []ToolClientSideModel
	if d := model.ToolClientSide.ElementsAs(ctx, &clientTools, false); d.HasError() {
		t.Fatalf("read client tools: %v", d.Errors())
	}
	if got := clientTools[0].InputSchema.ValueString(); got != configuredSchema {
		t.Errorf("input_schema = %s, want the configured %s", got, configuredSchema)
	}

	var searchTools []ToolAlgoliaSearchModel
	if d := model.ToolAlgoliaSearch.ElementsAs(ctx, &searchTools, false); d.HasError() {
		t.Fatalf("read search tools: %v", d.Errors())
	}
	var indices []AlgoliaSearchIndexModel
	if d := searchTools[0].Indices.ElementsAs(ctx, &indices, false); d.HasError() {
		t.Fatalf("read indices: %v", d.Errors())
	}
	if got := indices[0].SearchParameters.ValueString(); got != configuredParameters {
		t.Errorf("search_parameters = %s, want the configured %s", got, configuredParameters)
	}
}

// algoliaRewrittenAgentBody is a response body in the shape Agent Studio really
// answers with, recorded against the live API for a client_side tool, a search
// tool and a config that were all written with more in them than comes back:
//
//   - inputSchema lost `$schema` and `additionalProperties`
//   - searchParameters lost `aFutureParameter` and gained the full parameter
//     schema with every unset field explicitly null (abridged here)
//   - config gained `enableAlgoliaMcp`
//
// Each of those is enough to fail an apply, so this body is the regression
// fixture for the whole class.
const algoliaRewrittenAgentBody = `{
  "id": "rewritten-agent",
  "name": "rewritten-agent",
  "status": "draft",
  "instructions": "Use the tools",
  "createdAt": "2026-01-01T00:00:00Z",
  "config": {"temperature":0.5,"enableAlgoliaMcp":true},
  "tools": [
    {
      "name": "get_order",
      "type": "client_side",
      "description": "Get order status",
      "inputSchema": {"type":"object","properties":{"order_id":{"type":"string"}},"required":["order_id"]}
    },
    {
      "name": "search_products",
      "type": "algolia_search_index",
      "description": "products: Product catalog",
      "indices": [
        {
          "index": "products",
          "description": "Product catalog",
          "enhancedDescription": "",
          "searchParameters": {"hitsPerPage":10,"query":null,"filters":null,"distinct":null},
          "searchControls": null
        }
      ]
    }
  ]
}`

// TestFlattenAgentResponse_keepsConfiguredValuesAlgoliaRewrites is the unit-level
// reproduction of the acceptance failure. Algolia does not return these three
// documents the way they were written, so the configured value has to win or
// every apply of a configured value is rejected as an inconsistent result.
func TestFlattenAgentResponse_keepsConfiguredValuesAlgoliaRewrites(t *testing.T) {
	ctx := context.Background()

	const configuredSchema = `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,"properties":{"order_id":{"type":"string"}},"required":["order_id"]}`
	const configuredParameters = `{"hitsPerPage":10,"aFutureParameter":{"enabled":true}}`
	const configuredConfig = `{"temperature":0.5}`

	model := &AgentResourceModel{
		Config: types.StringValue(configuredConfig),
		ToolClientSide: mustList(t, ctx, types.ObjectType{AttrTypes: clientSideToolAttrTypes}, []ToolClientSideModel{
			{
				Name:        types.StringValue("get_order"),
				Description: types.StringValue("Get order status"),
				InputSchema: types.StringValue(configuredSchema),
			},
		}),
		ToolAlgoliaSearch: mustList(t, ctx, types.ObjectType{AttrTypes: algoliaSearchToolAttrTypes}, []ToolAlgoliaSearchModel{
			{
				Name: types.StringValue("search_products"),
				Indices: mustList(t, ctx, types.ObjectType{AttrTypes: algoliaSearchIndexAttrTypes}, []AlgoliaSearchIndexModel{
					{
						Name:                types.StringValue("products"),
						Description:         types.StringValue("Product catalog"),
						EnhancedDescription: types.StringNull(),
						SearchParameters:    types.StringValue(configuredParameters),
					},
				}),
			},
		}),
	}

	if diags := flattenAgentResponse(ctx, agentDocumentFromJSON(t, algoliaRewrittenAgentBody), model); diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if got := model.Config.ValueString(); got != configuredConfig {
		t.Errorf("config = %s, want the configured %s", got, configuredConfig)
	}

	var clientTools []ToolClientSideModel
	if d := model.ToolClientSide.ElementsAs(ctx, &clientTools, false); d.HasError() {
		t.Fatalf("read client tools: %v", d.Errors())
	}
	if got := clientTools[0].InputSchema.ValueString(); got != configuredSchema {
		t.Errorf("input_schema = %s, want the configured %s", got, configuredSchema)
	}

	var searchTools []ToolAlgoliaSearchModel
	if d := model.ToolAlgoliaSearch.ElementsAs(ctx, &searchTools, false); d.HasError() {
		t.Fatalf("read search tools: %v", d.Errors())
	}
	var indices []AlgoliaSearchIndexModel
	if d := searchTools[0].Indices.ElementsAs(ctx, &indices, false); d.HasError() {
		t.Fatalf("read indices: %v", d.Errors())
	}
	if got := indices[0].SearchParameters.ValueString(); got != configuredParameters {
		t.Errorf("search_parameters = %s, want the configured %s", got, configuredParameters)
	}
}

// TestFlattenAgentResponse_withoutPriorStoresAlgoliaView is the other half of the
// same rule: with nothing configured to preserve - an import or a data source read
// - Algolia's own document is what lands, and for search_parameters its null
// parameters are stripped so the value stays readable.
func TestFlattenAgentResponse_withoutPriorStoresAlgoliaView(t *testing.T) {
	ctx := context.Background()

	model := &AgentResourceModel{}
	if diags := flattenAgentResponse(ctx, agentDocumentFromJSON(t, algoliaRewrittenAgentBody), model); diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if got := model.Config.ValueString(); got != `{"temperature":0.5,"enableAlgoliaMcp":true}` {
		t.Errorf("config = %s, want Algolia's document", got)
	}

	var clientTools []ToolClientSideModel
	if d := model.ToolClientSide.ElementsAs(ctx, &clientTools, false); d.HasError() {
		t.Fatalf("read client tools: %v", d.Errors())
	}
	if got := clientTools[0].InputSchema.ValueString(); !strings.Contains(got, `"order_id"`) ||
		strings.Contains(got, "$schema") {
		t.Errorf("input_schema = %s, want Algolia's stripped schema", got)
	}

	var searchTools []ToolAlgoliaSearchModel
	if d := model.ToolAlgoliaSearch.ElementsAs(ctx, &searchTools, false); d.HasError() {
		t.Fatalf("read search tools: %v", d.Errors())
	}
	var indices []AlgoliaSearchIndexModel
	if d := searchTools[0].Indices.ElementsAs(ctx, &indices, false); d.HasError() {
		t.Fatalf("read indices: %v", d.Errors())
	}
	if got := indices[0].SearchParameters.ValueString(); got != `{"hitsPerPage":10}` {
		t.Errorf("search_parameters = %s, want Algolia's document with nulls stripped", got)
	}
}

// TestFlattenConfiguredJSONDocument covers the rule for the documents Algolia
// rewrites. It is deliberately stricter than flattenJSONDocument: the configured
// value wins even when the API's document differs, because the API will not say
// what it stored.
func TestFlattenConfiguredJSONDocument(t *testing.T) {
	tests := []struct {
		name     string
		document string
		prior    types.String
		want     types.String
	}{
		{
			name:     "configured value wins over a rewritten document",
			document: `{"a":1}`,
			prior:    types.StringValue(`{"a":1,"b":2}`),
			want:     types.StringValue(`{"a":1,"b":2}`),
		},
		{
			name:     "configured value wins over an absent document",
			document: "",
			prior:    types.StringValue(`{"a":1}`),
			want:     types.StringValue(`{"a":1}`),
		},
		{
			name:     "no prior keeps the API document",
			document: `{"a":1}`,
			prior:    types.StringUnknown(),
			want:     types.StringValue(`{"a":1}`),
		},
		{
			name:     "no prior and no document is null",
			document: "null",
			prior:    types.StringNull(),
			want:     types.StringNull(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := flattenConfiguredJSONDocument(json.RawMessage(test.document), test.prior)
			if !got.Equal(test.want) {
				t.Fatalf("flattenConfiguredJSONDocument = %v, want %v", got, test.want)
			}
		})
	}
}

// TestFlattenAgentResponse_reportsRecommendParameterDrift pins the one attribute
// that deliberately keeps the stricter rule.
// predefined_recommend_parameters is stored opaquely by Algolia and comes back
// exactly as written, unmodelled keys included, so a document that really did
// change out of band must still be reported as drift rather than hidden behind
// the configured value.
func TestFlattenAgentResponse_reportsRecommendParameterDrift(t *testing.T) {
	ctx := context.Background()

	model := &AgentResourceModel{
		ToolAlgoliaRecommend: mustList(t, ctx, types.ObjectType{AttrTypes: algoliaRecommendToolAttrTypes}, []ToolAlgoliaRecommendModel{
			{
				Name:                          types.StringValue("recommend_products"),
				AllowedConfigs:                types.ListNull(types.ObjectType{AttrTypes: algoliaRecommendConfigAttrTypes}),
				PredefinedRecommendParameters: types.StringValue(`{"maxRecommendations":3}`),
			},
		}),
	}

	body := `{
	  "id": "a", "name": "a", "instructions": "i", "status": "draft", "createdAt": "t",
	  "tools": [
	    {
	      "name": "recommend_products",
	      "type": "algolia_recommend",
	      "allowedConfigs": [{"index":"products","modelName":"bought-together","description":""}],
	      "predefinedRecommendParameters": {"maxRecommendations":9}
	    }
	  ]
	}`

	if diags := flattenAgentResponse(ctx, agentDocumentFromJSON(t, body), model); diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	var recommendTools []ToolAlgoliaRecommendModel
	if d := model.ToolAlgoliaRecommend.ElementsAs(ctx, &recommendTools, false); d.HasError() {
		t.Fatalf("read recommend tools: %v", d.Errors())
	}
	if got := recommendTools[0].PredefinedRecommendParameters.ValueString(); got != `{"maxRecommendations":9}` {
		t.Errorf("predefined_recommend_parameters = %s, want the API's changed document", got)
	}
}

// TestFlattenJSONDocument covers the preserve-the-configured-bytes rule used for
// predefined_recommend_parameters, the one document Algolia returns faithfully.
func TestFlattenJSONDocument(t *testing.T) {
	tests := []struct {
		name     string
		document string
		prior    types.String
		want     types.String
	}{
		{
			name:     "absent document is null",
			document: "",
			prior:    types.StringValue(`{"a":1}`),
			want:     types.StringNull(),
		},
		{
			name:     "literal null is null",
			document: "null",
			prior:    types.StringNull(),
			want:     types.StringNull(),
		},
		{
			name:     "empty object stays an empty object",
			document: "{}",
			prior:    types.StringValue("{}"),
			want:     types.StringValue("{}"),
		},
		{
			name:     "configured formatting and key order win",
			document: `{"a":1,"b":2}`,
			prior:    types.StringValue("{\n  \"b\": 2,\n  \"a\": 1\n}"),
			want:     types.StringValue("{\n  \"b\": 2,\n  \"a\": 1\n}"),
		},
		{
			name:     "a different document is drift and replaces the prior",
			document: `{"a":2}`,
			prior:    types.StringValue(`{"a":1}`),
			want:     types.StringValue(`{"a":2}`),
		},
		{
			name:     "no prior value keeps the API document",
			document: `{"a":1}`,
			prior:    types.StringUnknown(),
			want:     types.StringValue(`{"a":1}`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := flattenJSONDocument(json.RawMessage(test.document), test.prior)
			if !got.Equal(test.want) {
				t.Fatalf("flattenJSONDocument = %v, want %v", got, test.want)
			}
		})
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
	diags := flattenAgentResponse(ctx, agentDocumentOf(t, resp), model)
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
	diags = flattenAgentResponse(ctx, agentDocumentOf(t, &agentStudio.AgentWithVersionResponse{
		Id:           "display-roundtrip-id",
		Name:         "display-roundtrip",
		Status:       agentStudio.AGENT_STATUS_DRAFT,
		Instructions: "Show results",
		Tools:        cfg.Tools,
		CreatedAt:    "2026-01-01T00:00:00Z",
	}), out)
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
	diags := flattenAgentResponse(ctx, agentDocumentOfTools(resp), model)
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
	diags := flattenAgentResponse(ctx, agentDocumentOfTools(resp), model)
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
	if diags := flattenAgentResponse(ctx, agentDocumentOf(t, resp), model); diags.HasError() {
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
	if diags := flattenAgentResponse(ctx, agentDocumentOf(t, resp), model); diags.HasError() {
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

// agentDocumentOf turns a typed response into the agentDocument the flatten path
// consumes, by encoding it and decoding it back the way the provider does for a
// real response body. Tests needing a payload the client's own models cannot
// express - an input_schema carrying unmodelled JSON Schema keywords, say - build
// the body themselves with agentDocumentFromJSON.
func agentDocumentOf(t *testing.T, resp *agentStudio.AgentWithVersionResponse) *agentDocument {
	t.Helper()

	body, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("encode agent response: %v", err)
	}

	return agentDocumentFromJSON(t, string(body))
}

// agentDocumentFromJSON decodes a raw agent payload exactly as a successful API
// call would.
func agentDocumentFromJSON(t *testing.T, body string) *agentDocument {
	t.Helper()

	doc, err := decodeAgentDocument([]byte(body))
	if err != nil {
		t.Fatalf("decode agent response: %v", err)
	}

	return doc
}

// agentDocumentOfTools builds an agentDocument straight from typed tools,
// skipping the JSON round trip agentDocumentOf performs.
//
// flattenTools switches on which ToolConfigInput variant is populated, and two
// of its arms cannot be reached through a payload at all: the client's oneOf
// decoder falls back to AlgoliaRecommendToolConfigInput for any object carrying a
// name and a type, so neither an unrecognised tool type nor an empty variant
// survives decoding. Those arms guard against a future client release, so they
// are exercised by populating the variant directly.
func agentDocumentOfTools(resp *agentStudio.AgentWithVersionResponse) *agentDocument {
	return &agentDocument{
		agent: resp,
		raw:   rawAgent{tools: make([]rawTool, len(resp.Tools))},
	}
}

// roundTripAgentModel expands a model into a create request, feeds the request
// back the way Agent Studio echoes it, and flattens the result onto a copy of the
// same model. The copy matters: Create and Update flatten onto the plan, so the
// configured values are the prior values the flatten path compares against.
// Returns the flattened model and the encoded request body.
func roundTripAgentModel(t *testing.T, ctx context.Context, model *AgentResourceModel) (*AgentResourceModel, string) {
	t.Helper()

	config, diags := expandAgentConfigCreate(ctx, model)
	if diags.HasError() {
		t.Fatalf("expand: %v", diags.Errors())
	}

	body := echoAgentResponse(t, config)

	out := *model
	if diags := flattenAgentResponse(ctx, agentDocumentFromJSON(t, body), &out); diags.HasError() {
		t.Fatalf("flatten: %v", diags.Errors())
	}

	return &out, body
}

// echoAgentResponse builds the payload Agent Studio answers a create request
// with: it stores the config and tool list it was sent and returns them
// unchanged. Round-trip tests go through the encoded request rather than through
// the typed AgentConfigCreate, because the bytes that leave the provider are the
// whole point - a key dropped on the way out is invisible to any assertion made
// on the typed value.
func echoAgentResponse(t *testing.T, config *agentStudio.AgentConfigCreate) string {
	t.Helper()

	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("encode create request: %v", err)
	}

	var request map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &request); err != nil {
		t.Fatalf("decode create request: %v", err)
	}

	response := map[string]any{
		"id":           "echo-id",
		"name":         request["name"],
		"instructions": request["instructions"],
		"status":       "draft",
		"createdAt":    "2026-01-01T00:00:00Z",
	}
	for _, field := range []string{"config", "tools"} {
		if value, ok := request[field]; ok {
			response[field] = value
		}
	}

	body, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("encode agent response: %v", err)
	}

	return string(body)
}

func mustList[T any](t *testing.T, ctx context.Context, elemType types.ObjectType, items []T) types.List {
	t.Helper()
	list, diags := types.ListValueFrom(ctx, elemType, items)
	if diags.HasError() {
		t.Fatalf("build list: %v", diags.Errors())
	}
	return list
}
