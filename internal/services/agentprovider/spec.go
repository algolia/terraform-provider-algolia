package agentprovider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type providerFieldSpec struct {
	TerraformName string
	APIName       string
	Description   string
	Required      bool
	Sensitive     bool
	Computed      bool
}

type providerSpec struct {
	ProviderName string
	BlockName    string
	Description  string
	Fields       []providerFieldSpec
}

var providerSpecs = []providerSpec{
	{
		ProviderName: "openai",
		BlockName:    "openai",
		Description:  "OpenAI provider credentials and endpoint configuration.",
		Fields: []providerFieldSpec{
			{TerraformName: "api_key", APIName: "apiKey", Description: "OpenAI API key.", Required: true, Sensitive: true},
			{TerraformName: "base_url", APIName: "baseUrl", Description: "Optional custom OpenAI-compatible base URL.", Required: false, Computed: true},
		},
	},
	{
		ProviderName: "azure_openai",
		BlockName:    "azure_openai",
		Description:  "Azure OpenAI provider configuration.",
		Fields: []providerFieldSpec{
			{TerraformName: "api_key", APIName: "apiKey", Description: "Azure OpenAI API key.", Required: true, Sensitive: true},
			{TerraformName: "base_url", APIName: "baseUrl", Description: "Azure OpenAI base URL.", Required: true},
			{TerraformName: "deployment_name", APIName: "deploymentName", Description: "Azure OpenAI deployment name.", Required: true},
			{TerraformName: "api_version", APIName: "apiVersion", Description: "Azure OpenAI API version.", Required: true},
		},
	},
	{
		ProviderName: "google_genai",
		BlockName:    "google_genai",
		Description:  "Google Gemini provider credentials.",
		Fields: []providerFieldSpec{
			{TerraformName: "api_key", APIName: "apiKey", Description: "Google GenAI API key.", Required: true, Sensitive: true},
		},
	},
	{
		ProviderName: "anthropic",
		BlockName:    "anthropic",
		Description:  "Anthropic provider credentials and optional endpoint configuration.",
		Fields: []providerFieldSpec{
			{TerraformName: "api_key", APIName: "apiKey", Description: "Anthropic API key.", Required: true, Sensitive: true},
			{TerraformName: "base_url", APIName: "baseUrl", Description: "Optional custom Anthropic-compatible base URL.", Required: false, Computed: true},
		},
	},
	{
		ProviderName: "openai_compatible",
		BlockName:    "openai_compatible",
		Description:  "Generic OpenAI-compatible provider credentials and endpoint configuration.",
		Fields: []providerFieldSpec{
			{TerraformName: "api_key", APIName: "apiKey", Description: "OpenAI-compatible API key.", Required: true, Sensitive: true},
			{TerraformName: "base_url", APIName: "baseUrl", Description: "OpenAI-compatible provider base URL.", Required: true},
		},
	},
}

func providerNames() []string {
	names := make([]string, 0, len(providerSpecs))
	for _, spec := range providerSpecs {
		names = append(names, spec.ProviderName)
	}

	return names
}

func providerSpecByName(name string) (providerSpec, bool) {
	for _, spec := range providerSpecs {
		if spec.ProviderName == name {
			return spec, true
		}
	}

	return providerSpec{}, false
}

func providerBlockAttrTypes(spec providerSpec) map[string]attr.Type {
	attrTypes := make(map[string]attr.Type, len(spec.Fields))
	for _, field := range spec.Fields {
		attrTypes[field.TerraformName] = types.StringType
	}

	return attrTypes
}

func providerBlockNull(spec providerSpec) types.Object {
	return types.ObjectNull(providerBlockAttrTypes(spec))
}

func providerBlockNames() []string {
	names := make([]string, 0, len(providerSpecs))
	for _, spec := range providerSpecs {
		names = append(names, spec.BlockName)
	}

	return names
}

func nonSensitiveProviderFields(spec providerSpec) []providerFieldSpec {
	fields := make([]providerFieldSpec, 0, len(spec.Fields))
	for _, field := range spec.Fields {
		if field.Sensitive {
			continue
		}

		fields = append(fields, field)
	}

	return fields
}

func providerDataSourceBlockAttrTypes(spec providerSpec) map[string]attr.Type {
	attrTypes := make(map[string]attr.Type, len(spec.Fields))
	for _, field := range nonSensitiveProviderFields(spec) {
		attrTypes[field.TerraformName] = types.StringType
	}

	return attrTypes
}

func providerDataSourceBlockNull(spec providerSpec) types.Object {
	return types.ObjectNull(providerDataSourceBlockAttrTypes(spec))
}
