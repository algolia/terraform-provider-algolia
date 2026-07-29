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

// expandConfig parses the JSON-encoded config string into a map for the API. A
// configured empty object stays a non-nil empty map, which the client's own
// encoder sends as `{}` rather than omitting the field, so `config =
// jsonencode({})` is not silently turned into "no config at all".
func expandConfig(config types.String) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	if !isKnown(config) {
		return nil, diags
	}

	configObj, err := decodeJSONObject([]byte(config.ValueString()))
	if err != nil {
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
			tools = append(tools, *tool)
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
			tools = append(tools, *tool)
		}
	}

	// Algolia display results tools
	if isKnown(model.ToolAlgoliaDisplayResults) {
		var displayResultsTools []ToolAlgoliaDisplayResultsModel
		diags.Append(model.ToolAlgoliaDisplayResults.ElementsAs(ctx, &displayResultsTools, false)...)
		if diags.HasError() {
			return nil, diags
		}
		for i := range displayResultsTools {
			tools = append(tools, *expandAlgoliaDisplayResultsTool(&displayResultsTools[i]))
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
			tools = append(tools, *tool)
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
			tools = append(tools, *tool)
		}
	}

	return tools, diags
}

func expandAlgoliaSearchTool(ctx context.Context, model *ToolAlgoliaSearchModel) (*agentStudio.ToolConfigInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	var indices []AlgoliaSearchIndexModel
	diags.Append(model.Indices.ElementsAs(ctx, &indices, false)...)
	if diags.HasError() {
		return nil, diags
	}

	indexDocuments := make([]any, 0, len(indices))
	for _, idx := range indices {
		entry := agentStudio.AlgoliaSearchToolIndexConfig{
			Index:       idx.Name.ValueString(),
			Description: idx.Description.ValueString(),
		}
		if isKnown(idx.EnhancedDescription) {
			entry.EnhancedDescription = strPtr(idx.EnhancedDescription.ValueString())
		}

		document, err := jsonDocumentOf(entry)
		if err != nil {
			diags.AddError("Error encoding index configuration", err.Error())
			return nil, diags
		}

		// searchParameters is spliced in as the user's own bytes. Decoding it
		// into agentStudio.SearchParameters, which has no catch-all field, would
		// drop every search parameter the vendored client does not model yet
		// before the request is even sent.
		if isKnown(idx.SearchParameters) {
			params, err := jsonObjectBytes(idx.SearchParameters.ValueString())
			if err != nil {
				diags.AddError("Invalid search_parameters JSON", "Could not parse search_parameters: "+err.Error())
				return nil, diags
			}
			document["searchParameters"] = params
		}

		indexDocuments = append(indexDocuments, document)
	}

	document, err := jsonDocumentOf(agentStudio.AlgoliaSearchToolConfig{
		Name: model.Name.ValueString(),
		Type: "algolia_search_index",
	})
	if err != nil {
		diags.AddError("Error encoding algolia_search_index tool", err.Error())
		return nil, diags
	}
	document["indices"] = indexDocuments

	return rawToolConfig(document), diags
}

func expandAlgoliaRecommendTool(ctx context.Context, model *ToolAlgoliaRecommendModel) (*agentStudio.ToolConfigInput, diag.Diagnostics) {
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

	// PredefinedRecommendParameters is a map[string]any on the client, so no key
	// is lost, and a configured empty object stays a non-nil empty map that the
	// client's encoder sends as `{}` rather than omitting.
	if isKnown(model.PredefinedRecommendParameters) {
		params, err := decodeJSONObject([]byte(model.PredefinedRecommendParameters.ValueString()))
		if err != nil {
			diags.AddError("Invalid predefined_recommend_parameters JSON", "Could not parse predefined_recommend_parameters: "+err.Error())
			return nil, diags
		}
		tool.PredefinedRecommendParameters = params
	}

	return agentStudio.AlgoliaRecommendToolConfigInputAsToolConfigInput(tool), diags
}

func expandAlgoliaDisplayResultsTool(model *ToolAlgoliaDisplayResultsModel) *agentStudio.ToolConfigInput {
	tool := &agentStudio.AlgoliaDisplayResultsToolConfig{
		Type: "algolia_display_results",
	}

	if isKnown(model.Name) {
		tool.Name = strPtr(model.Name.ValueString())
	}
	tool.MinGroups = int32Ptr(model.MinGroups)
	tool.MaxGroups = int32Ptr(model.MaxGroups)
	tool.MinResultsPerGroup = int32Ptr(model.MinResultsPerGroup)
	tool.MaxResultsPerGroup = int32Ptr(model.MaxResultsPerGroup)

	return agentStudio.AlgoliaDisplayResultsToolConfigAsToolConfigInput(tool)
}

func expandClientSideTool(model *ToolClientSideModel) (*agentStudio.ToolConfigInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	// inputSchema is spliced in as the user's own bytes. Decoding it into
	// agentStudio.ClientToolsArgsSchema, which models only `type`, `properties`
	// and `required` and has no catch-all field, would drop every other JSON
	// Schema keyword - `additionalProperties`, `$schema`, `items`, `enum`, ... -
	// before the request is even sent.
	inputSchema, err := jsonObjectBytes(model.InputSchema.ValueString())
	if err != nil {
		diags.AddError("Invalid input_schema JSON", "Could not parse input_schema: "+err.Error())
		return nil, diags
	}

	document, err := jsonDocumentOf(agentStudio.ClientSideToolConfig{
		Name:        model.Name.ValueString(),
		Type:        "client_side",
		Description: model.Description.ValueString(),
	})
	if err != nil {
		diags.AddError("Error encoding client_side tool", err.Error())
		return nil, diags
	}
	document["inputSchema"] = inputSchema

	return rawToolConfig(document), diags
}

func expandMCPTool(ctx context.Context, model *ToolMCPModel) (*agentStudio.ToolConfigInput, diag.Diagnostics) {
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

	return agentStudio.McpServerToolConfigAsToolConfigInput(tool), diags
}

// rawToolConfig wraps an encoded tool document in a ToolConfigInput that
// marshals back to exactly those bytes.
//
// The tools carrying a user-supplied JSON document cannot go out through their
// own typed variant: agentStudio.ClientToolsArgsSchema and
// agentStudio.SearchParameters have no catch-all field, so encoding through them
// drops every key the vendored client does not model yet. UnknownToolConfig is
// the client's own forward-compatibility carrier - it serialises `name`, `type`
// and every AdditionalProperties entry at the top level - so a document routed
// through it keeps every key, in the order it was written, with its numeric
// literals intact. Only insignificant whitespace changes, because json.Marshal
// compacts what a json.Marshaler returns; the response is matched against the
// configured document semantically, so that costs nothing (see
// flattenJSONDocument). Reads are unaffected: ToolConfigInput.UnmarshalJSON
// routes on `type`, so a response never decodes into an UnknownToolConfig for a
// type the provider supports.
func rawToolConfig(document map[string]any) *agentStudio.ToolConfigInput {
	name, _ := document["name"].(string)
	toolType, _ := document["type"].(string)

	properties := make(map[string]any, len(document))
	for key, value := range document {
		if key != "name" && key != "type" {
			properties[key] = value
		}
	}

	return agentStudio.UnknownToolConfigAsToolConfigInput(agentStudio.UnknownToolConfig{
		Name:                 name,
		Type:                 toolType,
		AdditionalProperties: properties,
	})
}

// jsonDocumentOf encodes a typed tool config into a generic JSON object so a
// user-supplied document can be spliced into it before the request is sent. The
// client's own struct tags stay authoritative for every modelled field.
func jsonDocumentOf(value any) (map[string]any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	return decodeJSONObject(encoded)
}

// jsonObjectBytes checks that a configured attribute holds a JSON object and
// returns its bytes unchanged, so the document reaches the API exactly as
// written.
func jsonObjectBytes(document string) (json.RawMessage, error) {
	if _, err := decodeJSONObject([]byte(document)); err != nil {
		return nil, err
	}

	return json.RawMessage(document), nil
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

// int32Ptr converts a Terraform int64 into the *int32 the API expects,
// mapping null and unknown to nil so the field is omitted from the payload.
func int32Ptr(v types.Int64) *int32 {
	if !isKnown(v) {
		return nil
	}

	value := int32(v.ValueInt64())

	return &value
}
