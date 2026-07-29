package querysuggestions

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type QuerySuggestionsResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	IndexName              types.String `tfsdk:"index_name"`
	SourceIndices          types.List   `tfsdk:"source_indices"`
	Languages              types.Set    `tfsdk:"languages"`
	AllLanguages           types.Bool   `tfsdk:"all_languages"`
	Exclude                types.Set    `tfsdk:"exclude"`
	EnablePersonalization  types.Bool   `tfsdk:"enable_personalization"`
	AllowSpecialCharacters types.Bool   `tfsdk:"allow_special_characters"`
}

type QuerySuggestionsDataSourceModel = QuerySuggestionsResourceModel

var (
	facetModelAttrTypes = map[string]attr.Type{
		"attribute": types.StringType,
		"amount":    types.Int64Type,
	}
	facetModelType = types.ObjectType{AttrTypes: facetModelAttrTypes}

	// generateElementType is the element type of source_indices.generate: each
	// element is one facet-name combination, so the attribute is a list of
	// lists of strings.
	generateElementType = types.ListType{ElemType: types.StringType}

	sourceIndexModelAttrTypes = map[string]attr.Type{
		"index_name":     types.StringType,
		"replicas":       types.BoolType,
		"analytics_tags": types.SetType{ElemType: types.StringType},
		"facets":         types.ListType{ElemType: facetModelType},
		"min_hits":       types.Int64Type,
		"min_letters":    types.Int64Type,
		"generate":       types.ListType{ElemType: generateElementType},
		"external":       types.SetType{ElemType: types.StringType},
	}
	sourceIndexModelType = types.ObjectType{AttrTypes: sourceIndexModelAttrTypes}
)
