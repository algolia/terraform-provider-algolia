package query_suggestions

import (
	"context"
	"encoding/json"

	suggestions "github.com/algolia/algoliasearch-client-go/v4/algolia/query-suggestions"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// flattenConfigurationResponse maps a ConfigurationResponse onto the Terraform model.
// The region and deletion_protection fields are not present in the API response and must be preserved by the caller.
func flattenConfigurationResponse(ctx context.Context, resp *suggestions.ConfigurationResponse, model *QuerySuggestionsConfigResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.IndexName = types.StringValue(resp.IndexName)
	model.EnablePersonalization = types.BoolValue(resp.EnablePersonalization)
	model.AllowSpecialCharacters = types.BoolValue(resp.AllowSpecialCharacters)

	diags.Append(flattenLanguages(ctx, resp.Languages, model)...)
	if diags.HasError() {
		return diags
	}

	excludeList, d := flattenStringList(ctx, resp.Exclude)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	model.Exclude = excludeList

	sourceIndicesList, d := flattenSourceIndices(ctx, resp.SourceIndices)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	model.SourceIndices = sourceIndicesList

	return diags
}

// flattenLanguages maps the Languages union type onto the two split model fields.
func flattenLanguages(_ context.Context, langs suggestions.Languages, model *QuerySuggestionsConfigResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if langs.ArrayOfString != nil {
		list, d := types.ListValueFrom(context.Background(), types.StringType, *langs.ArrayOfString)
		diags.Append(d...)
		model.Languages = list
		model.LanguagesEnabled = types.BoolNull()
		return diags
	}

	if langs.Bool != nil {
		model.Languages = types.ListValueMust(types.StringType, []attr.Value{})
		model.LanguagesEnabled = types.BoolValue(*langs.Bool)
		return diags
	}

	model.Languages = types.ListValueMust(types.StringType, []attr.Value{})
	model.LanguagesEnabled = types.BoolNull()
	return diags
}

// flattenSourceIndices converts []suggestions.SourceIndex into a types.List of SourceIndexModel objects.
func flattenSourceIndices(ctx context.Context, sourceIndices []suggestions.SourceIndex) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	sourceIndexAttrTypes := sourceIndexAttrTypes()

	if len(sourceIndices) == 0 {
		return types.ListValueMust(types.ObjectType{AttrTypes: sourceIndexAttrTypes}, []attr.Value{}), diags
	}

	elems := make([]attr.Value, 0, len(sourceIndices))
	for _, si := range sourceIndices {
		analyticsTags, d := flattenStringList(ctx, si.AnalyticsTags)
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: sourceIndexAttrTypes}), diags
		}

		external, d := flattenStringList(ctx, si.External)
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: sourceIndexAttrTypes}), diags
		}

		facetsList, d := flattenFacets(ctx, si.Facets)
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: sourceIndexAttrTypes}), diags
		}

		generateStr := types.StringNull()
		if si.Generate != nil {
			raw, err := json.Marshal(si.Generate)
			if err == nil {
				generateStr = types.StringValue(string(raw))
			}
		}

		replicas := types.BoolNull()
		if si.Replicas != nil {
			replicas = types.BoolValue(*si.Replicas)
		}

		minHits := types.Int64Null()
		if si.MinHits != nil {
			minHits = types.Int64Value(int64(*si.MinHits))
		}

		minLetters := types.Int64Null()
		if si.MinLetters != nil {
			minLetters = types.Int64Value(int64(*si.MinLetters))
		}

		obj, d := types.ObjectValue(sourceIndexAttrTypes, map[string]attr.Value{
			"index_name":     types.StringValue(si.IndexName),
			"replicas":       replicas,
			"analytics_tags": analyticsTags,
			"min_hits":       minHits,
			"min_letters":    minLetters,
			"generate":       generateStr,
			"external":       external,
			"facets":         facetsList,
		})
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: sourceIndexAttrTypes}), diags
		}

		elems = append(elems, obj)
	}

	list, d := types.ListValue(types.ObjectType{AttrTypes: sourceIndexAttrTypes}, elems)
	diags.Append(d...)
	return list, diags
}

// flattenFacets converts []suggestions.Facet into a types.List of FacetModel objects.
func flattenFacets(ctx context.Context, facets []suggestions.Facet) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	facetAttrTypes := facetAttrTypes()

	if len(facets) == 0 {
		return types.ListValueMust(types.ObjectType{AttrTypes: facetAttrTypes}, []attr.Value{}), diags
	}

	elems := make([]attr.Value, 0, len(facets))
	for _, f := range facets {
		attribute := types.StringNull()
		if f.Attribute != nil {
			attribute = types.StringValue(*f.Attribute)
		}

		amount := types.Int64Null()
		if f.Amount != nil {
			amount = types.Int64Value(int64(*f.Amount))
		}

		obj, d := types.ObjectValue(facetAttrTypes, map[string]attr.Value{
			"attribute": attribute,
			"amount":    amount,
		})
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: facetAttrTypes}), diags
		}

		elems = append(elems, obj)
	}

	list, d := types.ListValue(types.ObjectType{AttrTypes: facetAttrTypes}, elems)
	diags.Append(d...)
	return list, diags
}

// flattenStringList converts a []string into a types.List of StringType.
func flattenStringList(_ context.Context, strs []string) (types.List, diag.Diagnostics) {
	if len(strs) == 0 {
		return types.ListValueMust(types.StringType, []attr.Value{}), nil
	}
	elems := make([]attr.Value, len(strs))
	for i, s := range strs {
		elems[i] = types.StringValue(s)
	}
	return types.ListValue(types.StringType, elems)
}

// sourceIndexAttrTypes returns the attr.Type map for SourceIndexModel, including the nested facets list.
func sourceIndexAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"index_name":     types.StringType,
		"replicas":       types.BoolType,
		"analytics_tags": types.ListType{ElemType: types.StringType},
		"min_hits":       types.Int64Type,
		"min_letters":    types.Int64Type,
		"generate":       types.StringType,
		"external":       types.ListType{ElemType: types.StringType},
		"facets":         types.ListType{ElemType: types.ObjectType{AttrTypes: facetAttrTypes()}},
	}
}

// facetAttrTypes returns the attr.Type map for FacetModel.
func facetAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"attribute": types.StringType,
		"amount":    types.Int64Type,
	}
}
