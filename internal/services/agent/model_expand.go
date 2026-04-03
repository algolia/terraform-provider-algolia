package agent

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// expandAgentRequest converts the Terraform model into an AgentRequest for the API.
func expandAgentRequest(ctx context.Context, model *AgentResourceModel) (*AgentRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	req := &AgentRequest{}

	if isKnown(model.Name) {
		v := model.Name.ValueString()
		req.Name = &v
	}
	if isKnown(model.Description) {
		v := model.Description.ValueString()
		req.Description = &v
	}
	if isKnown(model.Instructions) {
		v := model.Instructions.ValueString()
		req.Instructions = &v
	}
	if isKnown(model.SystemPrompt) {
		v := model.SystemPrompt.ValueString()
		req.SystemPrompt = &v
	}
	if isKnown(model.ProviderID) {
		v := model.ProviderID.ValueString()
		req.ProviderID = &v
	}
	if isKnown(model.Model) {
		v := model.Model.ValueString()
		req.Model = &v
	}
	if isKnown(model.TemplateType) {
		v := model.TemplateType.ValueString()
		req.TemplateType = &v
	}

	if isKnown(model.Config) {
		var configObj any
		if err := json.Unmarshal([]byte(model.Config.ValueString()), &configObj); err != nil {
			diags.AddError("Invalid config JSON", "Could not parse config: "+err.Error())
			return nil, diags
		}
		req.Config = configObj
	}

	tools, d := expandTools(ctx, model)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	req.Tools = tools

	return req, diags
}

// expandTools collects all tool blocks into a single []any for the API.
func expandTools(ctx context.Context, model *AgentResourceModel) ([]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	var tools []any

	// Algolia Search tools
	if isKnown(model.ToolAlgoliaSearch) {
		var searchTools []ToolAlgoliaSearchModel
		diags.Append(model.ToolAlgoliaSearch.ElementsAs(ctx, &searchTools, false)...)
		if diags.HasError() {
			return nil, diags
		}
		for _, st := range searchTools {
			tool, d := expandAlgoliaSearchTool(ctx, &st)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			tools = append(tools, tool)
		}
	}

	// Algolia Recommend tools
	if isKnown(model.ToolAlgoliaRecommend) {
		var recommendTools []ToolAlgoliaRecommendModel
		diags.Append(model.ToolAlgoliaRecommend.ElementsAs(ctx, &recommendTools, false)...)
		if diags.HasError() {
			return nil, diags
		}
		for _, rt := range recommendTools {
			tool, d := expandAlgoliaRecommendTool(ctx, &rt)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			tools = append(tools, tool)
		}
	}

	// Client-side tools
	if isKnown(model.ToolClientSide) {
		var clientTools []ToolClientSideModel
		diags.Append(model.ToolClientSide.ElementsAs(ctx, &clientTools, false)...)
		if diags.HasError() {
			return nil, diags
		}
		for _, ct := range clientTools {
			tool, d := expandClientSideTool(&ct)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			tools = append(tools, tool)
		}
	}

	// MCP tools
	if isKnown(model.ToolMCP) {
		var mcpTools []ToolMCPModel
		diags.Append(model.ToolMCP.ElementsAs(ctx, &mcpTools, false)...)
		if diags.HasError() {
			return nil, diags
		}
		for _, mt := range mcpTools {
			tool, d := expandMCPTool(ctx, &mt)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			tools = append(tools, tool)
		}
	}

	return tools, diags
}

func expandAlgoliaSearchTool(ctx context.Context, model *ToolAlgoliaSearchModel) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	tool := map[string]any{
		"type": "algolia_search_index",
		"name": model.Name.ValueString(),
	}

	var indices []AlgoliaSearchIndexModel
	diags.Append(model.Indices.ElementsAs(ctx, &indices, false)...)
	if diags.HasError() {
		return nil, diags
	}

	var apiIndices []map[string]any
	for _, idx := range indices {
		entry := map[string]any{
			"index":       idx.Name.ValueString(),
			"description": idx.Description.ValueString(),
		}
		if isKnown(idx.EnhancedDescription) {
			entry["enhancedDescription"] = idx.EnhancedDescription.ValueString()
		}
		if isKnown(idx.SearchParameters) {
			var params any
			if err := json.Unmarshal([]byte(idx.SearchParameters.ValueString()), &params); err != nil {
				diags.AddError("Invalid search_parameters JSON", "Could not parse search_parameters: "+err.Error())
				return nil, diags
			}
			entry["searchParameters"] = params
		}
		apiIndices = append(apiIndices, entry)
	}
	tool["indices"] = apiIndices

	return tool, diags
}

func expandAlgoliaRecommendTool(ctx context.Context, model *ToolAlgoliaRecommendModel) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	tool := map[string]any{
		"type": "algolia_recommend",
		"name": model.Name.ValueString(),
	}

	var configs []AlgoliaRecommendConfigModel
	diags.Append(model.AllowedConfigs.ElementsAs(ctx, &configs, false)...)
	if diags.HasError() {
		return nil, diags
	}

	var apiConfigs []map[string]any
	for _, cfg := range configs {
		entry := map[string]any{
			"index":     cfg.Index.ValueString(),
			"modelName": cfg.ModelName.ValueString(),
		}
		if isKnown(cfg.Description) {
			entry["description"] = cfg.Description.ValueString()
		}
		apiConfigs = append(apiConfigs, entry)
	}
	tool["allowedConfigs"] = apiConfigs

	if isKnown(model.PredefinedRecommendParameters) {
		var params any
		if err := json.Unmarshal([]byte(model.PredefinedRecommendParameters.ValueString()), &params); err != nil {
			diags.AddError("Invalid predefined_recommend_parameters JSON", "Could not parse predefined_recommend_parameters: "+err.Error())
			return nil, diags
		}
		tool["predefinedRecommendParameters"] = params
	}

	return tool, diags
}

func expandClientSideTool(model *ToolClientSideModel) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	tool := map[string]any{
		"type":        "client_side",
		"name":        model.Name.ValueString(),
		"description": model.Description.ValueString(),
	}

	var inputSchema any
	if err := json.Unmarshal([]byte(model.InputSchema.ValueString()), &inputSchema); err != nil {
		diags.AddError("Invalid input_schema JSON", "Could not parse input_schema: "+err.Error())
		return nil, diags
	}
	tool["inputSchema"] = inputSchema

	return tool, diags
}

func expandMCPTool(ctx context.Context, model *ToolMCPModel) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	tool := map[string]any{
		"type":      "mcp_tools",
		"name":      model.Name.ValueString(),
		"url":       model.URL.ValueString(),
		"transport": model.Transport.ValueString(),
	}

	if isKnown(model.Headers) {
		headers := make(map[string]string)
		diags.Append(model.Headers.ElementsAs(ctx, &headers, false)...)
		if diags.HasError() {
			return nil, diags
		}
		tool["headers"] = headers
	} else {
		tool["headers"] = map[string]string{}
	}

	if isKnown(model.AllowedTools) {
		var allowedTools []MCPAllowedToolModel
		diags.Append(model.AllowedTools.ElementsAs(ctx, &allowedTools, false)...)
		if diags.HasError() {
			return nil, diags
		}

		if len(allowedTools) > 0 {
			allowedToolsMap := make(map[string]any)
			for _, at := range allowedTools {
				entry := map[string]any{}
				if isKnown(at.RequiresApproval) {
					entry["requiresApproval"] = at.RequiresApproval.ValueBool()
				}
				if isKnown(at.Alias) {
					entry["alias"] = at.Alias.ValueString()
				}
				allowedToolsMap[at.Name.ValueString()] = entry
			}
			tool["allowedTools"] = allowedToolsMap
		}
	}

	return tool, diags
}

// isKnown returns true if the value is neither null nor unknown.
func isKnown(v interface{ IsNull() bool; IsUnknown() bool }) bool {
	return !v.IsNull() && !v.IsUnknown()
}

// flattenNullableString converts a *string to a types.String.
func flattenNullableString(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}
