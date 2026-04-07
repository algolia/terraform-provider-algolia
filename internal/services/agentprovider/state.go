package agentprovider

import (
	"context"

	agentstudio "github.com/algolia/terraform-provider-algolia/internal/services/agent"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func hydrateAgentProviderResourceState(_ context.Context, resp *agentstudio.ProviderResponse, preserved AgentProviderResourceModel, model *AgentProviderResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(resp.ID)
	model.Name = types.StringValue(resp.Name)
	model.ProviderName = types.StringValue(resp.ProviderName)
	model.CreatedAt = types.StringValue(resp.CreatedAt)
	model.UpdatedAt = types.StringValue(resp.UpdatedAt)
	model.LastUsedAt = nullableString(resp.LastUsedAt)

	for _, spec := range providerSpecs {
		setProviderBlockValue(model, spec, providerBlockNull(spec))
	}

	spec, ok := providerSpecByName(resp.ProviderName)
	if !ok {
		diags.AddError("Unsupported Provider Type", "Received unknown provider type "+resp.ProviderName+" from the Agent Studio API.")
		return diags
	}

	preservedBlock := providerBlockValue(preserved, spec)
	blockValue, blockDiags := providerBlockValueFromResponse(resp.Input, spec, preservedBlock)
	diags.Append(blockDiags...)
	if diags.HasError() {
		return diags
	}

	setProviderBlockValue(model, spec, blockValue)

	return diags
}

func hydrateImportedAgentProviderResourceState(_ context.Context, resp *agentstudio.ProviderResponse, model *AgentProviderResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(resp.ID)
	model.Name = types.StringValue(resp.Name)
	model.ProviderName = types.StringValue(resp.ProviderName)
	model.CreatedAt = types.StringValue(resp.CreatedAt)
	model.UpdatedAt = types.StringValue(resp.UpdatedAt)
	model.LastUsedAt = nullableString(resp.LastUsedAt)

	for _, spec := range providerSpecs {
		setProviderBlockValue(model, spec, providerBlockNull(spec))
	}

	spec, ok := providerSpecByName(resp.ProviderName)
	if !ok {
		diags.AddError("Unsupported Provider Type", "Received unknown provider type "+resp.ProviderName+" from the Agent Studio API.")
		return diags
	}

	blockValue, blockDiags := providerBlockValueFromResponse(resp.Input, spec, providerBlockNull(spec))
	diags.Append(blockDiags...)
	if diags.HasError() {
		return diags
	}

	setProviderBlockValue(model, spec, blockValue)
	return diags
}

func providerRequestFromModel(model AgentProviderResourceModel) (*agentstudio.ProviderRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	spec, ok := providerSpecByName(model.ProviderName.ValueString())
	if !ok {
		diags.AddError("Unsupported Provider Type", "Unknown provider_name "+model.ProviderName.ValueString())
		return nil, diags
	}

	input, inputDiags := expandProviderInput(spec, providerBlockValue(model, spec))
	diags.Append(inputDiags...)
	if diags.HasError() {
		return nil, diags
	}

	name := model.Name.ValueString()
	providerName := model.ProviderName.ValueString()

	req := &agentstudio.ProviderRequest{
		Name:         &name,
		ProviderName: &providerName,
		Input:        input,
	}

	return req, diags
}

func providerRequestFromModelForUpdate(model AgentProviderResourceModel) (*agentstudio.ProviderRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	spec, ok := providerSpecByName(model.ProviderName.ValueString())
	if !ok {
		diags.AddError("Unsupported Provider Type", "Unknown provider_name "+model.ProviderName.ValueString())
		return nil, diags
	}

	input, inputDiags := expandProviderInput(spec, providerBlockValue(model, spec))
	diags.Append(inputDiags...)
	if diags.HasError() {
		return nil, diags
	}

	name := model.Name.ValueString()
	return &agentstudio.ProviderRequest{
		Name:  &name,
		Input: input,
	}, diags
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

func expandProviderInput(spec providerSpec, block types.Object) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics

	if block.IsNull() || block.IsUnknown() {
		return map[string]any{}, diags
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

	return input, diags
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
