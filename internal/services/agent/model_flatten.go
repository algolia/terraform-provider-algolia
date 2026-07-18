package agent

import (
	"context"
	"encoding/json"
	"sort"

	agentStudio "github.com/algolia/algoliasearch-client-go/v4/algolia/agent-studio"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/utils"
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
	"index":       types.StringType,
	"model_name":  types.StringType,
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
func flattenAgentResponse(ctx context.Context, resp *agentStudio.AgentWithVersionResponse, model *AgentResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(resp.Id)
	model.Name = types.StringValue(resp.Name)
	model.Description = flattenNullableString(resp.Description)
	model.Instructions = types.StringValue(resp.Instructions)
	model.SystemPrompt = flattenNullableString(resp.SystemPrompt)
	model.ProviderID = flattenNullableString(resp.ProviderId)
	model.Model = flattenNullableString(resp.Model)
	model.TemplateType = flattenNullableString(resp.TemplateType)
	model.Status = types.StringValue(string(resp.Status))
	model.CreatedAt = types.StringValue(resp.CreatedAt)
	model.UpdatedAt = flattenNullableString(resp.UpdatedAt)

	// Config
	if len(resp.Config) > 0 {
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
func flattenTools(ctx context.Context, apiTools []agentStudio.ToolConfigInput, model *AgentResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	var searchTools []ToolAlgoliaSearchModel
	var recommendTools []ToolAlgoliaRecommendModel
	var clientTools []ToolClientSideModel
	var mcpTools []ToolMCPModel

	for i := range apiTools {
		tool := apiTools[i]
		switch {
		case tool.AlgoliaSearchToolConfig != nil:
			t, d := flattenAlgoliaSearchTool(tool.AlgoliaSearchToolConfig)
			diags.Append(d...)
			if t != nil {
				searchTools = append(searchTools, *t)
			}
		case tool.AlgoliaRecommendToolConfigInput != nil:
			t, d := flattenAlgoliaRecommendTool(tool.AlgoliaRecommendToolConfigInput)
			diags.Append(d...)
			if t != nil {
				recommendTools = append(recommendTools, *t)
			}
		case tool.ClientSideToolConfig != nil:
			t, d := flattenClientSideTool(tool.ClientSideToolConfig)
			diags.Append(d...)
			if t != nil {
				clientTools = append(clientTools, *t)
			}
		case tool.McpServerToolConfig != nil:
			t, d := flattenMCPTool(tool.McpServerToolConfig)
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

func flattenAlgoliaSearchTool(tool *agentStudio.AlgoliaSearchToolConfig) (*ToolAlgoliaSearchModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	model := &ToolAlgoliaSearchModel{
		Name: types.StringValue(tool.Name),
	}

	var indices []AlgoliaSearchIndexModel
	for i := range tool.Indices {
		src := tool.Indices[i]
		idx := AlgoliaSearchIndexModel{
			Name:        types.StringValue(src.Index),
			Description: types.StringValue(src.Description),
		}
		if src.EnhancedDescription != nil && *src.EnhancedDescription != "" {
			idx.EnhancedDescription = types.StringValue(*src.EnhancedDescription)
		} else {
			idx.EnhancedDescription = types.StringNull()
		}

		searchParams, d := flattenSearchParameters(src.SearchParameters)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		idx.SearchParameters = searchParams

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

// flattenSearchParameters marshals the typed search parameters back into the
// compact JSON string stored in Terraform state, stripping null values so the
// stored value matches what the user wrote.
func flattenSearchParameters(params utils.Nullable[agentStudio.SearchParameters]) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics
	if !params.IsSet() || params.Get() == nil {
		return types.StringNull(), diags
	}

	raw, err := json.Marshal(params.Get())
	if err != nil {
		diags.AddError("Error marshaling searchParameters", err.Error())
		return types.StringNull(), diags
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		diags.AddError("Error decoding searchParameters", err.Error())
		return types.StringNull(), diags
	}

	stripped := stripNullValues(decoded)
	if stripped == nil {
		return types.StringNull(), diags
	}

	b, err := json.Marshal(stripped)
	if err != nil {
		diags.AddError("Error marshaling searchParameters", err.Error())
		return types.StringNull(), diags
	}

	return types.StringValue(string(b)), diags
}

func flattenAlgoliaRecommendTool(tool *agentStudio.AlgoliaRecommendToolConfigInput) (*ToolAlgoliaRecommendModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	model := &ToolAlgoliaRecommendModel{
		Name: types.StringValue(tool.Name),
	}

	var configs []AlgoliaRecommendConfigModel
	for i := range tool.AllowedConfigs {
		src := tool.AllowedConfigs[i]
		cfg := AlgoliaRecommendConfigModel{
			Index:     types.StringValue(src.Index),
			ModelName: types.StringValue(src.ModelName),
		}
		if src.Description != nil && *src.Description != "" {
			cfg.Description = types.StringValue(*src.Description)
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

	if len(tool.PredefinedRecommendParameters) > 0 {
		b, err := json.Marshal(tool.PredefinedRecommendParameters)
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

func flattenClientSideTool(tool *agentStudio.ClientSideToolConfig) (*ToolClientSideModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	model := &ToolClientSideModel{
		Name:        types.StringValue(tool.Name),
		Description: types.StringValue(tool.Description),
	}

	b, err := json.Marshal(tool.InputSchema)
	if err != nil {
		diags.AddError("Error marshaling inputSchema", err.Error())
		return nil, diags
	}
	model.InputSchema = types.StringValue(string(b))

	return model, diags
}

func flattenMCPTool(tool *agentStudio.McpServerToolConfig) (*ToolMCPModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	model := &ToolMCPModel{
		Name: types.StringValue(tool.Name),
		URL:  types.StringValue(tool.Url),
	}

	if tool.Transport != nil {
		model.Transport = types.StringValue(*tool.Transport)
	} else {
		model.Transport = types.StringNull()
	}

	// Headers
	if len(tool.Headers) > 0 {
		headerMap, d := types.MapValueFrom(context.Background(), types.StringType, tool.Headers)
		diags.Append(d...)
		model.Headers = headerMap
	} else {
		model.Headers = types.MapNull(types.StringType)
	}

	// Allowed tools
	if len(tool.AllowedTools) > 0 {
		var allowedTools []MCPAllowedToolModel
		allowedToolNames := make([]string, 0, len(tool.AllowedTools))
		for name := range tool.AllowedTools {
			allowedToolNames = append(allowedToolNames, name)
		}
		sort.Strings(allowedToolNames)

		for _, name := range allowedToolNames {
			at := MCPAllowedToolModel{
				Name:             types.StringValue(name),
				RequiresApproval: types.BoolNull(),
				Alias:            types.StringNull(),
			}

			cfg := tool.AllowedTools[name]
			if cfg.McpToolConfig != nil {
				if v := cfg.McpToolConfig.RequiresApproval.Get(); v != nil {
					at.RequiresApproval = types.BoolValue(*v)
				}
				if v := cfg.McpToolConfig.Alias.Get(); v != nil && *v != "" {
					at.Alias = types.StringValue(*v)
				}
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

// flattenNullableString converts a utils.Nullable[string] to a types.String.
func flattenNullableString(s utils.Nullable[string]) types.String {
	if v := s.Get(); v != nil {
		return types.StringValue(*v)
	}
	return types.StringNull()
}

// stripNullValues recursively removes null (nil) values from a JSON-decoded
// structure. The Algolia API returns full search parameter schemas with every
// optional field set to null; stripping those keeps the stored JSON compact
// and consistent with what the user actually wrote.
func stripNullValues(v any) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, vv := range val {
			if vv == nil {
				continue
			}
			if stripped := stripNullValues(vv); stripped != nil {
				result[k] = stripped
			}
		}
		if len(result) == 0 {
			return nil
		}
		return result
	case []any:
		result := make([]any, 0, len(val))
		for _, vv := range val {
			if vv == nil {
				continue
			}
			if stripped := stripNullValues(vv); stripped != nil {
				result = append(result, stripped)
			}
		}
		if len(result) == 0 {
			return nil
		}
		return result
	default:
		return v
	}
}
