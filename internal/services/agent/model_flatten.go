package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

var algoliaDisplayResultsToolAttrTypes = map[string]attr.Type{
	"name":                  types.StringType,
	"min_groups":            types.Int64Type,
	"max_groups":            types.Int64Type,
	"min_results_per_group": types.Int64Type,
	"max_results_per_group": types.Int64Type,
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
func flattenAgentResponse(ctx context.Context, doc *agentDocument, model *AgentResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	resp := doc.agent

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

	// Agent Studio merges its own defaults into config - it answers a configured
	// {"temperature":0.5} with {"temperature":0.5,"enableAlgoliaMcp":true} - so
	// the document it returns is never the one that was written.
	model.Config = flattenConfiguredJSONDocument(doc.raw.config, model.Config)

	// Tools
	d := flattenTools(ctx, resp.Tools, doc.raw.tools, model)
	diags.Append(d...)

	return diags
}

// flattenJSONDocument renders one of the API's raw JSON documents into the
// string attribute that stores it.
//
// Those attributes are Required or Optional, so their planned value is the
// configuration verbatim: a document that carries the same data but differs in
// key order or whitespace would make Terraform reject the apply as an
// inconsistent result. Whenever the configured document and the API's carry the
// same data, the configured string is therefore kept verbatim. Note that an
// empty object is a document like any other: collapsing it to null would abort
// the apply of an explicitly configured `jsonencode({})`.
func flattenJSONDocument(document json.RawMessage, prior types.String) types.String {
	if isJSONNull(document) {
		return types.StringNull()
	}

	if !prior.IsNull() && !prior.IsUnknown() && jsonEqual([]byte(prior.ValueString()), document) {
		return prior
	}

	return types.StringValue(string(document))
}

// flattenConfiguredJSONDocument is flattenJSONDocument for the documents Agent
// Studio does not hand back the way they were written, whatever the provider
// sends: it strips the JSON Schema keywords it does not model out of
// `input_schema`, it strips unmodelled search parameters out of
// `search_parameters` and expands the rest into the full parameter schema with
// every unset field explicitly null, and it merges its own defaults into
// `config`. The documents reach Algolia intact - that is verifiable in the
// outbound request - but the response is a different document, so there is no
// value the provider could store that is both the plan and the truth.
//
// The configured value therefore wins whenever there is one, and the API's
// document is used only when there is none: an import or a data source read.
// This is the trade `preservePlannedValues` already makes for
// `ranking.relevancy_strictness` on algolia_index, and it costs the same thing -
// out-of-band drift on these attributes is not reported. That is not a
// regression, because a drift check needs the API to say what it stored, and here
// it will not. The alternative is worse: storing the response would make every
// apply of a configured value fail with "Provider produced inconsistent result
// after apply".
//
// Do not replace this with a looser comparison in jsonEqual, and do not make
// these attributes Computed to sidestep the contract - Computed frees the
// provider only while the configuration is null, so it would not help a
// configured value and would mask the next real defect here.
func flattenConfiguredJSONDocument(document json.RawMessage, prior types.String) types.String {
	if !prior.IsNull() && !prior.IsUnknown() {
		return prior
	}

	if isJSONNull(document) {
		return types.StringNull()
	}

	return types.StringValue(string(document))
}

// flattenTools distributes API tools into the typed tool list attributes on the model.
//
// Every variant of agentStudio.ToolConfigInput must be accounted for. Agent
// writes are full replacements of the tool list (see expandTools), so a
// variant that is read but not stored is silently deleted from the live agent
// by the next apply. The switch therefore ends in a default arm that raises an
// error naming the unhandled variant: a client upgrade that adds a tool type
// has to fail loudly instead of quietly dropping the user's configuration.
//
// raw carries the same tools as apiTools, positionally, holding the JSON
// documents that must not be rebuilt from the typed models - see agentDocument.
//
// The prior values of the model's tool lists (prior state on Read, configuration
// on Create/Update) are captured before the model is overwritten, so each tool's
// header map and JSON documents can be compared against what the same tool held
// before the refresh - see flattenMCPHeaders and flattenJSONDocument.
func flattenTools(ctx context.Context, apiTools []agentStudio.ToolConfigInput, raw []rawTool, model *AgentResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	priorHeaders := priorMCPHeaders(model.ToolMCP)
	priorInputSchemas := priorToolDocuments(model.ToolClientSide, "input_schema")
	priorRecommendParameters := priorToolDocuments(model.ToolAlgoliaRecommend, "predefined_recommend_parameters")
	priorSearchParameters := priorIndexSearchParameters(model.ToolAlgoliaSearch)

	var searchTools []ToolAlgoliaSearchModel
	var recommendTools []ToolAlgoliaRecommendModel
	var displayResultsTools []ToolAlgoliaDisplayResultsModel
	var clientTools []ToolClientSideModel
	var mcpTools []ToolMCPModel

	for i := range apiTools {
		tool := apiTools[i]
		switch {
		case tool.AlgoliaSearchToolConfig != nil:
			name := tool.AlgoliaSearchToolConfig.Name
			t, d := flattenAlgoliaSearchTool(tool.AlgoliaSearchToolConfig, raw[i], priorSearchParameters[name])
			diags.Append(d...)
			if t != nil {
				searchTools = append(searchTools, *t)
			}
		case tool.AlgoliaRecommendToolConfigInput != nil:
			name := tool.AlgoliaRecommendToolConfigInput.Name
			t, d := flattenAlgoliaRecommendTool(tool.AlgoliaRecommendToolConfigInput, raw[i], priorRecommendParameters[name])
			diags.Append(d...)
			if t != nil {
				recommendTools = append(recommendTools, *t)
			}
		case tool.ClientSideToolConfig != nil:
			name := tool.ClientSideToolConfig.Name
			clientTools = append(clientTools,
				flattenClientSideTool(tool.ClientSideToolConfig, raw[i], priorInputSchemas[name]))
		case tool.McpServerToolConfig != nil:
			name := tool.McpServerToolConfig.Name
			t, d := flattenMCPTool(tool.McpServerToolConfig, priorHeaders[name])
			diags.Append(d...)
			if t != nil {
				mcpTools = append(mcpTools, *t)
			}
		case tool.AlgoliaDisplayResultsToolConfig != nil:
			// Checked after the variants above: every optional field of
			// AlgoliaDisplayResultsToolConfig is a pointer, so the client's
			// greedy oneOf fallback can populate it for a payload that also
			// matched a more specific variant.
			displayResultsTools = append(displayResultsTools, flattenAlgoliaDisplayResultsTool(tool.AlgoliaDisplayResultsToolConfig))
		case tool.UnknownToolConfig != nil:
			// The client's placeholder for a tool whose `type` it does not
			// recognise. There is no Terraform schema that could hold its
			// arbitrary properties, so refuse to manage the agent instead of
			// dropping the tool on the next write.
			diags.AddError(
				"Unsupported agent tool type",
				"Agent tool "+tool.UnknownToolConfig.Name+" has type "+tool.UnknownToolConfig.Type+
					", which this provider version cannot represent. Terraform would delete the tool on the "+
					"next apply, because updating an agent replaces its whole tool list. Remove the tool from "+
					"the agent, or upgrade the provider to a version that supports this tool type.",
			)
		default:
			diags.AddError(
				"Unsupported agent tool variant",
				fmt.Sprintf("The Algolia client returned an agent tool variant this provider version does not "+
					"handle (%T). Terraform would delete the tool on the next apply, because updating an agent "+
					"replaces its whole tool list. This is a provider bug: please report it.", tool.GetActualInstance()),
			)
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

	displayResultsToolType := types.ObjectType{AttrTypes: algoliaDisplayResultsToolAttrTypes}
	if len(displayResultsTools) > 0 {
		list, d := types.ListValueFrom(ctx, displayResultsToolType, displayResultsTools)
		diags.Append(d...)
		model.ToolAlgoliaDisplayResults = list
	} else {
		model.ToolAlgoliaDisplayResults = types.ListNull(displayResultsToolType)
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

// flattenAlgoliaSearchTool maps an algolia_search_index tool. priorParameters
// holds the search_parameters each of its indices already carried in the model,
// keyed by index name.
func flattenAlgoliaSearchTool(
	tool *agentStudio.AlgoliaSearchToolConfig,
	raw rawTool,
	priorParameters map[string]types.String,
) (*ToolAlgoliaSearchModel, diag.Diagnostics) {
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

		searchParams, d := flattenSearchParameters(raw.searchParameters[src.Index], priorParameters[src.Index])
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

// flattenSearchParameters renders an index's raw searchParameters document. The
// configured value wins whenever there is one, because Algolia does not return
// this document faithfully - see flattenConfiguredJSONDocument.
//
// Without a configured value, for an import or a data source read, the API's
// document is stored with its null values stripped: Algolia answers with the full
// search parameter schema, every unset parameter explicitly null, which would
// otherwise bury the handful that are actually set under some sixty nulls.
func flattenSearchParameters(document json.RawMessage, prior types.String) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics
	if !prior.IsNull() && !prior.IsUnknown() {
		return prior, diags
	}

	if isJSONNull(document) {
		return types.StringNull(), diags
	}

	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.UseNumber()

	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
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

func flattenAlgoliaRecommendTool(
	tool *agentStudio.AlgoliaRecommendToolConfigInput,
	raw rawTool,
	priorParameters types.String,
) (*ToolAlgoliaRecommendModel, diag.Diagnostics) {
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

	// Unlike input_schema, search_parameters and config, this document does come
	// back exactly as it was written - Algolia stores it opaquely, unmodelled keys
	// included - so it keeps the stricter rule and with it real drift detection.
	// Verified against the live API; do not "align" it with the others.
	model.PredefinedRecommendParameters = flattenJSONDocument(raw.predefinedRecommendParameters, priorParameters)

	return model, diags
}

// flattenAlgoliaDisplayResultsTool maps an algolia_display_results tool. All
// of its fields are optional in the API, and all of them are Computed in the
// schema, so an absent field becomes null and is filled from the API response.
func flattenAlgoliaDisplayResultsTool(tool *agentStudio.AlgoliaDisplayResultsToolConfig) ToolAlgoliaDisplayResultsModel {
	return ToolAlgoliaDisplayResultsModel{
		Name:               types.StringPointerValue(tool.Name),
		MinGroups:          flattenNullableInt32(tool.MinGroups),
		MaxGroups:          flattenNullableInt32(tool.MaxGroups),
		MinResultsPerGroup: flattenNullableInt32(tool.MinResultsPerGroup),
		MaxResultsPerGroup: flattenNullableInt32(tool.MaxResultsPerGroup),
	}
}

func flattenClientSideTool(tool *agentStudio.ClientSideToolConfig, raw rawTool, priorSchema types.String) ToolClientSideModel {
	return ToolClientSideModel{
		Name:        types.StringValue(tool.Name),
		Description: types.StringValue(tool.Description),
		// Algolia strips the JSON Schema keywords it does not model - `$schema`
		// and `additionalProperties` among them - so the configured document
		// wins. See flattenConfiguredJSONDocument.
		InputSchema: flattenConfiguredJSONDocument(raw.inputSchema, priorSchema),
	}
}

func flattenMCPTool(tool *agentStudio.McpServerToolConfig, priorHeaders types.Map) (*ToolMCPModel, diag.Diagnostics) {
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

	headers, d := flattenMCPHeaders(tool.Headers, priorHeaders)
	diags.Append(d...)
	model.Headers = headers

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

// flattenMCPHeaders converts the API's header map into a Terraform map.
// `headers` is Optional and not Computed, so its planned value is the
// configuration verbatim: emitting a null map where the plan held a known
// empty map (`headers = {}`) makes Terraform reject the apply with "Provider
// produced inconsistent result after apply". When the API returns no headers
// the prior value therefore decides: a null prior stays null, while a prior
// that was explicitly configured as `{}` stays a known empty map. A prior with
// entries that the API no longer returns is real drift and becomes null.
func flattenMCPHeaders(headers map[string]string, prior types.Map) (types.Map, diag.Diagnostics) {
	if len(headers) == 0 {
		if !prior.IsNull() && !prior.IsUnknown() && len(prior.Elements()) == 0 {
			return prior, nil // explicit {}
		}

		return types.MapNull(types.StringType), nil
	}

	return types.MapValueFrom(context.Background(), types.StringType, headers)
}

// priorMCPHeaders indexes the `headers` map of every MCP tool already present
// in the model by tool name, so flattenMCPHeaders can compare the API response
// against the value the same tool held before the refresh. Returns nil for a
// null/unknown list, i.e. for imports and data source reads, where every empty
// header map maps to null.
func priorMCPHeaders(prior types.List) map[string]types.Map {
	if prior.IsNull() || prior.IsUnknown() {
		return nil
	}

	headers := make(map[string]types.Map, len(prior.Elements()))
	forEachNamedBlock(prior, func(name string, attributes map[string]attr.Value) {
		if value, ok := attributes["headers"].(types.Map); ok {
			headers[name] = value
		}
	})

	return headers
}

// priorToolDocuments indexes a JSON-document attribute of every tool already
// present in the model by tool name, so flattenJSONDocument can compare the API
// response against the value the same tool held before the refresh. A tool
// absent from the map yields the zero types.String, which is null, i.e. "no
// configured value to preserve".
func priorToolDocuments(prior types.List, attribute string) map[string]types.String {
	documents := make(map[string]types.String, len(prior.Elements()))
	forEachNamedBlock(prior, func(name string, attributes map[string]attr.Value) {
		if value, ok := attributes[attribute].(types.String); ok {
			documents[name] = value
		}
	})

	return documents
}

// priorIndexSearchParameters indexes the `search_parameters` of every index of
// every algolia_search_index tool already present in the model, by tool name and
// then index name.
func priorIndexSearchParameters(prior types.List) map[string]map[string]types.String {
	parameters := make(map[string]map[string]types.String, len(prior.Elements()))
	forEachNamedBlock(prior, func(name string, attributes map[string]attr.Value) {
		indices, ok := attributes["index"].(types.List)
		if !ok {
			return
		}
		parameters[name] = priorToolDocuments(indices, "search_parameters")
	})

	return parameters
}

// forEachNamedBlock calls fn with the name and attributes of every element of a
// list of named blocks, skipping any element without a usable name. Both tool
// names and index names are unique within their parent (a tool name addresses
// the tool in the LLM prompt), so they can key a lookup keeping the model's prior
// values reachable while it is being overwritten.
func forEachNamedBlock(list types.List, fn func(name string, attributes map[string]attr.Value)) {
	if list.IsNull() || list.IsUnknown() {
		return
	}

	for _, element := range list.Elements() {
		obj, ok := element.(types.Object)
		if !ok {
			continue
		}

		name, ok := obj.Attributes()["name"].(types.String)
		if !ok || name.IsNull() || name.IsUnknown() {
			continue
		}

		fn(name.ValueString(), obj.Attributes())
	}
}

// flattenNullableInt32 converts an optional API int32 to a types.Int64.
func flattenNullableInt32(value *int32) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}

	return types.Int64Value(int64(*value))
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
