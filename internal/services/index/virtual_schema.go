package index

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func virtualIndexResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages an Algolia virtual replica index and its settings. A virtual replica shares " +
			"the primary index's records and applies its own custom ranking, so it is a view over the " +
			"primary rather than a copy of it.\n\n" +
			"This resource owns a `virtual(<name>)` entry in the primary index's `replicas` setting. " +
			"That setting also holds the primary's standard replicas, which `algolia_index`'s " +
			"`advanced.replicas` owns - the two are disjoint, so both resources can write it without " +
			"overwriting each other, and a `virtual(...)` entry named in `advanced.replicas` is " +
			"rejected. Remove this resource to unlink the replica; unlinking empties it.\n\n" +
			"The `virtual(...)` form is also what distinguishes a virtual replica from a standard one: " +
			"listing a replica under its plain name makes Algolia keep it as a standard replica and " +
			"copy the primary index's records into it. This resource manages virtual replicas only, so " +
			"it reports an error for an index Algolia holds as a standard replica - manage that with " +
			"`algolia_index` instead.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the virtual replica index, without the virtual() wrapper.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"primary_index_name": schema.StringAttribute{
				Description: "The primary index linked to this virtual replica. Changing it forces " +
					"replacement: Algolia has no way to move a replica between primaries.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"deletion_protection": schema.BoolAttribute{
				Description: "Whether to prevent deletion of the virtual index. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"entries": schema.Int64Attribute{
				Description: "The number of records in the virtual index.",
				Computed:    true,
			},
			"data_size": schema.Int64Attribute{
				Description: "The size of the virtual index data in bytes.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation date of the virtual index.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					useStateForKnownString(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The last update date of the virtual index.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"attributes":     schema.SingleNestedBlock{Description: "Configuration for searchable and retrievable attributes.", Attributes: virtualAttributesBlockSchema()},
			"ranking":        schema.SingleNestedBlock{Description: "Configuration for virtual replica ranking behavior.", Attributes: virtualRankingBlockSchema()},
			"faceting":       schema.SingleNestedBlock{Description: "Configuration for faceting behavior.", Attributes: virtualFacetingBlockSchema()},
			"highlighting":   schema.SingleNestedBlock{Description: "Configuration for highlighting and snippeting.", Attributes: highlightingBlockSchema()},
			"pagination":     schema.SingleNestedBlock{Description: "Configuration for pagination.", Attributes: paginationBlockSchema()},
			"typos":          schema.SingleNestedBlock{Description: "Configuration for typo tolerance.", Attributes: virtualTyposBlockSchema()},
			"languages":      schema.SingleNestedBlock{Description: "Configuration for language-specific settings.", Attributes: virtualLanguagesBlockSchema()},
			"query_strategy": schema.SingleNestedBlock{Description: "Configuration for query strategy.", Attributes: queryStrategyBlockSchema()},
			"performance":    schema.SingleNestedBlock{Description: "Configuration for performance settings.", Attributes: performanceBlockSchema()},
			"advanced":       schema.SingleNestedBlock{Description: "Configuration for advanced settings.", Attributes: virtualAdvancedBlockSchema()},
		},
	}
}

func virtualAttributesBlockSchema() map[string]schema.Attribute {
	attributes := attributesBlockSchema()
	attributes["searchable_attributes"] = computedVirtualList(attributes["searchable_attributes"])
	attributes["attribute_for_distinct"] = computedVirtualString(attributes["attribute_for_distinct"])
	return attributes
}

func virtualFacetingBlockSchema() map[string]schema.Attribute {
	attributes := facetingBlockSchema()
	attributes["attributes_for_faceting"] = computedVirtualList(attributes["attributes_for_faceting"])
	return attributes
}

func virtualTyposBlockSchema() map[string]schema.Attribute {
	attributes := typosBlockSchema()
	attributes["disable_typo_tolerance_on_attributes"] = computedVirtualList(attributes["disable_typo_tolerance_on_attributes"])
	return attributes
}

func virtualLanguagesBlockSchema() map[string]schema.Attribute {
	attributes := languagesBlockSchema()
	attributes["attributes_to_transliterate"] = computedVirtualList(attributes["attributes_to_transliterate"])
	attributes["decompounded_attributes"] = computedVirtualString(attributes["decompounded_attributes"])
	attributes["index_languages"] = computedVirtualList(attributes["index_languages"])
	return attributes
}

func virtualAdvancedBlockSchema() map[string]schema.Attribute {
	attributes := advancedBlockSchema()
	attributes["separators_to_index"] = computedVirtualString(attributes["separators_to_index"])
	return attributes
}

func computedVirtualList(attribute schema.Attribute) schema.ListAttribute {
	computed := attribute.(schema.ListAttribute)
	computed.Description += " Algolia inherits this setting from the primary index, so it cannot be configured on a virtual replica."
	computed.Optional = false
	computed.Computed = true
	return computed
}

func computedVirtualString(attribute schema.Attribute) schema.StringAttribute {
	computed := attribute.(schema.StringAttribute)
	computed.Description += " Algolia inherits this setting from the primary index, so it cannot be configured on a virtual replica."
	computed.Optional = false
	computed.Computed = true
	return computed
}

func virtualIndexDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read the settings of an existing Algolia virtual replica index.",
		Attributes: map[string]datasourceschema.Attribute{
			"name": datasourceschema.StringAttribute{
				Description: "The name of the virtual replica index.",
				Required:    true,
			},
			"primary_index_name": datasourceschema.StringAttribute{
				Description: "The primary index linked to this virtual replica.",
				Computed:    true,
			},
			"entries": datasourceschema.Int64Attribute{
				Description: "The number of records in the virtual index.",
				Computed:    true,
			},
			"data_size": datasourceschema.Int64Attribute{
				Description: "The size of the virtual index data in bytes.",
				Computed:    true,
			},
			"created_at": datasourceschema.StringAttribute{
				Description: "The creation date of the virtual index.",
				Computed:    true,
			},
			"updated_at": datasourceschema.StringAttribute{
				Description: "The last update date of the virtual index.",
				Computed:    true,
			},
		},
		Blocks: map[string]datasourceschema.Block{
			"attributes":     datasourceschema.SingleNestedBlock{Description: "Configuration for searchable and retrievable attributes.", Attributes: attributesDataSourceBlockSchema()},
			"ranking":        datasourceschema.SingleNestedBlock{Description: "Configuration for virtual replica ranking behavior.", Attributes: virtualRankingDataSourceBlockSchema()},
			"faceting":       datasourceschema.SingleNestedBlock{Description: "Configuration for faceting behavior.", Attributes: facetingDataSourceBlockSchema()},
			"highlighting":   datasourceschema.SingleNestedBlock{Description: "Configuration for highlighting and snippeting.", Attributes: highlightingDataSourceBlockSchema()},
			"pagination":     datasourceschema.SingleNestedBlock{Description: "Configuration for pagination.", Attributes: paginationDataSourceBlockSchema()},
			"typos":          datasourceschema.SingleNestedBlock{Description: "Configuration for typo tolerance.", Attributes: typosDataSourceBlockSchema()},
			"languages":      datasourceschema.SingleNestedBlock{Description: "Configuration for language-specific settings.", Attributes: languagesDataSourceBlockSchema()},
			"query_strategy": datasourceschema.SingleNestedBlock{Description: "Configuration for query strategy.", Attributes: queryStrategyDataSourceBlockSchema()},
			"performance":    datasourceschema.SingleNestedBlock{Description: "Configuration for performance settings.", Attributes: performanceDataSourceBlockSchema()},
			"advanced":       datasourceschema.SingleNestedBlock{Description: "Configuration for advanced settings.", Attributes: advancedDataSourceBlockSchema()},
		},
	}
}

func virtualRankingBlockSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"custom_ranking":       rankingBlockSchema()["custom_ranking"],
		"relevancy_strictness": rankingBlockSchema()["relevancy_strictness"],
	}
}

func virtualRankingDataSourceBlockSchema() map[string]datasourceschema.Attribute {
	return map[string]datasourceschema.Attribute{
		"custom_ranking":       rankingDataSourceBlockSchema()["custom_ranking"],
		"relevancy_strictness": rankingDataSourceBlockSchema()["relevancy_strictness"],
	}
}
