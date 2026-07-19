package agent

import (
	"context"
	"encoding/json"

	agentStudio "github.com/algolia/algoliasearch-client-go/v4/algolia/agent-studio"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/utils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// expandAgentConfigCreate converts the Terraform model into an AgentConfigCreate for the Create API.
func expandAgentConfigCreate(ctx context.Context, model *AgentResourceModel) (*agentStudio.AgentConfigCreate, diag.Diagnostics) {
	var diags diag.Diagnostics

	cfg := &agentStudio.AgentConfigCreate{
		Name:         model.Name.ValueString(),
		Instructions: model.Instructions.ValueString(),
	}

	if isKnown(model.Description) {
		cfg.Description = strPtr(model.Description.ValueString())
	}
	if isKnown(model.SystemPrompt) {
		cfg.SystemPrompt = strPtr(model.SystemPrompt.ValueString())
	}
	if isKnown(model.ProviderID) {
		cfg.ProviderId = strPtr(model.ProviderID.ValueString())
	}
	if isKnown(model.Model) {
		cfg.Model = strPtr(model.Model.ValueString())
	}
	if isKnown(model.TemplateType) {
		cfg.TemplateType = strPtr(model.TemplateType.ValueString())
	}

	config, d := expandConfig(model.Config)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	cfg.Config = config

	tools, d := expandTools(ctx, model)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	cfg.Tools = tools

	return cfg, diags
}

// expandAgentConfigUpdate converts the Terraform model into an AgentConfigUpdate for the Update API.
// Only known (non-null) scalar fields are set so that unset optionals are omitted from the PATCH body,
// preserving the behaviour of the previous hand-rolled client.
func expandAgentConfigUpdate(ctx context.Context, model *AgentResourceModel) (*agentStudio.AgentConfigUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics

	cfg := &agentStudio.AgentConfigUpdate{}

	if isKnown(model.Name) {
		cfg.Name = *utils.NewNullable(strPtr(model.Name.ValueString()))
	}
	if isKnown(model.Instructions) {
		cfg.Instructions = *utils.NewNullable(strPtr(model.Instructions.ValueString()))
	}
	if isKnown(model.Description) {
		cfg.Description = *utils.NewNullable(strPtr(model.Description.ValueString()))
	}
	if isKnown(model.SystemPrompt) {
		cfg.SystemPrompt = *utils.NewNullable(strPtr(model.SystemPrompt.ValueString()))
	}
	if isKnown(model.ProviderID) {
		cfg.ProviderId = *utils.NewNullable(strPtr(model.ProviderID.ValueString()))
	}
	if isKnown(model.Model) {
		cfg.Model = *utils.NewNullable(strPtr(model.Model.ValueString()))
	}
	if isKnown(model.TemplateType) {
		cfg.TemplateType = *utils.NewNullable(strPtr(model.TemplateType.ValueString()))
	}

	config, d := expandConfig(model.Config)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	cfg.Config = config

	tools, d := expandTools(ctx, model)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	cfg.Tools = tools

	return cfg, diags
}

// expandConfig parses the JSON-encoded config string into a map for the API.
func expandConfig(config types.String) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	if !isKnown(config) {
		return nil, diags
	}

	var configObj map[string]any
	if err := json.Unmarshal([]byte(config.ValueString()), &configObj); err != nil {
		diags.AddError("Invalid config JSON", "Could not parse config: "+err.Error())
		return nil, diags
	}

	return configObj, diags
}

// expandTools collects all tool blocks into a single []ToolConfigInput for the API.
func expandTools(ctx context.Context, model *AgentResourceModel) ([]agentStudio.ToolConfigInput, diag.Diagnostics) {
	var diags diag.Diagnostics
	var tools []agentStudio.ToolConfigInput

	// Algolia Search tools
	if isKnown(model.ToolAlgoliaSearch) {
		var searchTools []ToolAlgoliaSearchModel
		diags.Append(model.ToolAlgoliaSearch.ElementsAs(ctx, &searchTools, false)...)
		if diags.HasError() {
			return nil, diags
		}
		for i := range searchTools {
			tool, d := expandAlgoliaSearchTool(ctx, &searchTools[i])
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			tools = append(tools, *agentStudio.AlgoliaSearchToolConfigAsToolConfigInput(tool))
		}
	}

	// Algolia Recommend tools
	if isKnown(model.ToolAlgoliaRecommend) {
		var recommendTools []ToolAlgoliaRecommendModel
		diags.Append(model.ToolAlgoliaRecommend.ElementsAs(ctx, &recommendTools, false)...)
		if diags.HasError() {
			return nil, diags
		}
		for i := range recommendTools {
			tool, d := expandAlgoliaRecommendTool(ctx, &recommendTools[i])
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			tools = append(tools, *agentStudio.AlgoliaRecommendToolConfigInputAsToolConfigInput(tool))
		}
	}

	// Client-side tools
	if isKnown(model.ToolClientSide) {
		var clientTools []ToolClientSideModel
		diags.Append(model.ToolClientSide.ElementsAs(ctx, &clientTools, false)...)
		if diags.HasError() {
			return nil, diags
		}
		for i := range clientTools {
			tool, d := expandClientSideTool(&clientTools[i])
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			tools = append(tools, *agentStudio.ClientSideToolConfigAsToolConfigInput(tool))
		}
	}

	// MCP tools
	if isKnown(model.ToolMCP) {
		var mcpTools []ToolMCPModel
		diags.Append(model.ToolMCP.ElementsAs(ctx, &mcpTools, false)...)
		if diags.HasError() {
			return nil, diags
		}
		for i := range mcpTools {
			tool, d := expandMCPTool(ctx, &mcpTools[i])
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			tools = append(tools, *agentStudio.McpServerToolConfigAsToolConfigInput(tool))
		}
	}

	return tools, diags
}

func expandAlgoliaSearchTool(ctx context.Context, model *ToolAlgoliaSearchModel) (*agentStudio.AlgoliaSearchToolConfig, diag.Diagnostics) {
	var diags diag.Diagnostics

	var indices []AlgoliaSearchIndexModel
	diags.Append(model.Indices.ElementsAs(ctx, &indices, false)...)
	if diags.HasError() {
		return nil, diags
	}

	var apiIndices []agentStudio.AlgoliaSearchToolIndexConfig
	for _, idx := range indices {
		entry := agentStudio.AlgoliaSearchToolIndexConfig{
			Index:       idx.Name.ValueString(),
			Description: idx.Description.ValueString(),
		}
		if isKnown(idx.EnhancedDescription) {
			entry.EnhancedDescription = strPtr(idx.EnhancedDescription.ValueString())
		}
		if isKnown(idx.SearchParameters) {
			var params agentStudio.SearchParameters
			if err := json.Unmarshal([]byte(idx.SearchParameters.ValueString()), &params); err != nil {
				diags.AddError("Invalid search_parameters JSON", "Could not parse search_parameters: "+err.Error())
				return nil, diags
			}
			entry.SearchParameters = *utils.NewNullable(&params)
		}
		apiIndices = append(apiIndices, entry)
	}

	return &agentStudio.AlgoliaSearchToolConfig{
		Name:    model.Name.ValueString(),
		Type:    "algolia_search_index",
		Indices: apiIndices,
	}, diags
}

func expandAlgoliaRecommendTool(ctx context.Context, model *ToolAlgoliaRecommendModel) (*agentStudio.AlgoliaRecommendToolConfigInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	var configs []AlgoliaRecommendConfigModel
	diags.Append(model.AllowedConfigs.ElementsAs(ctx, &configs, false)...)
	if diags.HasError() {
		return nil, diags
	}

	var apiConfigs []agentStudio.AlgoliaRecommendToolIndexConfig
	for _, cfg := range configs {
		entry := agentStudio.AlgoliaRecommendToolIndexConfig{
			Index:     cfg.Index.ValueString(),
			ModelName: cfg.ModelName.ValueString(),
		}
		if isKnown(cfg.Description) {
			entry.Description = strPtr(cfg.Description.ValueString())
		}
		apiConfigs = append(apiConfigs, entry)
	}

	tool := &agentStudio.AlgoliaRecommendToolConfigInput{
		Name:           model.Name.ValueString(),
		Type:           "algolia_recommend",
		AllowedConfigs: apiConfigs,
	}

	if isKnown(model.PredefinedRecommendParameters) {
		var params map[string]any
		if err := json.Unmarshal([]byte(model.PredefinedRecommendParameters.ValueString()), &params); err != nil {
			diags.AddError("Invalid predefined_recommend_parameters JSON", "Could not parse predefined_recommend_parameters: "+err.Error())
			return nil, diags
		}
		tool.PredefinedRecommendParameters = params
	}

	return tool, diags
}

func expandClientSideTool(model *ToolClientSideModel) (*agentStudio.ClientSideToolConfig, diag.Diagnostics) {
	var diags diag.Diagnostics

	var inputSchema agentStudio.ClientToolsArgsSchema
	if err := json.Unmarshal([]byte(model.InputSchema.ValueString()), &inputSchema); err != nil {
		diags.AddError("Invalid input_schema JSON", "Could not parse input_schema: "+err.Error())
		return nil, diags
	}

	return &agentStudio.ClientSideToolConfig{
		Name:        model.Name.ValueString(),
		Type:        "client_side",
		Description: model.Description.ValueString(),
		InputSchema: inputSchema,
	}, diags
}

func expandMCPTool(ctx context.Context, model *ToolMCPModel) (*agentStudio.McpServerToolConfig, diag.Diagnostics) {
	var diags diag.Diagnostics

	tool := &agentStudio.McpServerToolConfig{
		Name: model.Name.ValueString(),
		Type: "mcp_tools",
		Url:  model.URL.ValueString(),
	}

	if isKnown(model.Transport) {
		tool.Transport = strPtr(model.Transport.ValueString())
	}

	headers := map[string]string{}
	if isKnown(model.Headers) {
		diags.Append(model.Headers.ElementsAs(ctx, &headers, false)...)
		if diags.HasError() {
			return nil, diags
		}
	}
	tool.Headers = headers

	if isKnown(model.AllowedTools) {
		var allowedTools []MCPAllowedToolModel
		diags.Append(model.AllowedTools.ElementsAs(ctx, &allowedTools, false)...)
		if diags.HasError() {
			return nil, diags
		}

		if len(allowedTools) > 0 {
			allowedToolsMap := make(map[string]agentStudio.ToolConfig, len(allowedTools))
			for _, at := range allowedTools {
				entry := &agentStudio.McpToolConfig{}
				if isKnown(at.RequiresApproval) {
					entry.RequiresApproval = *utils.NewNullable(boolPtr(at.RequiresApproval.ValueBool()))
				}
				if isKnown(at.Alias) {
					entry.Alias = *utils.NewNullable(strPtr(at.Alias.ValueString()))
				}
				allowedToolsMap[at.Name.ValueString()] = *agentStudio.McpToolConfigAsToolConfig(entry)
			}
			tool.AllowedTools = allowedToolsMap
		}
	}

	return tool, diags
}

// isKnown returns true if the value is neither null nor unknown.
func isKnown(v interface {
	IsNull() bool
	IsUnknown() bool
},
) bool {
	return !v.IsNull() && !v.IsUnknown()
}

func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
