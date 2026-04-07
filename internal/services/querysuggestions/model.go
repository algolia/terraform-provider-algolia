package querysuggestions

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type QuerySuggestionsResourceModel struct {
	ID            types.String `tfsdk:"id"`
	IndexName     types.String `tfsdk:"index_name"`
	Region        types.String `tfsdk:"region"`
	SourceIndices types.List   `tfsdk:"source_indices"`
	Languages     types.Set    `tfsdk:"languages"`
	Exclude       types.Set    `tfsdk:"exclude"`
}

var (
	facetModelAttrTypes = map[string]attr.Type{
		"attribute": types.StringType,
		"amount":    types.Int64Type,
	}
	facetModelType = types.ObjectType{AttrTypes: facetModelAttrTypes}

	sourceIndexModelAttrTypes = map[string]attr.Type{
		"index_name":     types.StringType,
		"analytics_tags": types.SetType{ElemType: types.StringType},
		"facets":         types.ListType{ElemType: facetModelType},
		"min_hits":       types.Int64Type,
		"min_letters":    types.Int64Type,
		"generate":       types.ListType{ElemType: types.ListType{ElemType: types.StringType}},
		"external":       types.SetType{ElemType: types.StringType},
	}
	sourceIndexModelType = types.ObjectType{AttrTypes: sourceIndexModelAttrTypes}
)

