package querysuggestions

import (
	"fmt"
	"sort"
	"strings"

	suggestions "github.com/algolia/algoliasearch-client-go/v4/algolia/query-suggestions"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func buildConfigurationWithIndex(model *QuerySuggestionsResourceModel) (*suggestions.ConfigurationWithIndex, diag.Diagnostics) {
	var diags diag.Diagnostics

	if model.SourceIndices.IsNull() || model.SourceIndices.IsUnknown() || len(model.SourceIndices.Elements()) == 0 {
		diags.AddError("Missing source indices", "At least one source_indices block is required.")
		return nil, diags
	}

	sourceIndices, sourceDiags := expandSourceIndices(model.SourceIndices)
	diags.Append(sourceDiags...)
	if diags.HasError() {
		return nil, diags
	}

	config := suggestions.NewConfigurationWithIndex(sourceIndices, model.IndexName.ValueString())

	if !model.Languages.IsNull() && !model.Languages.IsUnknown() {
		values := setStrings(model.Languages)
		sort.Strings(values)
		config.Languages = suggestions.ArrayOfStringAsLanguages(values)
	}

	if !model.Exclude.IsNull() && !model.Exclude.IsUnknown() {
		values := setStrings(model.Exclude)
		sort.Strings(values)
		config.Exclude = values
	}

	return config, diags
}

func hydrateQuerySuggestionsModel(resp *suggestions.ConfigurationResponse, model *QuerySuggestionsResourceModel) diag.Diagnostics {
	sourceIndices, diags := flattenSourceIndices(resp.GetSourceIndices())
	if diags.HasError() {
		return diags
	}

	model.ID = types.StringValue(querySuggestionsResourceID(resp.GetIndexName()))
	model.IndexName = types.StringValue(resp.GetIndexName())
	model.SourceIndices = sourceIndices
	model.Languages = stringSetFromSlice(languagesFromAPI(resp.GetLanguages()))
	model.Exclude = stringSetFromSlice(resp.GetExclude())

	return diags
}

func parseImportID(id string) (string, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" || strings.Contains(trimmed, "/") {
		return "", fmt.Errorf("expected import ID in the form <index_name>")
	}

	return trimmed, nil
}

func querySuggestionsResourceID(indexName string) string {
	return indexName
}

func expandSourceIndices(list types.List) ([]suggestions.SourceIndex, diag.Diagnostics) {
	var diags diag.Diagnostics

	sourceIndices := make([]suggestions.SourceIndex, 0, len(list.Elements()))
	for i, value := range list.Elements() {
		objValue, ok := value.(types.Object)
		if !ok {
			diags.AddError("Invalid source_indices value", fmt.Sprintf("source_indices[%d] is not an object.", i))
			return nil, diags
		}

		attrs := objValue.Attributes()
		indexName := attrs["index_name"].(types.String).ValueString()
		source := suggestions.NewSourceIndex(indexName)

		if value, ok := attrs["analytics_tags"].(types.Set); ok && !value.IsNull() && !value.IsUnknown() {
			tags := setStrings(value)
			sort.Strings(tags)
			source.AnalyticsTags = tags
		}

		if value, ok := attrs["facets"].(types.List); ok && !value.IsNull() && !value.IsUnknown() {
			facets := make([]suggestions.Facet, 0, len(value.Elements()))
			for _, facetValue := range value.Elements() {
				facetObject := facetValue.(types.Object)
				facetAttrs := facetObject.Attributes()
				facet := suggestions.NewFacet()
				attribute := facetAttrs["attribute"].(types.String).ValueString()
				amount := facetAttrs["amount"].(types.Int64).ValueInt64()
				facet.Attribute = &attribute
				amount32 := int32(amount)
				facet.Amount = &amount32
				facets = append(facets, *facet)
			}
			source.Facets = facets
		}

		if value, ok := attrs["min_hits"].(types.Int64); ok && !value.IsNull() && !value.IsUnknown() {
			minHits := int32(value.ValueInt64())
			source.MinHits = &minHits
		}
		if value, ok := attrs["min_letters"].(types.Int64); ok && !value.IsNull() && !value.IsUnknown() {
			minLetters := int32(value.ValueInt64())
			source.MinLetters = &minLetters
		}

		if value, ok := attrs["generate"].(types.List); ok && !value.IsNull() && !value.IsUnknown() {
			generate := make([][]string, 0, len(value.Elements()))
			for _, generateValue := range value.Elements() {
				listValue := generateValue.(types.List)
				generate = append(generate, listStrings(listValue))
			}
			source.Generate = generate
		}

		if value, ok := attrs["external"].(types.Set); ok && !value.IsNull() && !value.IsUnknown() {
			external := setStrings(value)
			sort.Strings(external)
			source.External = external
		}

		sourceIndices = append(sourceIndices, *source)
	}

	return sourceIndices, diags
}

func flattenSourceIndices(sourceIndices []suggestions.SourceIndex) (types.List, diag.Diagnostics) {
	values := make([]attr.Value, 0, len(sourceIndices))
	for _, source := range sourceIndices {
		facetValues := make([]attr.Value, 0, len(source.GetFacets()))
		for _, facet := range source.GetFacets() {
			value, diags := types.ObjectValue(facetModelAttrTypes, map[string]attr.Value{
				"attribute": nullableString(facet.Attribute),
				"amount":    nullableInt32(facet.Amount),
			})
			if diags.HasError() {
				return types.ListNull(sourceIndexModelType), diags
			}
			facetValues = append(facetValues, value)
		}
		facets, diags := types.ListValue(facetModelType, facetValues)
		if diags.HasError() {
			return types.ListNull(sourceIndexModelType), diags
		}

		generateValues := make([]attr.Value, 0, len(source.GetGenerate()))
		for _, group := range source.GetGenerate() {
			generateValues = append(generateValues, types.ListValueMust(types.StringType, stringSliceValues(group)))
		}
		generate, diags := types.ListValue(types.ListType{ElemType: types.StringType}, generateValues)
		if diags.HasError() {
			return types.ListNull(sourceIndexModelType), diags
		}

		value, diags := types.ObjectValue(sourceIndexModelAttrTypes, map[string]attr.Value{
			"index_name":     types.StringValue(source.GetIndexName()),
			"analytics_tags": stringSetFromSlice(source.GetAnalyticsTags()),
			"facets":         facets,
			"min_hits":       nullableInt32(source.MinHits),
			"min_letters":    nullableInt32(source.MinLetters),
			"generate":       generate,
			"external":       stringSetFromSlice(source.GetExternal()),
		})
		if diags.HasError() {
			return types.ListNull(sourceIndexModelType), diags
		}
		values = append(values, value)
	}

	return types.ListValue(sourceIndexModelType, values)
}

func listStrings(value types.List) []string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	values := make([]string, 0, len(value.Elements()))
	for _, element := range value.Elements() {
		if stringValue, ok := element.(types.String); ok && !stringValue.IsNull() && !stringValue.IsUnknown() {
			values = append(values, stringValue.ValueString())
		}
	}
	return values
}

func setStrings(value types.Set) []string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	values := make([]string, 0, len(value.Elements()))
	for _, element := range value.Elements() {
		if stringValue, ok := element.(types.String); ok && !stringValue.IsNull() && !stringValue.IsUnknown() {
			values = append(values, stringValue.ValueString())
		}
	}
	return values
}

func stringSetFromSlice(values []string) types.Set {
	if len(values) == 0 {
		return types.SetNull(types.StringType)
	}
	return types.SetValueMust(types.StringType, stringSliceValues(values))
}

func stringSliceValues(values []string) []attr.Value {
	result := make([]attr.Value, 0, len(values))
	for _, value := range values {
		result = append(result, types.StringValue(value))
	}
	return result
}

func nullableString(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func nullableInt32(value *int32) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*value))
}

func languagesFromAPI(value suggestions.Languages) []string {
	actual := value.GetActualInstance()
	if actual == nil {
		return nil
	}
	languages, ok := actual.([]string)
	if !ok {
		return nil
	}
	return languages
}
