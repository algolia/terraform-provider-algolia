package querysuggestions

import (
	"testing"

	suggestions "github.com/algolia/algoliasearch-client-go/v4/algolia/query-suggestions"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// sourceIndexObject builds a source_indices element, defaulting every attribute
// the caller does not override to null. It mirrors what the framework hands the
// provider as the planned value of a source_indices block.
func sourceIndexObject(t *testing.T, overrides map[string]attr.Value) types.Object {
	t.Helper()

	attrs := map[string]attr.Value{
		"index_name":     types.StringValue("products"),
		"replicas":       types.BoolNull(),
		"analytics_tags": types.SetNull(types.StringType),
		"facets":         types.ListValueMust(facetModelType, []attr.Value{}),
		"min_hits":       types.Int64Null(),
		"min_letters":    types.Int64Null(),
		"generate":       types.ListNull(generateElementType),
		"external":       types.SetNull(types.StringType),
	}
	for name, value := range overrides {
		attrs[name] = value
	}

	object, diags := types.ObjectValue(sourceIndexModelAttrTypes, attrs)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics building source index object: %v", diags)
	}

	return object
}

// sourceIndexAttributes returns the attributes of the single flattened
// source_indices element produced from one API source index.
func sourceIndexAttributes(t *testing.T, source suggestions.SourceIndex, prior types.List) map[string]attr.Value {
	t.Helper()

	list, diags := flattenSourceIndices([]suggestions.SourceIndex{source}, prior)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	elements := list.Elements()
	if len(elements) != 1 {
		t.Fatalf("expected 1 source index, got %#v", elements)
	}

	object, ok := elements[0].(types.Object)
	if !ok {
		t.Fatalf("expected an object element, got %#v", elements[0])
	}

	return object.Attributes()
}

func TestBuildConfigurationWithIndex(t *testing.T) {
	model := QuerySuggestionsResourceModel{
		IndexName: types.StringValue("qs_products"),
		SourceIndices: types.ListValueMust(sourceIndexModelType, []attr.Value{
			sourceIndexObject(t, map[string]attr.Value{
				"analytics_tags": types.SetValueMust(types.StringType, []attr.Value{types.StringValue("mobile")}),
				"facets": types.ListValueMust(facetModelType, []attr.Value{
					types.ObjectValueMust(facetModelAttrTypes, map[string]attr.Value{
						"attribute": types.StringValue("brand"),
						"amount":    types.Int64Value(5),
					}),
				}),
				"min_hits":    types.Int64Value(10),
				"min_letters": types.Int64Value(3),
				"generate": types.ListValueMust(generateElementType, []attr.Value{
					types.ListValueMust(types.StringType, []attr.Value{types.StringValue("brand"), types.StringValue("category")}),
				}),
				"external": types.SetValueMust(types.StringType, []attr.Value{types.StringValue("external_products")}),
			}),
		}),
		Languages: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("en")}),
		Exclude:   types.SetValueMust(types.StringType, []attr.Value{types.StringValue("free")}),
	}

	config, diags := buildConfigurationWithIndex(&model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := config.GetIndexName(); got != "qs_products" {
		t.Fatalf("index_name = %q, want %q", got, "qs_products")
	}
	if got := config.GetSourceIndices(); len(got) != 1 {
		t.Fatalf("source_indices = %#v, want 1 item", got)
	}
	if got := config.GetExclude(); len(got) != 1 || got[0] != "free" {
		t.Fatalf("exclude = %#v, want [free]", got)
	}
}

func TestHydrateQuerySuggestionsModel(t *testing.T) {
	resp := suggestions.NewConfigurationResponse(
		"app",
		"qs_products",
		[]suggestions.SourceIndex{
			*suggestions.NewSourceIndex(
				"products",
				suggestions.WithSourceIndexAnalyticsTags([]string{"mobile"}),
				suggestions.WithSourceIndexMinHits(10),
				suggestions.WithSourceIndexMinLetters(3),
				suggestions.WithSourceIndexFacets([]suggestions.Facet{
					*suggestions.NewFacet(
						suggestions.WithFacetAttribute("brand"),
						suggestions.WithFacetAmount(5),
					),
				}),
				suggestions.WithSourceIndexGenerate([][]string{{"brand", "category"}}),
				suggestions.WithSourceIndexExternal([]string{"external_products"}),
			),
		},
		*suggestions.ArrayOfStringAsLanguages([]string{"en"}),
		[]string{"free"},
		false,
		false,
	)

	model := QuerySuggestionsResourceModel{}
	diags := hydrateQuerySuggestionsModel(resp, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.ID.ValueString(); got != "qs_products" {
		t.Fatalf("id = %q, want query suggestions index name", got)
	}
	if got := model.IndexName.ValueString(); got != "qs_products" {
		t.Fatalf("index_name = %q, want qs_products", got)
	}
}

func TestQuerySuggestionsSchemas_DoNotExposeRegion(t *testing.T) {
	resourceSchema := querySuggestionsResourceSchema()
	if _, ok := resourceSchema.Attributes["region"]; ok {
		t.Fatal("expected query suggestions resource schema to omit region")
	}

	dataSourceSchema := querySuggestionsDataSourceSchema()
	if _, ok := dataSourceSchema.Attributes["region"]; ok {
		t.Fatal("expected query suggestions data source schema to omit region")
	}
}

// TestFlattenSourceIndices_GeneratePriorDecidesEmptyResult covers the three
// cases of the prior-decides contract for source_indices.generate, which is
// Optional and not Computed: omitting it must keep null, an explicit `[]` must
// keep a known empty list, and a non-empty API result must win over both.
func TestFlattenSourceIndices_GeneratePriorDecidesEmptyResult(t *testing.T) {
	nullPrior := types.ListValueMust(sourceIndexModelType, []attr.Value{sourceIndexObject(t, nil)})
	emptyPrior := types.ListValueMust(sourceIndexModelType, []attr.Value{sourceIndexObject(t, map[string]attr.Value{
		"generate": types.ListValueMust(generateElementType, []attr.Value{}),
	})})

	t.Run("api empty and prior null stays null", func(t *testing.T) {
		attrs := sourceIndexAttributes(t, *suggestions.NewSourceIndex("products"), nullPrior)
		generate, ok := attrs["generate"].(types.List)
		if !ok {
			t.Fatalf("generate = %#v, want a list", attrs["generate"])
		}
		if !generate.IsNull() {
			t.Fatalf("generate = %#v, want null", generate)
		}
	})

	t.Run("api empty and prior known empty stays known empty", func(t *testing.T) {
		attrs := sourceIndexAttributes(t, *suggestions.NewSourceIndex("products"), emptyPrior)
		generate, ok := attrs["generate"].(types.List)
		if !ok {
			t.Fatalf("generate = %#v, want a list", attrs["generate"])
		}
		if generate.IsNull() || generate.IsUnknown() {
			t.Fatalf("generate = %#v, want a known empty list", generate)
		}
		if len(generate.Elements()) != 0 {
			t.Fatalf("generate = %#v, want no elements", generate.Elements())
		}
	})

	t.Run("api non-empty wins over a null prior", func(t *testing.T) {
		source := *suggestions.NewSourceIndex("products", suggestions.WithSourceIndexGenerate([][]string{
			{"brand", "category"},
			{"color"},
		}))
		attrs := sourceIndexAttributes(t, source, nullPrior)
		generate, ok := attrs["generate"].(types.List)
		if !ok {
			t.Fatalf("generate = %#v, want a list", attrs["generate"])
		}
		if len(generate.Elements()) != 2 {
			t.Fatalf("generate = %#v, want 2 combinations", generate.Elements())
		}

		first, ok := generate.Elements()[0].(types.List)
		if !ok {
			t.Fatalf("generate[0] = %#v, want a list", generate.Elements()[0])
		}
		if len(first.Elements()) != 2 || first.Elements()[0].(types.String).ValueString() != "brand" {
			t.Fatalf("generate[0] = %#v, want [brand category]", first.Elements())
		}
	})
}

// TestFlattenSourceIndices_StringSetsPriorDecidesEmptyResult covers the same
// three cases for the nested Optional, non-Computed string sets.
func TestFlattenSourceIndices_StringSetsPriorDecidesEmptyResult(t *testing.T) {
	for _, name := range []string{"analytics_tags", "external"} {
		t.Run(name, func(t *testing.T) {
			nullPrior := types.ListValueMust(sourceIndexModelType, []attr.Value{sourceIndexObject(t, nil)})
			emptyPrior := types.ListValueMust(sourceIndexModelType, []attr.Value{sourceIndexObject(t, map[string]attr.Value{
				name: types.SetValueMust(types.StringType, []attr.Value{}),
			})})

			attrs := sourceIndexAttributes(t, *suggestions.NewSourceIndex("products"), nullPrior)
			if value := attrs[name].(types.Set); !value.IsNull() {
				t.Fatalf("%s with a null prior = %#v, want null", name, value)
			}

			attrs = sourceIndexAttributes(t, *suggestions.NewSourceIndex("products"), emptyPrior)
			value := attrs[name].(types.Set)
			if value.IsNull() || value.IsUnknown() {
				t.Fatalf("%s with an explicitly empty prior = %#v, want a known empty set", name, value)
			}
			if len(value.Elements()) != 0 {
				t.Fatalf("%s = %#v, want no elements", name, value.Elements())
			}

			source := *suggestions.NewSourceIndex("products",
				suggestions.WithSourceIndexAnalyticsTags([]string{"mobile"}),
				suggestions.WithSourceIndexExternal([]string{"external_products"}),
			)
			attrs = sourceIndexAttributes(t, source, nullPrior)
			value = attrs[name].(types.Set)
			if len(value.Elements()) != 1 {
				t.Fatalf("%s = %#v, want 1 element", name, value.Elements())
			}
		})
	}
}

// TestHydrateQuerySuggestionsModel_TopLevelSetsPriorDecidesEmptyResult covers
// the mirror direction: `languages` and `exclude` are Optional and not
// Computed, so an explicit `[]` must survive an empty API response.
func TestHydrateQuerySuggestionsModel_TopLevelSetsPriorDecidesEmptyResult(t *testing.T) {
	response := func(languages []string, exclude []string) *suggestions.ConfigurationResponse {
		return suggestions.NewConfigurationResponse(
			"app",
			"qs_products",
			[]suggestions.SourceIndex{*suggestions.NewSourceIndex("products")},
			*suggestions.ArrayOfStringAsLanguages(languages),
			exclude,
			false,
			false,
		)
	}

	t.Run("api empty and prior null stays null", func(t *testing.T) {
		model := QuerySuggestionsResourceModel{
			Languages: types.SetNull(types.StringType),
			Exclude:   types.SetNull(types.StringType),
		}
		if diags := hydrateQuerySuggestionsModel(response(nil, nil), &model); diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if !model.Languages.IsNull() {
			t.Fatalf("languages = %#v, want null", model.Languages)
		}
		if !model.Exclude.IsNull() {
			t.Fatalf("exclude = %#v, want null", model.Exclude)
		}
	})

	t.Run("api empty and prior known empty stays known empty", func(t *testing.T) {
		model := QuerySuggestionsResourceModel{
			Languages: types.SetValueMust(types.StringType, []attr.Value{}),
			Exclude:   types.SetValueMust(types.StringType, []attr.Value{}),
		}
		if diags := hydrateQuerySuggestionsModel(response(nil, nil), &model); diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if model.Languages.IsNull() || len(model.Languages.Elements()) != 0 {
			t.Fatalf("languages = %#v, want a known empty set", model.Languages)
		}
		if model.Exclude.IsNull() || len(model.Exclude.Elements()) != 0 {
			t.Fatalf("exclude = %#v, want a known empty set", model.Exclude)
		}
	})

	t.Run("api non-empty wins over a null prior", func(t *testing.T) {
		model := QuerySuggestionsResourceModel{
			Languages: types.SetNull(types.StringType),
			Exclude:   types.SetNull(types.StringType),
		}
		if diags := hydrateQuerySuggestionsModel(response([]string{"en"}, []string{"free"}), &model); diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if len(model.Languages.Elements()) != 1 {
			t.Fatalf("languages = %#v, want 1 element", model.Languages.Elements())
		}
		if len(model.Exclude.Elements()) != 1 {
			t.Fatalf("exclude = %#v, want 1 element", model.Exclude.Elements())
		}
	})
}

// TestBuildConfigurationWithIndex_ExplicitlyEmptySetsAreSent checks that an
// explicit `exclude = []` reaches the API as an empty list rather than being
// dropped, which is what makes the known-empty prior above round-trip.
func TestBuildConfigurationWithIndex_ExplicitlyEmptySetsAreSent(t *testing.T) {
	model := QuerySuggestionsResourceModel{
		IndexName:     types.StringValue("qs_products"),
		SourceIndices: types.ListValueMust(sourceIndexModelType, []attr.Value{sourceIndexObject(t, nil)}),
		Exclude:       types.SetValueMust(types.StringType, []attr.Value{}),
	}

	config, diags := buildConfigurationWithIndex(&model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !config.HasExclude() {
		t.Fatal("expected an explicitly empty exclude to be sent to the API")
	}
	if got := config.GetExclude(); len(got) != 0 {
		t.Fatalf("exclude = %#v, want an empty list", got)
	}
}

// TestBuildConfigurationWithIndex_RoundTripsFullReplacementFields checks the
// expand direction of the fields UpdateConfig would otherwise wipe, since it is
// a full PUT.
func TestBuildConfigurationWithIndex_RoundTripsFullReplacementFields(t *testing.T) {
	model := QuerySuggestionsResourceModel{
		IndexName: types.StringValue("qs_products"),
		SourceIndices: types.ListValueMust(sourceIndexModelType, []attr.Value{
			sourceIndexObject(t, map[string]attr.Value{"replicas": types.BoolValue(true)}),
		}),
		EnablePersonalization:  types.BoolValue(true),
		AllowSpecialCharacters: types.BoolValue(true),
	}

	config, diags := buildConfigurationWithIndex(&model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !config.HasEnablePersonalization() || !config.GetEnablePersonalization() {
		t.Fatalf("enable_personalization = %#v, want true", config.EnablePersonalization)
	}
	if !config.HasAllowSpecialCharacters() || !config.GetAllowSpecialCharacters() {
		t.Fatalf("allow_special_characters = %#v, want true", config.AllowSpecialCharacters)
	}

	sources := config.GetSourceIndices()
	if len(sources) != 1 || !sources[0].HasReplicas() || !sources[0].GetReplicas() {
		t.Fatalf("source_indices[0].replicas = %#v, want true", sources[0].Replicas)
	}
}

// TestBuildConfigurationWithIndex_UnsetFullReplacementFieldsAreOmitted checks
// that an unset Optional+Computed field is left out of the payload so the API
// keeps whatever it already has, rather than being forced to false.
func TestBuildConfigurationWithIndex_UnsetFullReplacementFieldsAreOmitted(t *testing.T) {
	model := QuerySuggestionsResourceModel{
		IndexName:              types.StringValue("qs_products"),
		SourceIndices:          types.ListValueMust(sourceIndexModelType, []attr.Value{sourceIndexObject(t, nil)}),
		EnablePersonalization:  types.BoolUnknown(),
		AllowSpecialCharacters: types.BoolNull(),
	}

	config, diags := buildConfigurationWithIndex(&model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if config.HasEnablePersonalization() {
		t.Fatalf("enable_personalization = %#v, want omitted", config.EnablePersonalization)
	}
	if config.HasAllowSpecialCharacters() {
		t.Fatalf("allow_special_characters = %#v, want omitted", config.AllowSpecialCharacters)
	}
	if config.GetSourceIndices()[0].HasReplicas() {
		t.Fatalf("source_indices[0].replicas = %#v, want omitted", config.GetSourceIndices()[0].Replicas)
	}
}

// TestConfigurationFromWithIndex_CarriesEveryField is the regression test for
// Update dropping fields: UpdateConfig is a full PUT, so the update payload has
// to carry everything the create payload carried.
func TestConfigurationFromWithIndex_CarriesEveryField(t *testing.T) {
	withIndex := suggestions.NewConfigurationWithIndex(
		[]suggestions.SourceIndex{*suggestions.NewSourceIndex("products", suggestions.WithSourceIndexReplicas(true))},
		"qs_products",
		suggestions.WithConfigurationWithIndexLanguages(*suggestions.ArrayOfStringAsLanguages([]string{"en"})),
		suggestions.WithConfigurationWithIndexExclude([]string{"free"}),
		suggestions.WithConfigurationWithIndexEnablePersonalization(true),
		suggestions.WithConfigurationWithIndexAllowSpecialCharacters(true),
	)

	configuration := configurationFromWithIndex(withIndex)

	if !configuration.HasEnablePersonalization() || !configuration.GetEnablePersonalization() {
		t.Fatalf("enable_personalization = %#v, want true", configuration.EnablePersonalization)
	}
	if !configuration.HasAllowSpecialCharacters() || !configuration.GetAllowSpecialCharacters() {
		t.Fatalf("allow_special_characters = %#v, want true", configuration.AllowSpecialCharacters)
	}
	if !configuration.HasLanguages() {
		t.Fatal("expected languages to be carried into the update payload")
	}
	if got := configuration.GetExclude(); len(got) != 1 || got[0] != "free" {
		t.Fatalf("exclude = %#v, want [free]", got)
	}

	sources := configuration.GetSourceIndices()
	if len(sources) != 1 || !sources[0].HasReplicas() || !sources[0].GetReplicas() {
		t.Fatalf("source_indices[0].replicas = %#v, want true", sources[0].Replicas)
	}
}

// TestHydrateQuerySuggestionsModel_ReadsFullReplacementFields checks the flatten
// direction of the same fields.
func TestHydrateQuerySuggestionsModel_ReadsFullReplacementFields(t *testing.T) {
	resp := suggestions.NewConfigurationResponse(
		"app",
		"qs_products",
		[]suggestions.SourceIndex{*suggestions.NewSourceIndex("products", suggestions.WithSourceIndexReplicas(true))},
		*suggestions.ArrayOfStringAsLanguages(nil),
		nil,
		true,
		true,
	)

	var model QuerySuggestionsResourceModel
	if diags := hydrateQuerySuggestionsModel(resp, &model); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.EnablePersonalization.IsNull() || !model.EnablePersonalization.ValueBool() {
		t.Fatalf("enable_personalization = %#v, want true", model.EnablePersonalization)
	}
	if model.AllowSpecialCharacters.IsNull() || !model.AllowSpecialCharacters.ValueBool() {
		t.Fatalf("allow_special_characters = %#v, want true", model.AllowSpecialCharacters)
	}

	attrs := model.SourceIndices.Elements()[0].(types.Object).Attributes()
	if replicas := attrs["replicas"].(types.Bool); replicas.IsNull() || !replicas.ValueBool() {
		t.Fatalf("source_indices[0].replicas = %#v, want true", replicas)
	}
}

// TestFlattenSourceIndices_ReplicasFallsBackToPrior guards the Optional+Computed
// contract for `replicas`: once the plan holds a known value, an API response
// that omits the field must not regress it to null.
func TestFlattenSourceIndices_ReplicasFallsBackToPrior(t *testing.T) {
	knownPrior := types.ListValueMust(sourceIndexModelType, []attr.Value{
		sourceIndexObject(t, map[string]attr.Value{"replicas": types.BoolValue(false)}),
	})
	attrs := sourceIndexAttributes(t, *suggestions.NewSourceIndex("products"), knownPrior)
	replicas := attrs["replicas"].(types.Bool)
	if replicas.IsNull() || replicas.ValueBool() {
		t.Fatalf("replicas = %#v, want the prior value false", replicas)
	}

	unknownPrior := types.ListValueMust(sourceIndexModelType, []attr.Value{
		sourceIndexObject(t, map[string]attr.Value{"replicas": types.BoolUnknown()}),
	})
	attrs = sourceIndexAttributes(t, *suggestions.NewSourceIndex("products"), unknownPrior)
	if replicas := attrs["replicas"].(types.Bool); !replicas.IsNull() {
		t.Fatalf("replicas = %#v, want null when the plan was unknown", replicas)
	}
}

// TestExpandSourceIndices_RejectsMistypedAttributes checks that a model that
// does not match the schema produces a diagnostic instead of panicking.
func TestExpandSourceIndices_RejectsMistypedAttributes(t *testing.T) {
	mistyped := types.ListValueMust(types.ObjectType{AttrTypes: map[string]attr.Type{"index_name": types.Int64Type}}, []attr.Value{
		types.ObjectValueMust(map[string]attr.Type{"index_name": types.Int64Type}, map[string]attr.Value{
			"index_name": types.Int64Value(1),
		}),
	})

	if _, diags := expandSourceIndices(mistyped); !diags.HasError() {
		t.Fatal("expected a diagnostic for a mistyped index_name, got none")
	}
}
