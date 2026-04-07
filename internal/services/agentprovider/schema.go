package agentprovider

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func agentProviderResourceSchema() schema.Schema {
	blocks := make(map[string]schema.Block, len(providerSpecs))
	for _, spec := range providerSpecs {
		blocks[spec.BlockName] = providerBlockSchema(spec)
	}

	return schema.Schema{
		Description: "Manages an Algolia Agent Studio language model provider.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier (UUID) of the provider.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Display name for the Agent Studio provider.",
				Required:    true,
			},
			"provider_name": schema.StringAttribute{
				Description: "Provider type. Supported values: " + strings.Join(providerNames(), ", ") + ".",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(providerNames()...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "ISO 8601 timestamp of when the provider was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "ISO 8601 timestamp of when the provider was last updated.",
				Computed:    true,
			},
			"last_used_at": schema.StringAttribute{
				Description: "ISO 8601 timestamp of the last provider usage, if available.",
				Computed:    true,
			},
		},
		Blocks: blocks,
	}
}

func providerBlockSchema(spec providerSpec) schema.Block {
	attributes := make(map[string]schema.Attribute, len(spec.Fields))
	for _, field := range spec.Fields {
		description := field.Description
		if field.Required {
			description += " Required when this provider block is configured."
		}

		attributes[field.TerraformName] = schema.StringAttribute{
			Description: description,
			Optional:    true,
			Sensitive:   field.Sensitive,
			Computed:    field.Computed,
		}
	}

	return schema.SingleNestedBlock{
		Description: spec.Description,
		Attributes:  attributes,
	}
}
