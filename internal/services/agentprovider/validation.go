package agentprovider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func validateAgentProviderConfig(ctx context.Context, model AgentProviderResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	activeSpecs := configuredProviderSpecs(model)
	if len(activeSpecs) == 0 {
		diags.AddAttributeError(
			path.Root("provider_name"),
			"Missing Provider Configuration Block",
			"Exactly one provider configuration block must be set and must match provider_name.",
		)
		return diags
	}

	if len(activeSpecs) > 1 {
		diags.AddAttributeError(
			path.Root("provider_name"),
			"Multiple Provider Configuration Blocks",
			"Exactly one provider configuration block must be set.",
		)
		return diags
	}

	activeSpec := activeSpecs[0]
	if !model.ProviderName.IsNull() && !model.ProviderName.IsUnknown() && model.ProviderName.ValueString() != activeSpec.ProviderName {
		diags.AddAttributeError(
			path.Root("provider_name"),
			"Provider Block Does Not Match provider_name",
			fmt.Sprintf("The %q block is configured, but provider_name is %q.", activeSpec.BlockName, model.ProviderName.ValueString()),
		)
		return diags
	}

	block := providerBlockValue(model, activeSpec)
	if block.IsNull() || block.IsUnknown() {
		return diags
	}

	attrs := block.Attributes()
	for _, field := range activeSpec.Fields {
		value, ok := attrs[field.TerraformName]
		if !ok {
			continue
		}

		stringValue, ok := value.(types.String)
		if !ok {
			continue
		}

		if stringValue.IsUnknown() {
			continue
		}

		if field.Required && stringValue.IsNull() {
			diags.AddAttributeError(
				path.Root(activeSpec.BlockName).AtName(field.TerraformName),
				"Missing Provider Field",
				fmt.Sprintf("The %q field is required when provider_name is %q.", field.TerraformName, activeSpec.ProviderName),
			)
		}
	}

	return diags
}

func configuredProviderSpecs(model AgentProviderResourceModel) []providerSpec {
	specs := make([]providerSpec, 0, len(providerSpecs))
	for _, spec := range providerSpecs {
		block := providerBlockValue(model, spec)
		if block.IsNull() || block.IsUnknown() {
			continue
		}

		specs = append(specs, spec)
	}

	return specs
}

func providerBlockValue(model AgentProviderResourceModel, spec providerSpec) types.Object {
	switch spec.BlockName {
	case "openai":
		return model.OpenAI
	case "azure_openai":
		return model.AzureOpenAI
	case "google_genai":
		return model.GoogleGenAI
	case "anthropic":
		return model.Anthropic
	case "openai_compatible":
		return model.OpenAICompatible
	default:
		return providerBlockNull(spec)
	}
}

func setProviderBlockValue(model *AgentProviderResourceModel, spec providerSpec, value types.Object) {
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
