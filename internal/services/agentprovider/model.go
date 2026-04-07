package agentprovider

import "github.com/hashicorp/terraform-plugin-framework/types"

type AgentProviderResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	ProviderName     types.String `tfsdk:"provider_name"`
	OpenAI           types.Object `tfsdk:"openai"`
	AzureOpenAI      types.Object `tfsdk:"azure_openai"`
	GoogleGenAI      types.Object `tfsdk:"google_genai"`
	Anthropic        types.Object `tfsdk:"anthropic"`
	OpenAICompatible types.Object `tfsdk:"openai_compatible"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
	LastUsedAt       types.String `tfsdk:"last_used_at"`
}

type AgentProviderModelsDataSourceModel struct {
	ProviderID types.String `tfsdk:"provider_id"`
	Models     types.List   `tfsdk:"models"`
}
