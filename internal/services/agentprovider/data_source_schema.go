package agentprovider

import datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
import "github.com/hashicorp/terraform-plugin-framework/types"

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
