package collection

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func collectionDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Read-only lookup of an Algolia Collection by ID.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Description: "The unique identifier (UUID) of the collection.",
				Required:    true,
			},
			"name": datasourceschema.StringAttribute{
				Description: "Display name of the collection.",
				Computed:    true,
			},
			"index_name": datasourceschema.StringAttribute{
				Description: "Name of the index the collection belongs to.",
				Computed:    true,
			},
			"description": datasourceschema.StringAttribute{
				Description: "Free-form description of the collection.",
				Computed:    true,
			},
			"records": datasourceschema.ListAttribute{
				Description: "objectIDs currently in the collection.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"status": datasourceschema.StringAttribute{
				Description: "Commit status reported by the API.",
				Computed:    true,
			},
			"created_at": datasourceschema.StringAttribute{
				Description: "RFC 3339 timestamp of when the collection was created.",
				Computed:    true,
			},
			"updated_at": datasourceschema.StringAttribute{
				Description: "RFC 3339 timestamp of when the collection was last updated.",
				Computed:    true,
			},
		},
		Blocks: map[string]datasourceschema.Block{
			"conditions": datasourceschema.SingleNestedBlock{
				Description: "Filter rules selecting records.",
				Blocks: map[string]datasourceschema.Block{
					"facet_filter":   filterGroupDataSourceBlockSchema(),
					"numeric_filter": filterGroupDataSourceBlockSchema(),
				},
			},
		},
	}
}

func filterGroupDataSourceBlockSchema() datasourceschema.ListNestedBlock {
	return datasourceschema.ListNestedBlock{
		Description: "A single AND clause of OR-ed filters.",
		NestedObject: datasourceschema.NestedBlockObject{
			Attributes: map[string]datasourceschema.Attribute{
				"filters": datasourceschema.ListAttribute{
					Description: "OR-ed filter expressions.",
					Computed:    true,
					ElementType: types.StringType,
				},
			},
		},
	}
}
