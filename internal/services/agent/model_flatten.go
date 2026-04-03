package agent

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Attribute type maps for types.ObjectValueFrom conversions.
var algoliaSearchIndexAttrTypes = map[string]attr.Type{
	"name":                 types.StringType,
	"description":          types.StringType,
	"enhanced_description": types.StringType,
	"search_parameters":    types.StringType,
}

var algoliaSearchToolAttrTypes = map[string]attr.Type{
	"name": types.StringType,
	"index": types.ListType{ElemType: types.ObjectType{
		AttrTypes: algoliaSearchIndexAttrTypes,
	}},
}

var algoliaRecommendConfigAttrTypes = map[string]attr.Type{
	"index":      types.StringType,
	"model_name": types.StringType,
	"description": types.StringType,
}

var algoliaRecommendToolAttrTypes = map[string]attr.Type{
	"name": types.StringType,
	"allowed_config": types.ListType{ElemType: types.ObjectType{
		AttrTypes: algoliaRecommendConfigAttrTypes,
	}},
	"predefined_recommend_parameters": types.StringType,
}

var clientSideToolAttrTypes = map[string]attr.Type{
	"name":         types.StringType,
	"description":  types.StringType,
	"input_schema": types.StringType,
}

var mcpAllowedToolAttrTypes = map[string]attr.Type{
	"name":              types.StringType,
	"requires_approval": types.BoolType,
	"alias":             types.StringType,
}

var mcpToolAttrTypes = map[string]attr.Type{
	"name":      types.StringType,
	"url":       types.StringType,
	"transport": types.StringType,
	"headers":   types.MapType{ElemType: types.StringType},
	"allowed_tool": types.ListType{ElemType: types.ObjectType{
		AttrTypes: mcpAllowedToolAttrTypes,
	}},
}

// flattenAgentResponse populates the Terraform model from an API response.
func flattenAgentResponse(ctx context.Context, resp *AgentResponse, model *AgentResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(resp.ID)
	model.Name = types.StringValue(resp.Name)
	model.Description = flattenNullableString(resp.Description)
	model.Instructions = types.StringValue(resp.Instructions)
	model.SystemPrompt = flattenNullableString(resp.SystemPrompt)
	model.ProviderID = flattenNullableString(resp.ProviderID)
	model.Model = flattenNullableString(resp.Model)
	model.TemplateType = flattenNullableString(resp.TemplateType)
	model.Status = types.StringValue(resp.Status)
	model.CreatedAt = types.StringValue(resp.CreatedAt)
	model.UpdatedAt = flattenNullableString(resp.UpdatedAt)

	// Config
	if resp.Config != nil {
		configJSON, err := json.Marshal(resp.Config)
		if err != nil {
			diags.AddError("Error marshaling config", err.Error())
			return diags
		}
		configStr := string(configJSON)
		if configStr == "{}" || configStr == "null" {
			model.Config = types.StringNull()
		} else {
			model.Config = types.StringValue(configStr)
		}
	} else {
		model.Config = types.StringNull()
	}

	// Tools
	d := flattenTools(ctx, resp.Tools, model)
	diags.Append(d...)

	return diags
}

// flattenTools distributes API tools into the typed tool list attributes on the model.
func flattenTools(ctx context.Context, apiTools []any, model *AgentResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	var searchTools []ToolAlgoliaSearchModel
	var recommendTools []ToolAlgoliaRecommendModel
	var clientTools []ToolClientSideModel
	var mcpTools []ToolMCPModel

	for _, rawTool := range apiTools {
		toolMap, ok := rawTool.(map[string]any)
		if !ok {
			continue
		}

		toolType, _ := toolMap["type"].(string)
		switch toolType {
		case "algolia_search_index":
			t, d := flattenAlgoliaSearchTool(ctx, toolMap)
			diags.Append(d...)
			if t != nil {
				searchTools = append(searchTools, *t)
			}
		case "algolia_recommend":
			t, d := flattenAlgoliaRecommendTool(ctx, toolMap)
			diags.Append(d...)
			if t != nil {
				recommendTools = append(recommendTools, *t)
			}
		case "client_side":
			t, d := flattenClientSideTool(toolMap)
			diags.Append(d...)
			if t != nil {
				clientTools = append(clientTools, *t)
			}
		case "mcp_tools":
			t, d := flattenMCPTool(ctx, toolMap)
			diags.Append(d...)
			if t != nil {
				mcpTools = append(mcpTools, *t)
			}
		}
	}

	if diags.HasError() {
		return diags
	}

	// Convert to types.List
	searchToolType := types.ObjectType{AttrTypes: algoliaSearchToolAttrTypes}
	if len(searchTools) > 0 {
		list, d := types.ListValueFrom(ctx, searchToolType, searchTools)
		diags.Append(d...)
		model.ToolAlgoliaSearch = list
	} else {
		model.ToolAlgoliaSearch = types.ListNull(searchToolType)
	}

	recommendToolType := types.ObjectType{AttrTypes: algoliaRecommendToolAttrTypes}
	if len(recommendTools) > 0 {
		list, d := types.ListValueFrom(ctx, recommendToolType, recommendTools)
		diags.Append(d...)
		model.ToolAlgoliaRecommend = list
	} else {
		model.ToolAlgoliaRecommend = types.ListNull(recommendToolType)
	}

	clientToolType := types.ObjectType{AttrTypes: clientSideToolAttrTypes}
	if len(clientTools) > 0 {
		list, d := types.ListValueFrom(ctx, clientToolType, clientTools)
		diags.Append(d...)
		model.ToolClientSide = list
	} else {
		model.ToolClientSide = types.ListNull(clientToolType)
	}

	mcpToolType := types.ObjectType{AttrTypes: mcpToolAttrTypes}
	if len(mcpTools) > 0 {
		list, d := types.ListValueFrom(ctx, mcpToolType, mcpTools)
		diags.Append(d...)
		model.ToolMCP = list
	} else {
		model.ToolMCP = types.ListNull(mcpToolType)
	}

	return diags
}

func flattenAlgoliaSearchTool(_ context.Context, toolMap map[string]any) (*ToolAlgoliaSearchModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	model := &ToolAlgoliaSearchModel{
		Name: types.StringValue(getString(toolMap, "name")),
	}

	rawIndices, _ := toolMap["indices"].([]any)
	var indices []AlgoliaSearchIndexModel
	for _, raw := range rawIndices {
		idxMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		idx := AlgoliaSearchIndexModel{
			Name:        types.StringValue(getString(idxMap, "index")),
			Description: types.StringValue(getString(idxMap, "description")),
		}
		if v, ok := idxMap["enhancedDescription"].(string); ok && v != "" {
			idx.EnhancedDescription = types.StringValue(v)
		} else {
			idx.EnhancedDescription = types.StringNull()
		}
		if v, ok := idxMap["searchParameters"]; ok && v != nil {
			b, err := json.Marshal(v)
			if err != nil {
				diags.AddError("Error marshaling searchParameters", err.Error())
				return nil, diags
			}
			idx.SearchParameters = types.StringValue(string(b))
		} else {
			idx.SearchParameters = types.StringNull()
		}
		indices = append(indices, idx)
	}

	indexObjType := types.ObjectType{AttrTypes: algoliaSearchIndexAttrTypes}
	if len(indices) > 0 {
		list, d := types.ListValueFrom(context.Background(), indexObjType, indices)
		diags.Append(d...)
		model.Indices = list
	} else {
		model.Indices = types.ListNull(indexObjType)
	}

	return model, diags
}

func flattenAlgoliaRecommendTool(_ context.Context, toolMap map[string]any) (*ToolAlgoliaRecommendModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	model := &ToolAlgoliaRecommendModel{
		Name: types.StringValue(getString(toolMap, "name")),
	}

	rawConfigs, _ := toolMap["allowedConfigs"].([]any)
	var configs []AlgoliaRecommendConfigModel
	for _, raw := range rawConfigs {
		cfgMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		cfg := AlgoliaRecommendConfigModel{
			Index:     types.StringValue(getString(cfgMap, "index")),
			ModelName: types.StringValue(getString(cfgMap, "modelName")),
		}
		if v, ok := cfgMap["description"].(string); ok && v != "" {
			cfg.Description = types.StringValue(v)
		} else {
			cfg.Description = types.StringNull()
		}
		configs = append(configs, cfg)
	}

	configObjType := types.ObjectType{AttrTypes: algoliaRecommendConfigAttrTypes}
	if len(configs) > 0 {
		list, d := types.ListValueFrom(context.Background(), configObjType, configs)
		diags.Append(d...)
		model.AllowedConfigs = list
	} else {
		model.AllowedConfigs = types.ListNull(configObjType)
	}

	if v, ok := toolMap["predefinedRecommendParameters"]; ok && v != nil {
		b, err := json.Marshal(v)
		if err != nil {
			diags.AddError("Error marshaling predefinedRecommendParameters", err.Error())
			return nil, diags
		}
		model.PredefinedRecommendParameters = types.StringValue(string(b))
	} else {
		model.PredefinedRecommendParameters = types.StringNull()
	}

	return model, diags
}

func flattenClientSideTool(toolMap map[string]any) (*ToolClientSideModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	model := &ToolClientSideModel{
		Name:        types.StringValue(getString(toolMap, "name")),
		Description: types.StringValue(getString(toolMap, "description")),
	}

	if v, ok := toolMap["inputSchema"]; ok && v != nil {
		b, err := json.Marshal(v)
		if err != nil {
			diags.AddError("Error marshaling inputSchema", err.Error())
			return nil, diags
		}
		model.InputSchema = types.StringValue(string(b))
	} else {
		model.InputSchema = types.StringValue("{}")
	}

	return model, diags
}

func flattenMCPTool(_ context.Context, toolMap map[string]any) (*ToolMCPModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	model := &ToolMCPModel{
		Name:      types.StringValue(getString(toolMap, "name")),
		URL:       types.StringValue(getString(toolMap, "url")),
		Transport: types.StringValue(getString(toolMap, "transport")),
	}

	// Headers
	if rawHeaders, ok := toolMap["headers"].(map[string]any); ok && len(rawHeaders) > 0 {
		headers := make(map[string]string, len(rawHeaders))
		for k, v := range rawHeaders {
			headers[k], _ = v.(string)
		}
		headerMap, d := types.MapValueFrom(context.Background(), types.StringType, headers)
		diags.Append(d...)
		model.Headers = headerMap
	} else {
		model.Headers = types.MapNull(types.StringType)
	}

	// Allowed tools
	if rawAllowed, ok := toolMap["allowedTools"].(map[string]any); ok && len(rawAllowed) > 0 {
		var allowedTools []MCPAllowedToolModel
		for name, rawCfg := range rawAllowed {
			at := MCPAllowedToolModel{
				Name: types.StringValue(name),
			}
			if cfgMap, ok := rawCfg.(map[string]any); ok {
				if v, ok := cfgMap["requiresApproval"].(bool); ok {
					at.RequiresApproval = types.BoolValue(v)
				} else {
					at.RequiresApproval = types.BoolNull()
				}
				if v, ok := cfgMap["alias"].(string); ok && v != "" {
					at.Alias = types.StringValue(v)
				} else {
					at.Alias = types.StringNull()
				}
			} else {
				at.RequiresApproval = types.BoolNull()
				at.Alias = types.StringNull()
			}
			allowedTools = append(allowedTools, at)
		}

		allowedToolObjType := types.ObjectType{AttrTypes: mcpAllowedToolAttrTypes}
		list, d := types.ListValueFrom(context.Background(), allowedToolObjType, allowedTools)
		diags.Append(d...)
		model.AllowedTools = list
	} else {
		model.AllowedTools = types.ListNull(types.ObjectType{AttrTypes: mcpAllowedToolAttrTypes})
	}

	return model, diags
}

// getString safely extracts a string from a map.
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
