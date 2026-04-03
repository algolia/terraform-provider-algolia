package agent

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AgentResourceModel describes the Terraform resource data model for algolia_agent.
type AgentResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Instructions       types.String `tfsdk:"instructions"`
	SystemPrompt       types.String `tfsdk:"system_prompt"`
	ProviderID         types.String `tfsdk:"provider_id"`
	Model              types.String `tfsdk:"model"`
	TemplateType       types.String `tfsdk:"template_type"`
	Config             types.String `tfsdk:"config"`
	Publish            types.Bool   `tfsdk:"publish"`
	DeletionProtection types.Bool   `tfsdk:"deletion_protection"`

	// Tool blocks
	ToolAlgoliaSearch    types.List `tfsdk:"tool_algolia_search"`
	ToolAlgoliaRecommend types.List `tfsdk:"tool_algolia_recommend"`
	ToolClientSide       types.List `tfsdk:"tool_client_side"`
	ToolMCP              types.List `tfsdk:"tool_mcp"`

	// Computed
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

// AgentDataSourceModel describes the Terraform data source model for algolia_agent.
type AgentDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	Instructions types.String `tfsdk:"instructions"`
	SystemPrompt types.String `tfsdk:"system_prompt"`
	ProviderID   types.String `tfsdk:"provider_id"`
	Model        types.String `tfsdk:"model"`
	TemplateType types.String `tfsdk:"template_type"`
	Config       types.String `tfsdk:"config"`
	Publish      types.Bool   `tfsdk:"publish"`

	// Tool blocks
	ToolAlgoliaSearch    types.List `tfsdk:"tool_algolia_search"`
	ToolAlgoliaRecommend types.List `tfsdk:"tool_algolia_recommend"`
	ToolClientSide       types.List `tfsdk:"tool_client_side"`
	ToolMCP              types.List `tfsdk:"tool_mcp"`

	// Computed
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

// ToolAlgoliaSearchModel represents an algolia_search_index tool.
type ToolAlgoliaSearchModel struct {
	Name    types.String `tfsdk:"name"`
	Indices types.List   `tfsdk:"index"`
}

// AlgoliaSearchIndexModel represents a single index within an algolia_search_index tool.
type AlgoliaSearchIndexModel struct {
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	EnhancedDescription types.String `tfsdk:"enhanced_description"`
	SearchParameters    types.String `tfsdk:"search_parameters"`
}

// ToolAlgoliaRecommendModel represents an algolia_recommend tool.
type ToolAlgoliaRecommendModel struct {
	Name                          types.String `tfsdk:"name"`
	AllowedConfigs                types.List   `tfsdk:"allowed_config"`
	PredefinedRecommendParameters types.String `tfsdk:"predefined_recommend_parameters"`
}

// AlgoliaRecommendConfigModel represents a single config within an algolia_recommend tool.
type AlgoliaRecommendConfigModel struct {
	Index       types.String `tfsdk:"index"`
	ModelName   types.String `tfsdk:"model_name"`
	Description types.String `tfsdk:"description"`
}

// ToolClientSideModel represents a client_side tool.
type ToolClientSideModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	InputSchema types.String `tfsdk:"input_schema"`
}

// ToolMCPModel represents an mcp_tools tool.
type ToolMCPModel struct {
	Name         types.String `tfsdk:"name"`
	URL          types.String `tfsdk:"url"`
	Transport    types.String `tfsdk:"transport"`
	Headers      types.Map    `tfsdk:"headers"`
	AllowedTools types.List   `tfsdk:"allowed_tool"`
}

// MCPAllowedToolModel represents a single allowed tool within an MCP tool.
type MCPAllowedToolModel struct {
	Name             types.String `tfsdk:"name"`
	RequiresApproval types.Bool   `tfsdk:"requires_approval"`
	Alias            types.String `tfsdk:"alias"`
}
