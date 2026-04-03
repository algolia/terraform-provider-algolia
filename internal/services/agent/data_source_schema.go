package agent

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func agentDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read an Algolia Agent Studio agent.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Description: "The unique identifier (UUID) of the agent.",
				Required:    true,
			},
			"name": datasourceschema.StringAttribute{
				Description: "The agent display name.",
				Computed:    true,
			},
			"description": datasourceschema.StringAttribute{
				Description: "A summary of the agent's purpose.",
				Computed:    true,
			},
			"instructions": datasourceschema.StringAttribute{
				Description: "The agent prompt.",
				Computed:    true,
			},
			"system_prompt": datasourceschema.StringAttribute{
				Description: "System-level rules and constraints.",
				Computed:    true,
			},
			"provider_id": datasourceschema.StringAttribute{
				Description: "The LLM provider identifier (UUID).",
				Computed:    true,
			},
			"model": datasourceschema.StringAttribute{
				Description: "The LLM model name.",
				Computed:    true,
			},
			"template_type": datasourceschema.StringAttribute{
				Description: "Template classification for the agent.",
				Computed:    true,
			},
			"config": datasourceschema.StringAttribute{
				Description: "JSON-encoded configuration parameters.",
				Computed:    true,
			},
			"publish": datasourceschema.BoolAttribute{
				Description: "Whether the agent is published.",
				Computed:    true,
			},
			"deletion_protection": datasourceschema.BoolAttribute{
				Description: "Whether deletion protection is enabled.",
				Computed:    true,
			},
			"status": datasourceschema.StringAttribute{
				Description: "The agent status: draft or published.",
				Computed:    true,
			},
			"created_at": datasourceschema.StringAttribute{
				Description: "ISO 8601 timestamp of when the agent was created.",
				Computed:    true,
			},
			"updated_at": datasourceschema.StringAttribute{
				Description: "ISO 8601 timestamp of when the agent was last updated.",
				Computed:    true,
			},
		},
		Blocks: map[string]datasourceschema.Block{
			"tool_algolia_search":    toolAlgoliaSearchDataSourceBlockSchema(),
			"tool_algolia_recommend": toolAlgoliaRecommendDataSourceBlockSchema(),
			"tool_client_side":       toolClientSideDataSourceBlockSchema(),
			"tool_mcp":               toolMCPDataSourceBlockSchema(),
		},
	}
}

func toolAlgoliaSearchDataSourceBlockSchema() datasourceschema.Block {
	return datasourceschema.ListNestedBlock{
		Description: "Algolia search index tool configuration.",
		NestedObject: datasourceschema.NestedBlockObject{
			Attributes: map[string]datasourceschema.Attribute{
				"name": datasourceschema.StringAttribute{
					Computed: true,
				},
			},
			Blocks: map[string]datasourceschema.Block{
				"index": datasourceschema.ListNestedBlock{
					NestedObject: datasourceschema.NestedBlockObject{
						Attributes: map[string]datasourceschema.Attribute{
							"name":                 datasourceschema.StringAttribute{Computed: true},
							"description":          datasourceschema.StringAttribute{Computed: true},
							"enhanced_description": datasourceschema.StringAttribute{Computed: true},
							"search_parameters":    datasourceschema.StringAttribute{Computed: true},
						},
					},
				},
			},
		},
	}
}

func toolAlgoliaRecommendDataSourceBlockSchema() datasourceschema.Block {
	return datasourceschema.ListNestedBlock{
		Description: "Algolia recommend tool configuration.",
		NestedObject: datasourceschema.NestedBlockObject{
			Attributes: map[string]datasourceschema.Attribute{
				"name":                              datasourceschema.StringAttribute{Computed: true},
				"predefined_recommend_parameters": datasourceschema.StringAttribute{Computed: true},
			},
			Blocks: map[string]datasourceschema.Block{
				"allowed_config": datasourceschema.ListNestedBlock{
					NestedObject: datasourceschema.NestedBlockObject{
						Attributes: map[string]datasourceschema.Attribute{
							"index":      datasourceschema.StringAttribute{Computed: true},
							"model_name": datasourceschema.StringAttribute{Computed: true},
							"description": datasourceschema.StringAttribute{Computed: true},
						},
					},
				},
			},
		},
	}
}

func toolClientSideDataSourceBlockSchema() datasourceschema.Block {
	return datasourceschema.ListNestedBlock{
		Description: "Client-side tool configuration.",
		NestedObject: datasourceschema.NestedBlockObject{
			Attributes: map[string]datasourceschema.Attribute{
				"name":         datasourceschema.StringAttribute{Computed: true},
				"description":  datasourceschema.StringAttribute{Computed: true},
				"input_schema": datasourceschema.StringAttribute{Computed: true},
			},
		},
	}
}

func toolMCPDataSourceBlockSchema() datasourceschema.Block {
	return datasourceschema.ListNestedBlock{
		Description: "MCP server tool configuration.",
		NestedObject: datasourceschema.NestedBlockObject{
			Attributes: map[string]datasourceschema.Attribute{
				"name":      datasourceschema.StringAttribute{Computed: true},
				"url":       datasourceschema.StringAttribute{Computed: true},
				"transport": datasourceschema.StringAttribute{Computed: true},
				"headers": datasourceschema.MapAttribute{
					Computed:    true,
					ElementType: types.StringType,
				},
			},
			Blocks: map[string]datasourceschema.Block{
				"allowed_tool": datasourceschema.ListNestedBlock{
					NestedObject: datasourceschema.NestedBlockObject{
						Attributes: map[string]datasourceschema.Attribute{
							"name":              datasourceschema.StringAttribute{Computed: true},
							"requires_approval": datasourceschema.BoolAttribute{Computed: true},
							"alias":             datasourceschema.StringAttribute{Computed: true},
						},
					},
				},
			},
		},
	}
}
