package agentprovider

import (
	"context"
	"encoding/json"

	agentStudio "github.com/algolia/algoliasearch-client-go/v4/algolia/agent-studio"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/utils"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func hydrateAgentProviderResourceState(_ context.Context, resp *agentStudio.ProviderAuthenticationResponse, preserved AgentProviderResourceModel, model *AgentProviderResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	input, inputDiags := providerInputToMap(resp.Input)
	diags.Append(inputDiags...)
	if diags.HasError() {
		return diags
	}

	model.ID = types.StringValue(resp.Id)
	model.Name = types.StringValue(resp.Name)
	model.ProviderName = types.StringValue(resp.ProviderName)
	model.CreatedAt = types.StringValue(resp.CreatedAt)
	model.UpdatedAt = types.StringValue(resp.UpdatedAt)
	model.LastUsedAt = nullableString(resp.LastUsedAt.Get())

	for _, spec := range providerSpecs {
		setProviderBlockValue(model, spec, providerBlockNull(spec))
	}

	spec, ok := providerSpecByName(resp.ProviderName)
	if !ok {
		diags.AddError("Unsupported Provider Type", "Received unknown provider type "+resp.ProviderName+" from the Agent Studio API.")
		return diags
	}

	preservedBlock := providerBlockValue(preserved, spec)
	blockValue, blockDiags := providerBlockValueFromResponse(input, spec, preservedBlock)
	diags.Append(blockDiags...)
	if diags.HasError() {
		return diags
	}

	setProviderBlockValue(model, spec, blockValue)

	return diags
}

func hydrateImportedAgentProviderResourceState(_ context.Context, resp *agentStudio.ProviderAuthenticationResponse, model *AgentProviderResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	input, inputDiags := providerInputToMap(resp.Input)
	diags.Append(inputDiags...)
	if diags.HasError() {
		return diags
	}

	model.ID = types.StringValue(resp.Id)
	model.Name = types.StringValue(resp.Name)
	model.ProviderName = types.StringValue(resp.ProviderName)
	model.CreatedAt = types.StringValue(resp.CreatedAt)
	model.UpdatedAt = types.StringValue(resp.UpdatedAt)
	model.LastUsedAt = nullableString(resp.LastUsedAt.Get())

	for _, spec := range providerSpecs {
		setProviderBlockValue(model, spec, providerBlockNull(spec))
	}

	spec, ok := providerSpecByName(resp.ProviderName)
	if !ok {
		diags.AddError("Unsupported Provider Type", "Received unknown provider type "+resp.ProviderName+" from the Agent Studio API.")
		return diags
	}

	blockValue, blockDiags := providerBlockValueFromResponse(input, spec, providerBlockNull(spec))
	diags.Append(blockDiags...)
	if diags.HasError() {
		return diags
	}

	setProviderBlockValue(model, spec, blockValue)
	return diags
}

func hydrateAgentProviderDataSourceState(_ context.Context, resp *agentStudio.ProviderAuthenticationResponse, model *AgentProviderDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	input, inputDiags := providerInputToMap(resp.Input)
	diags.Append(inputDiags...)
	if diags.HasError() {
		return diags
	}

	model.ProviderID = types.StringValue(resp.Id)
	model.ID = types.StringValue(resp.Id)
	model.Name = types.StringValue(resp.Name)
	model.ProviderName = types.StringValue(resp.ProviderName)
	model.CreatedAt = types.StringValue(resp.CreatedAt)
	model.UpdatedAt = types.StringValue(resp.UpdatedAt)
	model.LastUsedAt = nullableString(resp.LastUsedAt.Get())

	for _, spec := range providerSpecs {
		setDataSourceProviderBlockValue(model, spec, providerDataSourceBlockNull(spec))
	}

	spec, ok := providerSpecByName(resp.ProviderName)
	if !ok {
		diags.AddError("Unsupported Provider Type", "Received unknown provider type "+resp.ProviderName+" from the Agent Studio API.")
		return diags
	}

	blockValue, blockDiags := providerDataSourceBlockValueFromResponse(input, spec)
	diags.Append(blockDiags...)
	if diags.HasError() {
		return diags
	}

	setDataSourceProviderBlockValue(model, spec, blockValue)
	return diags
}

func providerRequestFromModel(model AgentProviderResourceModel) (*agentStudio.ProviderAuthenticationCreate, diag.Diagnostics) {
	var diags diag.Diagnostics

	spec, ok := providerSpecByName(model.ProviderName.ValueString())
	if !ok {
		diags.AddError("Unsupported Provider Type", "Unknown provider_name "+model.ProviderName.ValueString())
		return nil, diags
	}

	inputMap := providerInputMap(spec, providerBlockValue(model, spec))
	input, inputDiags := buildProviderInput(spec.ProviderName, inputMap)
	diags.Append(inputDiags...)
	if diags.HasError() {
		return nil, diags
	}

	return &agentStudio.ProviderAuthenticationCreate{
		Name:         model.Name.ValueString(),
		ProviderName: agentStudio.ProviderName(model.ProviderName.ValueString()),
		Input:        input,
	}, diags
}

func providerRequestFromModelForUpdate(model AgentProviderResourceModel) (*agentStudio.ProviderAuthenticationPatch, diag.Diagnostics) {
	var diags diag.Diagnostics

	spec, ok := providerSpecByName(model.ProviderName.ValueString())
	if !ok {
		diags.AddError("Unsupported Provider Type", "Unknown provider_name "+model.ProviderName.ValueString())
		return nil, diags
	}

	inputMap := providerInputMap(spec, providerBlockValue(model, spec))
	input, inputDiags := buildProviderInputNullable(spec.ProviderName, inputMap)
	diags.Append(inputDiags...)
	if diags.HasError() {
		return nil, diags
	}

	patch := &agentStudio.ProviderAuthenticationPatch{
		Name:  *utils.NewNullable(strPtr(model.Name.ValueString())),
		Input: *utils.NewNullable(&input),
	}

	return patch, diags
}

func providerBlockValueFromResponse(input map[string]any, spec providerSpec, preservedBlock types.Object) (types.Object, diag.Diagnostics) {
	attrValues := make(map[string]attr.Value, len(spec.Fields))
	preservedAttrs := map[string]attr.Value{}
	if !preservedBlock.IsNull() && !preservedBlock.IsUnknown() {
		preservedAttrs = preservedBlock.Attributes()
	}

	for _, field := range spec.Fields {
		switch {
		case field.Sensitive:
			if preservedValue, ok := preservedAttrs[field.TerraformName]; ok {
				attrValues[field.TerraformName] = preservedValue
			} else {
				attrValues[field.TerraformName] = types.StringNull()
			}
		default:
			attrValues[field.TerraformName] = stringFromAny(input[field.APIName])
		}
	}

	return types.ObjectValue(providerBlockAttrTypes(spec), attrValues)
}

func providerDataSourceBlockValueFromResponse(input map[string]any, spec providerSpec) (types.Object, diag.Diagnostics) {
	attrTypes := providerDataSourceBlockAttrTypes(spec)
	attrValues := make(map[string]attr.Value, len(attrTypes))

	for _, field := range nonSensitiveProviderFields(spec) {
		attrValues[field.TerraformName] = stringFromAny(input[field.APIName])
	}

	return types.ObjectValue(attrTypes, attrValues)
}

// providerInputMap collects the configured (known) provider fields into a map
// keyed by the provider's API field names.
func providerInputMap(spec providerSpec, block types.Object) map[string]any {
	if block.IsNull() || block.IsUnknown() {
		return map[string]any{}
	}

	input := make(map[string]any, len(spec.Fields))
	for _, field := range spec.Fields {
		attrValue, ok := block.Attributes()[field.TerraformName]
		if !ok {
			continue
		}

		stringValue, ok := attrValue.(types.String)
		if !ok || stringValue.IsNull() || stringValue.IsUnknown() {
			continue
		}

		input[field.APIName] = stringValue.ValueString()
	}

	return input
}

// buildProviderInput converts the API-keyed input map into the typed ProviderInput
// union expected by the SDK, based on the provider name.
func buildProviderInput(providerName string, input map[string]any) (agentStudio.ProviderInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	value, decodeDiags := decodeProviderInputStruct(providerName, input)
	diags.Append(decodeDiags...)
	if diags.HasError() {
		return agentStudio.ProviderInput{}, diags
	}

	switch v := value.(type) {
	case *agentStudio.OpenAIProviderInput:
		return *agentStudio.OpenAIProviderInputAsProviderInput(v), diags
	case *agentStudio.AnthropicProviderInput:
		return *agentStudio.AnthropicProviderInputAsProviderInput(v), diags
	case *agentStudio.AzureOpenAIProviderInput:
		return *agentStudio.AzureOpenAIProviderInputAsProviderInput(v), diags
	case *agentStudio.OpenAICompatibleProviderInput:
		return *agentStudio.OpenAICompatibleProviderInputAsProviderInput(v), diags
	default:
		return *agentStudio.BaseProviderInputAsProviderInput(value.(*agentStudio.BaseProviderInput)), diags
	}
}

// buildProviderInputNullable mirrors buildProviderInput for the PATCH request body.
func buildProviderInputNullable(providerName string, input map[string]any) (agentStudio.ProviderInputNullable, diag.Diagnostics) {
	var diags diag.Diagnostics

	value, decodeDiags := decodeProviderInputStruct(providerName, input)
	diags.Append(decodeDiags...)
	if diags.HasError() {
		return agentStudio.ProviderInputNullable{}, diags
	}

	switch v := value.(type) {
	case *agentStudio.OpenAIProviderInput:
		return *agentStudio.OpenAIProviderInputAsProviderInputNullable(v), diags
	case *agentStudio.AnthropicProviderInput:
		return *agentStudio.AnthropicProviderInputAsProviderInputNullable(v), diags
	case *agentStudio.AzureOpenAIProviderInput:
		return *agentStudio.AzureOpenAIProviderInputAsProviderInputNullable(v), diags
	case *agentStudio.OpenAICompatibleProviderInput:
		return *agentStudio.OpenAICompatibleProviderInputAsProviderInputNullable(v), diags
	default:
		return *agentStudio.BaseProviderInputAsProviderInputNullable(value.(*agentStudio.BaseProviderInput)), diags
	}
}

// decodeProviderInputStruct unmarshals the API-keyed input map into the concrete
// provider input struct that matches the provider name.
func decodeProviderInputStruct(providerName string, input map[string]any) (any, diag.Diagnostics) {
	var diags diag.Diagnostics

	raw, err := json.Marshal(input)
	if err != nil {
		diags.AddError("Error building provider input", err.Error())
		return nil, diags
	}

	var target any
	switch providerName {
	case "openai":
		target = &agentStudio.OpenAIProviderInput{}
	case "anthropic":
		target = &agentStudio.AnthropicProviderInput{}
	case "azure_openai":
		target = &agentStudio.AzureOpenAIProviderInput{}
	case "openai_compatible":
		target = &agentStudio.OpenAICompatibleProviderInput{}
	default:
		// google_genai, deepseek and any other single-key provider use the base input.
		target = &agentStudio.BaseProviderInput{}
	}

	if err := json.Unmarshal(raw, target); err != nil {
		diags.AddError("Error building provider input", err.Error())
		return nil, diags
	}

	return target, diags
}

// providerInputToMap converts the typed ProviderInput union from the API response
// into a generic map keyed by the provider's API field names.
func providerInputToMap(input agentStudio.ProviderInput) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics

	raw, err := json.Marshal(input)
	if err != nil {
		diags.AddError("Error reading provider input", err.Error())
		return nil, diags
	}

	if len(raw) == 0 || string(raw) == "null" {
		return map[string]any{}, diags
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		diags.AddError("Error reading provider input", err.Error())
		return nil, diags
	}

	return result, diags
}

func nullableString(value *string) types.String {
	if value == nil || *value == "" {
		return types.StringNull()
	}

	return types.StringValue(*value)
}

func stringFromAny(value any) types.String {
	stringValue, ok := value.(string)
	if !ok || stringValue == "" {
		return types.StringNull()
	}

	return types.StringValue(stringValue)
}

func strPtr(s string) *string {
	return &s
}

func setDataSourceProviderBlockValue(model *AgentProviderDataSourceModel, spec providerSpec, value types.Object) {
	switch spec.BlockName {
	case "openai":
		model.OpenAI = value
	case "azure_openai":
		model.AzureOpenAI = value
	case "google_genai":
		model.GoogleGenAI = value
	case "anthropic":
		model.Anthropic = value
	case "openai_compatible":
		model.OpenAICompatible = value
	}
}
