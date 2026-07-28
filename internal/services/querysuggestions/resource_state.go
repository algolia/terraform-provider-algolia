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

	if !model.EnablePersonalization.IsNull() && !model.EnablePersonalization.IsUnknown() {
		config.SetEnablePersonalization(model.EnablePersonalization.ValueBool())
	}

	if !model.AllowSpecialCharacters.IsNull() && !model.AllowSpecialCharacters.IsUnknown() {
		config.SetAllowSpecialCharacters(model.AllowSpecialCharacters.ValueBool())
	}

	return config, diags
}

// configurationFromWithIndex converts the create payload into the update
// payload. UpdateConfig is a full PUT: every field the provider knows about has
// to be carried across, otherwise the API silently drops the ones left out -
// which is invisible in a plan.
func configurationFromWithIndex(config *suggestions.ConfigurationWithIndex) *suggestions.Configuration {
	configuration := suggestions.NewConfiguration(config.GetSourceIndices())
	configuration.Languages = config.Languages
	configuration.Exclude = config.Exclude
	configuration.EnablePersonalization = config.EnablePersonalization
	configuration.AllowSpecialCharacters = config.AllowSpecialCharacters

	return configuration
}

// hydrateQuerySuggestionsModel writes the API response into model. model must
// already hold the prior value (the plan during Create/Update, the state during
// Read), because several attributes are Optional and not Computed: for those the
// planned value is the configuration verbatim, so whether an empty API result
// becomes null or a known empty collection depends on what the prior value was.
// See nullableStringSet for the full contract.
func hydrateQuerySuggestionsModel(resp *suggestions.ConfigurationResponse, model *QuerySuggestionsResourceModel) diag.Diagnostics {
	sourceIndices, diags := flattenSourceIndices(resp.GetSourceIndices(), model.SourceIndices)
	if diags.HasError() {
		return diags
	}

	model.ID = types.StringValue(querySuggestionsResourceID(resp.GetIndexName()))
	model.IndexName = types.StringValue(resp.GetIndexName())
	model.SourceIndices = sourceIndices
	model.Languages = nullableStringSet(model.Languages, languagesFromAPI(resp.GetLanguages()))
	model.Exclude = nullableStringSet(model.Exclude, resp.GetExclude())
	model.EnablePersonalization = types.BoolValue(resp.GetEnablePersonalization())
	model.AllowSpecialCharacters = types.BoolValue(resp.GetAllowSpecialCharacters())

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
		indexNameValue, ok := attrs["index_name"].(types.String)
		if !ok {
			diags.AddError("Invalid source_indices value", fmt.Sprintf("source_indices[%d].index_name is not a string.", i))
			return nil, diags
		}
		source := suggestions.NewSourceIndex(indexNameValue.ValueString())

		if value, ok := attrs["replicas"].(types.Bool); ok && !value.IsNull() && !value.IsUnknown() {
			replicas := value.ValueBool()
			source.Replicas = &replicas
		}

		if value, ok := attrs["analytics_tags"].(types.Set); ok && !value.IsNull() && !value.IsUnknown() {
			tags := setStrings(value)
			sort.Strings(tags)
			source.AnalyticsTags = tags
		}

		if value, ok := attrs["facets"].(types.List); ok && !value.IsNull() && !value.IsUnknown() {
			facets := make([]suggestions.Facet, 0, len(value.Elements()))
			for j, facetValue := range value.Elements() {
				facetObject, ok := facetValue.(types.Object)
				if !ok {
					diags.AddError("Invalid facets value", fmt.Sprintf("source_indices[%d].facets[%d] is not an object.", i, j))
					return nil, diags
				}

				facetAttrs := facetObject.Attributes()
				attributeValue, ok := facetAttrs["attribute"].(types.String)
				if !ok {
					diags.AddError("Invalid facets value", fmt.Sprintf("source_indices[%d].facets[%d].attribute is not a string.", i, j))
					return nil, diags
				}
				amountValue, ok := facetAttrs["amount"].(types.Int64)
				if !ok {
					diags.AddError("Invalid facets value", fmt.Sprintf("source_indices[%d].facets[%d].amount is not a number.", i, j))
					return nil, diags
				}

				facet := suggestions.NewFacet()
				attribute := attributeValue.ValueString()
				facet.Attribute = &attribute
				amount32 := int32(amountValue.ValueInt64())
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
			for j, generateValue := range value.Elements() {
				listValue, ok := generateValue.(types.List)
				if !ok {
					diags.AddError("Invalid generate value", fmt.Sprintf("source_indices[%d].generate[%d] is not a list.", i, j))
					return nil, diags
				}
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

// flattenSourceIndices converts the API's source indices into the Terraform
// list. prior is the planned (Create/Update) or stored (Read) value of the whole
// source_indices list; source_indices is itself a list, so correlating prior and
// response by position is exactly the correlation Terraform uses when comparing
// the planned value against the applied one.
func flattenSourceIndices(sourceIndices []suggestions.SourceIndex, prior types.List) (types.List, diag.Diagnostics) {
	values := make([]attr.Value, 0, len(sourceIndices))
	for i, source := range sourceIndices {
		priorSource := priorSourceIndex(prior, i)

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

		// facets is a block, not an attribute: Terraform represents "no facets
		// blocks" as a known empty list rather than null, so it needs no
		// prior-aware handling.
		facets, diags := types.ListValue(facetModelType, facetValues)
		if diags.HasError() {
			return types.ListNull(sourceIndexModelType), diags
		}

		generate, diags := nullableGenerateList(priorList(priorSource, "generate", generateElementType), source.GetGenerate())
		if diags.HasError() {
			return types.ListNull(sourceIndexModelType), diags
		}

		value, diags := types.ObjectValue(sourceIndexModelAttrTypes, map[string]attr.Value{
			"index_name":     types.StringValue(source.GetIndexName()),
			"replicas":       computedBool(source.Replicas, priorBool(priorSource, "replicas")),
			"analytics_tags": nullableStringSet(priorSet(priorSource, "analytics_tags"), source.GetAnalyticsTags()),
			"facets":         facets,
			"min_hits":       computedInt64(source.MinHits, priorInt64(priorSource, "min_hits")),
			"min_letters":    computedInt64(source.MinLetters, priorInt64(priorSource, "min_letters")),
			"generate":       generate,
			"external":       nullableStringSet(priorSet(priorSource, "external"), source.GetExternal()),
		})
		if diags.HasError() {
			return types.ListNull(sourceIndexModelType), diags
		}
		values = append(values, value)
	}

	return types.ListValue(sourceIndexModelType, values)
}

// priorSourceIndex returns the prior source_indices element at position i, or a
// null object when there is none - which is the case for data source reads and
// imports, where no configuration was ever supplied.
func priorSourceIndex(prior types.List, i int) types.Object {
	if prior.IsNull() || prior.IsUnknown() {
		return types.ObjectNull(sourceIndexModelAttrTypes)
	}

	elements := prior.Elements()
	if i < 0 || i >= len(elements) {
		return types.ObjectNull(sourceIndexModelAttrTypes)
	}

	object, ok := elements[i].(types.Object)
	if !ok {
		return types.ObjectNull(sourceIndexModelAttrTypes)
	}

	return object
}

func priorAttribute(prior types.Object, name string) attr.Value {
	if prior.IsNull() || prior.IsUnknown() {
		return nil
	}

	return prior.Attributes()[name]
}

func priorSet(prior types.Object, name string) types.Set {
	if value, ok := priorAttribute(prior, name).(types.Set); ok {
		return value
	}

	return types.SetNull(types.StringType)
}

func priorList(prior types.Object, name string, elementType attr.Type) types.List {
	if value, ok := priorAttribute(prior, name).(types.List); ok {
		return value
	}

	return types.ListNull(elementType)
}

func priorBool(prior types.Object, name string) types.Bool {
	if value, ok := priorAttribute(prior, name).(types.Bool); ok {
		return value
	}

	return types.BoolNull()
}

func priorInt64(prior types.Object, name string) types.Int64 {
	if value, ok := priorAttribute(prior, name).(types.Int64); ok {
		return value
	}

	return types.Int64Null()
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

// nullableStringSet converts an API string slice into a Terraform set.
// `languages`, `exclude`, `source_indices.analytics_tags` and
// `source_indices.external` are Optional and not Computed, so their planned
// value is the configuration verbatim: emitting a known empty set where the plan
// held null - or null where the plan held `[]` - makes Terraform reject the
// apply with "Provider produced inconsistent result after apply". When the API
// returns nothing, the prior value therefore decides: a null prior stays null,
// while a prior that was explicitly configured as `[]` stays a known empty set.
func nullableStringSet(prior types.Set, values []string) types.Set {
	if len(values) == 0 {
		if prior.IsNull() || prior.IsUnknown() {
			return types.SetNull(types.StringType)
		}

		return types.SetValueMust(types.StringType, []attr.Value{})
	}

	return types.SetValueMust(types.StringType, stringSliceValues(values))
}

// nullableGenerateList converts the API's facet combinations into the Terraform
// list of lists behind `source_indices.generate`. It follows the same
// prior-decides contract as nullableStringSet: `generate` is Optional and not
// Computed, so an omitted attribute plans null and must stay null, while an
// explicit `generate = []` plans a known empty list and must stay one.
func nullableGenerateList(prior types.List, groups [][]string) (types.List, diag.Diagnostics) {
	if len(groups) == 0 {
		if prior.IsNull() || prior.IsUnknown() {
			return types.ListNull(generateElementType), nil
		}

		return types.ListValue(generateElementType, []attr.Value{})
	}

	values := make([]attr.Value, 0, len(groups))
	for _, group := range groups {
		// An inner combination is never null in configuration, so an empty
		// combination round-trips as a known empty list.
		value, diags := types.ListValue(types.StringType, stringSliceValues(group))
		if diags.HasError() {
			return types.ListNull(generateElementType), diags
		}
		values = append(values, value)
	}

	return types.ListValue(generateElementType, values)
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

// computedBool resolves an Optional+Computed bool from the API response. The
// Query Suggestions API omits `replicas` when it has no value to report, so
// falling back to the prior value keeps an already-known plan value from
// regressing to null (which Terraform rejects); an unknown prior - a source
// index Terraform has not applied yet - resolves to null.
func computedBool(value *bool, prior types.Bool) types.Bool {
	if value != nil {
		return types.BoolValue(*value)
	}

	if prior.IsUnknown() {
		return types.BoolNull()
	}

	return prior
}

// computedInt64 is computedBool for `min_hits` and `min_letters`. Both are
// Optional+Computed because the API substitutes its own default when they are
// omitted (min_letters defaults to 4) and reports that default back: modelling
// them as Optional-only made an omitted attribute plan null and then come back
// as the server default, which Terraform rejects with "Provider produced
// inconsistent result after apply".
func computedInt64(value *int32, prior types.Int64) types.Int64 {
	if value != nil {
		return types.Int64Value(int64(*value))
	}

	if prior.IsUnknown() {
		return types.Int64Null()
	}

	return prior
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
