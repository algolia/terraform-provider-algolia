package agentprovider

import datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
import "github.com/hashicorp/terraform-plugin-framework/types"

func agentProviderDataSourceSchema() datasourceschema.Schema {
	blocks := make(map[string]datasourceschema.Block, len(providerSpecs))
	for _, spec := range providerSpecs {
		blocks[spec.BlockName] = providerDataSourceBlockSchema(spec)
	}

	return datasourceschema.Schema{
		Description: "Use this data source to read an Algolia Agent Studio language model provider without exposing secret credentials.",
		Attributes: map[string]datasourceschema.Attribute{
			"provider_id": datasourceschema.StringAttribute{
				Description: "The unique identifier (UUID) of the Agent Studio provider.",
				Required:    true,
			},
			"id": datasourceschema.StringAttribute{
				Description: "The unique identifier (UUID) of the provider.",
				Computed:    true,
			},
			"name": datasourceschema.StringAttribute{
				Description: "Display name for the Agent Studio provider.",
				Computed:    true,
			},
			"provider_name": datasourceschema.StringAttribute{
				Description: "Provider type.",
				Computed:    true,
			},
			"created_at": datasourceschema.StringAttribute{
				Description: "ISO 8601 timestamp of when the provider was created.",
				Computed:    true,
			},
			"updated_at": datasourceschema.StringAttribute{
				Description: "ISO 8601 timestamp of when the provider was last updated.",
				Computed:    true,
			},
			"last_used_at": datasourceschema.StringAttribute{
				Description: "ISO 8601 timestamp of the last provider usage, if available.",
				Computed:    true,
			},
		},
		Blocks: blocks,
	}
}

func agentProviderModelsDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read the available models for an Algolia Agent Studio provider.",
		Attributes: map[string]datasourceschema.Attribute{
			"provider_id": datasourceschema.StringAttribute{
				Description: "The unique identifier (UUID) of the Agent Studio provider.",
				Required:    true,
			},
			"models": datasourceschema.ListAttribute{
				Description: "Available model identifiers for the provider.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func providerDataSourceBlockSchema(spec providerSpec) datasourceschema.Block {
	attributes := make(map[string]datasourceschema.Attribute, len(spec.Fields))
	for _, field := range nonSensitiveProviderFields(spec) {
		attributes[field.TerraformName] = datasourceschema.StringAttribute{
			Description: field.Description,
			Computed:    true,
		}
	}

	return datasourceschema.SingleNestedBlock{
		Description: spec.Description,
		Attributes:  attributes,
	}
}
