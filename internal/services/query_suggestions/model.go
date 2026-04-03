package query_suggestions

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// QuerySuggestionsConfigResourceModel describes the Terraform resource data model for algolia_query_suggestions_config.
type QuerySuggestionsConfigResourceModel struct {
	IndexName              types.String `tfsdk:"index_name"`
	Region                 types.String `tfsdk:"region"`
	SourceIndices          types.List   `tfsdk:"source_index"`
	Languages              types.List   `tfsdk:"languages"`
	LanguagesEnabled       types.Bool   `tfsdk:"languages_enabled"`
	Exclude                types.List   `tfsdk:"exclude"`
	EnablePersonalization  types.Bool   `tfsdk:"enable_personalization"`
	AllowSpecialCharacters types.Bool   `tfsdk:"allow_special_characters"`
	DeletionProtection     types.Bool   `tfsdk:"deletion_protection"`
}

// SourceIndexModel represents a source_index block.
type SourceIndexModel struct {
	IndexName     types.String `tfsdk:"index_name"`
	Replicas      types.Bool   `tfsdk:"replicas"`
	AnalyticsTags types.List   `tfsdk:"analytics_tags"`
	Facets        types.List   `tfsdk:"facets"`
	MinHits       types.Int64  `tfsdk:"min_hits"`
	MinLetters    types.Int64  `tfsdk:"min_letters"`
	// JSON-encoded [][]string — use jsonencode([["brand"], ["category", "brand"]]) in HCL.
	Generate types.String `tfsdk:"generate"`
	External types.List   `tfsdk:"external"`
}

// FacetModel represents a facet block inside a source_index block.
type FacetModel struct {
	Attribute types.String `tfsdk:"attribute"`
	Amount    types.Int64  `tfsdk:"amount"`
}
