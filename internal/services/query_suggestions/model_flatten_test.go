package query_suggestions

import (
	"context"
	"testing"

	suggestions "github.com/algolia/algoliasearch-client-go/v4/algolia/query-suggestions"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---- flattenConfigurationResponse ----

func TestFlattenConfigurationResponse_basic(t *testing.T) {
	ctx := context.Background()

	langCodes := []string{"en", "fr"}
	resp := &suggestions.ConfigurationResponse{
		IndexName: "my_suggestions",
		SourceIndices: []suggestions.SourceIndex{
			*suggestions.NewSourceIndex("products"),
		},
		Languages:              suggestions.Languages{ArrayOfString: &langCodes},
		Exclude:                []string{"bad_query"},
		EnablePersonalization:  true,
		AllowSpecialCharacters: false,
	}

	model := &QuerySuggestionsConfigResourceModel{}
	diags := flattenConfigurationResponse(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if model.IndexName.ValueString() != "my_suggestions" {
		t.Errorf("expected index_name 'my_suggestions', got %q", model.IndexName.ValueString())
	}
	if model.EnablePersonalization.ValueBool() != true {
		t.Error("expected enable_personalization=true")
	}
	if model.AllowSpecialCharacters.ValueBool() != false {
		t.Error("expected allow_special_characters=false")
	}

	// Languages should be a list of 2 codes, LanguagesEnabled should be null.
	if model.Languages.IsNull() {
		t.Fatal("expected non-null languages list")
	}
	var langs []string
	diags = model.Languages.ElementsAs(ctx, &langs, false)
	if diags.HasError() {
		t.Fatalf("reading languages: %v", diags.Errors())
	}
	if len(langs) != 2 || langs[0] != "en" || langs[1] != "fr" {
		t.Errorf("unexpected languages: %v", langs)
	}
	if !model.LanguagesEnabled.IsNull() {
		t.Errorf("expected null languages_enabled, got %v", model.LanguagesEnabled.ValueBool())
	}

	// Exclude list
	if model.Exclude.IsNull() {
		t.Fatal("expected non-null exclude list")
	}
	var exclude []string
	diags = model.Exclude.ElementsAs(ctx, &exclude, false)
	if diags.HasError() {
		t.Fatalf("reading exclude: %v", diags.Errors())
	}
	if len(exclude) != 1 || exclude[0] != "bad_query" {
		t.Errorf("unexpected exclude: %v", exclude)
	}

	// Source indices
	if model.SourceIndices.IsNull() {
		t.Fatal("expected non-null source_indices")
	}
	var sourceModels []SourceIndexModel
	diags = model.SourceIndices.ElementsAs(ctx, &sourceModels, false)
	if diags.HasError() {
		t.Fatalf("reading source indices: %v", diags.Errors())
	}
	if len(sourceModels) != 1 || sourceModels[0].IndexName.ValueString() != "products" {
		t.Errorf("unexpected source indices: %v", sourceModels)
	}
}

func TestFlattenConfigurationResponse_languagesAsBoolTrue(t *testing.T) {
	ctx := context.Background()

	boolTrue := true
	resp := &suggestions.ConfigurationResponse{
		IndexName:     "idx",
		SourceIndices: []suggestions.SourceIndex{},
		Languages:     suggestions.Languages{Bool: &boolTrue},
	}

	model := &QuerySuggestionsConfigResourceModel{}
	diags := flattenConfigurationResponse(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if model.LanguagesEnabled.IsNull() || !model.LanguagesEnabled.ValueBool() {
		t.Error("expected languages_enabled=true")
	}
	// languages list should be empty (not null) when bool mode
	if model.Languages.IsNull() {
		t.Error("expected non-null (empty) languages list in bool mode")
	}
}

func TestFlattenConfigurationResponse_languagesAsBoolFalse(t *testing.T) {
	ctx := context.Background()

	boolFalse := false
	resp := &suggestions.ConfigurationResponse{
		IndexName:     "idx",
		SourceIndices: []suggestions.SourceIndex{},
		Languages:     suggestions.Languages{Bool: &boolFalse},
	}

	model := &QuerySuggestionsConfigResourceModel{}
	diags := flattenConfigurationResponse(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if model.LanguagesEnabled.IsNull() || model.LanguagesEnabled.ValueBool() {
		t.Error("expected languages_enabled=false")
	}
}

func TestFlattenConfigurationResponse_noLanguages(t *testing.T) {
	ctx := context.Background()

	resp := &suggestions.ConfigurationResponse{
		IndexName:     "idx",
		SourceIndices: []suggestions.SourceIndex{},
		Languages:     suggestions.Languages{}, // neither field set
	}

	model := &QuerySuggestionsConfigResourceModel{}
	diags := flattenConfigurationResponse(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if !model.Languages.IsNull() && model.Languages.Elements() != nil {
		var langs []string
		diags = model.Languages.ElementsAs(ctx, &langs, false)
		if diags.HasError() {
			t.Fatalf("reading languages: %v", diags.Errors())
		}
		if len(langs) != 0 {
			t.Errorf("expected empty languages, got %v", langs)
		}
	}
	if !model.LanguagesEnabled.IsNull() {
		t.Errorf("expected null languages_enabled, got %v", model.LanguagesEnabled)
	}
}

func TestFlattenConfigurationResponse_emptySourceIndices(t *testing.T) {
	ctx := context.Background()

	resp := &suggestions.ConfigurationResponse{
		IndexName:     "idx",
		SourceIndices: []suggestions.SourceIndex{},
	}

	model := &QuerySuggestionsConfigResourceModel{}
	diags := flattenConfigurationResponse(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if model.SourceIndices.IsNull() {
		t.Fatal("expected non-null (empty) source indices list")
	}
	var sourceModels []SourceIndexModel
	diags = model.SourceIndices.ElementsAs(ctx, &sourceModels, false)
	if diags.HasError() {
		t.Fatalf("reading source indices: %v", diags.Errors())
	}
	if len(sourceModels) != 0 {
		t.Errorf("expected 0 source indices, got %d", len(sourceModels))
	}
}

func TestFlattenConfigurationResponse_sourceIndexWithFacetsAndGenerate(t *testing.T) {
	ctx := context.Background()

	attr := "brand"
	amount := int32(3)
	replicas := true
	minHits := int32(10)
	minLetters := int32(3)

	resp := &suggestions.ConfigurationResponse{
		IndexName: "idx",
		SourceIndices: []suggestions.SourceIndex{
			{
				IndexName:     "products",
				Replicas:      &replicas,
				AnalyticsTags: []string{"tag1", "tag2"},
				Facets:        []suggestions.Facet{{Attribute: &attr, Amount: &amount}},
				MinHits:       &minHits,
				MinLetters:    &minLetters,
				Generate:      [][]string{{"brand"}, {"category", "brand"}},
				External:      []string{"ext"},
			},
		},
	}

	model := &QuerySuggestionsConfigResourceModel{}
	diags := flattenConfigurationResponse(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	var sourceModels []SourceIndexModel
	diags = model.SourceIndices.ElementsAs(ctx, &sourceModels, false)
	if diags.HasError() {
		t.Fatalf("reading source indices: %v", diags.Errors())
	}
	if len(sourceModels) != 1 {
		t.Fatalf("expected 1 source index, got %d", len(sourceModels))
	}
	si := sourceModels[0]

	if si.IndexName.ValueString() != "products" {
		t.Errorf("expected index_name 'products', got %q", si.IndexName.ValueString())
	}
	if !si.Replicas.ValueBool() {
		t.Error("expected replicas=true")
	}
	if si.MinHits.ValueInt64() != 10 {
		t.Errorf("expected min_hits=10, got %d", si.MinHits.ValueInt64())
	}
	if si.MinLetters.ValueInt64() != 3 {
		t.Errorf("expected min_letters=3, got %d", si.MinLetters.ValueInt64())
	}

	var tags []string
	diags = si.AnalyticsTags.ElementsAs(ctx, &tags, false)
	if diags.HasError() {
		t.Fatalf("reading analytics_tags: %v", diags.Errors())
	}
	if len(tags) != 2 || tags[0] != "tag1" || tags[1] != "tag2" {
		t.Errorf("unexpected analytics_tags: %v", tags)
	}

	var facets []FacetModel
	diags = si.Facets.ElementsAs(ctx, &facets, false)
	if diags.HasError() {
		t.Fatalf("reading facets: %v", diags.Errors())
	}
	if len(facets) != 1 {
		t.Fatalf("expected 1 facet, got %d", len(facets))
	}
	if facets[0].Attribute.ValueString() != "brand" {
		t.Errorf("expected facet attribute 'brand', got %q", facets[0].Attribute.ValueString())
	}
	if facets[0].Amount.ValueInt64() != 3 {
		t.Errorf("expected facet amount=3, got %d", facets[0].Amount.ValueInt64())
	}

	// generate is JSON-encoded
	if si.Generate.IsNull() {
		t.Fatal("expected non-null generate")
	}
	if si.Generate.ValueString() != `[["brand"],["category","brand"]]` {
		t.Errorf("unexpected generate JSON: %q", si.Generate.ValueString())
	}

	var external []string
	diags = si.External.ElementsAs(ctx, &external, false)
	if diags.HasError() {
		t.Fatalf("reading external: %v", diags.Errors())
	}
	if len(external) != 1 || external[0] != "ext" {
		t.Errorf("unexpected external: %v", external)
	}
}

// ---- round-trip: expand → flatten ----

func TestExpandFlattenRoundTrip(t *testing.T) {
	ctx := context.Background()

	sourceList := buildSourceIndexList(t, SourceIndexModel{
		IndexName:     types.StringValue("products"),
		Replicas:      types.BoolValue(true),
		AnalyticsTags: emptyStringList(),
		Facets:        buildFacetList(t, FacetModel{Attribute: types.StringValue("color"), Amount: types.Int64Value(2)}),
		MinHits:       types.Int64Value(3),
		MinLetters:    types.Int64Value(2),
		Generate:      types.StringNull(),
		External:      emptyStringList(),
	})

	original := &QuerySuggestionsConfigResourceModel{
		IndexName:              types.StringValue("roundtrip_idx"),
		Region:                 types.StringValue("eu"),
		SourceIndices:          sourceList,
		Languages:              types.ListNull(types.StringType), // null so LanguagesEnabled takes effect
		LanguagesEnabled:       types.BoolValue(true),
		Exclude:                types.ListNull(types.StringType),
		EnablePersonalization:  types.BoolValue(false),
		AllowSpecialCharacters: types.BoolValue(true),
	}

	// Expand to API type
	cfg, diags := expandConfigurationWithIndex(ctx, original)
	if diags.HasError() {
		t.Fatalf("expand: %v", diags.Errors())
	}

	// Simulate API response from the expanded value
	apiResp := &suggestions.ConfigurationResponse{
		IndexName:              cfg.IndexName,
		SourceIndices:          cfg.SourceIndices,
		Languages:              cfg.GetLanguages(),
		Exclude:                cfg.Exclude,
		EnablePersonalization:  cfg.GetEnablePersonalization(),
		AllowSpecialCharacters: cfg.GetAllowSpecialCharacters(),
	}

	// Flatten back
	roundtripped := &QuerySuggestionsConfigResourceModel{}
	diags = flattenConfigurationResponse(ctx, apiResp, roundtripped)
	if diags.HasError() {
		t.Fatalf("flatten: %v", diags.Errors())
	}

	if roundtripped.IndexName.ValueString() != "roundtrip_idx" {
		t.Errorf("index_name mismatch: %q", roundtripped.IndexName.ValueString())
	}
	if roundtripped.EnablePersonalization.ValueBool() != false {
		t.Error("expected enable_personalization=false after round-trip")
	}
	if roundtripped.AllowSpecialCharacters.ValueBool() != true {
		t.Error("expected allow_special_characters=true after round-trip")
	}
	if roundtripped.LanguagesEnabled.IsNull() || !roundtripped.LanguagesEnabled.ValueBool() {
		t.Error("expected languages_enabled=true after round-trip")
	}

	var sourceModels []SourceIndexModel
	diags = roundtripped.SourceIndices.ElementsAs(ctx, &sourceModels, false)
	if diags.HasError() {
		t.Fatalf("reading source indices: %v", diags.Errors())
	}
	if len(sourceModels) != 1 {
		t.Fatalf("expected 1 source index, got %d", len(sourceModels))
	}
	if sourceModels[0].IndexName.ValueString() != "products" {
		t.Errorf("source index_name mismatch: %q", sourceModels[0].IndexName.ValueString())
	}
	if !sourceModels[0].Replicas.ValueBool() {
		t.Error("expected replicas=true after round-trip")
	}
	if sourceModels[0].MinHits.ValueInt64() != 3 {
		t.Errorf("expected min_hits=3, got %d", sourceModels[0].MinHits.ValueInt64())
	}
}
