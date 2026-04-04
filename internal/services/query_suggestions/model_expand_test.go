package query_suggestions

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// helpers — build a null source-index list and an empty facet list.

func nullSourceIndexList() types.List {
	return types.ListNull(types.ObjectType{AttrTypes: sourceIndexAttrTypes()})
}

func emptyFacetList() types.List {
	return types.ListValueMust(types.ObjectType{AttrTypes: facetAttrTypes()}, []attr.Value{})
}

func emptyStringList() types.List {
	return types.ListValueMust(types.StringType, []attr.Value{})
}

// buildSourceIndexList wraps one or more SourceIndexModel values into a types.List.
func buildSourceIndexList(t *testing.T, models ...SourceIndexModel) types.List {
	t.Helper()
	list, diags := types.ListValueFrom(
		context.Background(),
		types.ObjectType{AttrTypes: sourceIndexAttrTypes()},
		models,
	)
	if diags.HasError() {
		t.Fatalf("buildSourceIndexList: %v", diags.Errors())
	}
	return list
}

// buildFacetList wraps one or more FacetModel values into a types.List.
func buildFacetList(t *testing.T, models ...FacetModel) types.List {
	t.Helper()
	list, diags := types.ListValueFrom(
		context.Background(),
		types.ObjectType{AttrTypes: facetAttrTypes()},
		models,
	)
	if diags.HasError() {
		t.Fatalf("buildFacetList: %v", diags.Errors())
	}
	return list
}

// ---- expandConfigurationWithIndex ----

func TestExpandConfigurationWithIndex_basic(t *testing.T) {
	ctx := context.Background()

	sourceList := buildSourceIndexList(t, SourceIndexModel{
		IndexName:     types.StringValue("products"),
		Replicas:      types.BoolValue(true),
		AnalyticsTags: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("prod")}),
		Facets:        buildFacetList(t, FacetModel{Attribute: types.StringValue("brand"), Amount: types.Int64Value(1)}),
		MinHits:       types.Int64Value(5),
		MinLetters:    types.Int64Value(4),
		Generate:      types.StringValue(`[["brand"],["category","brand"]]`),
		External:      types.ListValueMust(types.StringType, []attr.Value{types.StringValue("ext_idx")}),
	})

	model := &QuerySuggestionsConfigResourceModel{
		IndexName:              types.StringValue("my_suggestions"),
		Region:                 types.StringValue("us"),
		SourceIndices:          sourceList,
		Languages:              types.ListValueMust(types.StringType, []attr.Value{types.StringValue("en"), types.StringValue("fr")}),
		LanguagesEnabled:       types.BoolNull(),
		Exclude:                types.ListValueMust(types.StringType, []attr.Value{types.StringValue("stop")}),
		EnablePersonalization:  types.BoolValue(true),
		AllowSpecialCharacters: types.BoolValue(false),
	}

	cfg, diags := expandConfigurationWithIndex(ctx, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if cfg.IndexName != "my_suggestions" {
		t.Errorf("expected index_name 'my_suggestions', got %q", cfg.IndexName)
	}

	if len(cfg.SourceIndices) != 1 {
		t.Fatalf("expected 1 source index, got %d", len(cfg.SourceIndices))
	}
	si := cfg.SourceIndices[0]
	if si.IndexName != "products" {
		t.Errorf("expected source index_name 'products', got %q", si.IndexName)
	}
	if si.Replicas == nil || !*si.Replicas {
		t.Error("expected replicas=true")
	}
	if si.MinHits == nil || *si.MinHits != 5 {
		t.Errorf("expected min_hits=5, got %v", si.MinHits)
	}
	if si.MinLetters == nil || *si.MinLetters != 4 {
		t.Errorf("expected min_letters=4, got %v", si.MinLetters)
	}
	if len(si.AnalyticsTags) != 1 || si.AnalyticsTags[0] != "prod" {
		t.Errorf("unexpected analytics_tags: %v", si.AnalyticsTags)
	}
	if len(si.Facets) != 1 {
		t.Fatalf("expected 1 facet, got %d", len(si.Facets))
	}
	if si.Facets[0].Attribute == nil || *si.Facets[0].Attribute != "brand" {
		t.Errorf("expected facet attribute 'brand', got %v", si.Facets[0].Attribute)
	}
	if si.Facets[0].Amount == nil || *si.Facets[0].Amount != 1 {
		t.Errorf("expected facet amount=1, got %v", si.Facets[0].Amount)
	}
	if len(si.Generate) != 2 {
		t.Errorf("expected 2 generate groups, got %v", si.Generate)
	}
	if len(si.External) != 1 || si.External[0] != "ext_idx" {
		t.Errorf("unexpected external: %v", si.External)
	}

	langs := cfg.GetLanguages()
	if langs.ArrayOfString == nil || len(*langs.ArrayOfString) != 2 {
		t.Errorf("expected 2 language codes, got %v", langs)
	}

	if len(cfg.Exclude) != 1 || cfg.Exclude[0] != "stop" {
		t.Errorf("unexpected exclude: %v", cfg.Exclude)
	}
	if cfg.EnablePersonalization == nil || !*cfg.EnablePersonalization {
		t.Error("expected enable_personalization=true")
	}
	if cfg.AllowSpecialCharacters == nil || *cfg.AllowSpecialCharacters {
		t.Error("expected allow_special_characters=false")
	}
}

func TestExpandConfigurationWithIndex_minimalRequired(t *testing.T) {
	ctx := context.Background()

	sourceList := buildSourceIndexList(t, SourceIndexModel{
		IndexName:     types.StringValue("logs"),
		Replicas:      types.BoolNull(),
		AnalyticsTags: emptyStringList(),
		Facets:        emptyFacetList(),
		MinHits:       types.Int64Null(),
		MinLetters:    types.Int64Null(),
		Generate:      types.StringNull(),
		External:      emptyStringList(),
	})

	model := &QuerySuggestionsConfigResourceModel{
		IndexName:              types.StringValue("suggestions_idx"),
		Region:                 types.StringValue("eu"),
		SourceIndices:          sourceList,
		Languages:              types.ListNull(types.StringType),
		LanguagesEnabled:       types.BoolNull(),
		Exclude:                types.ListNull(types.StringType),
		EnablePersonalization:  types.BoolNull(),
		AllowSpecialCharacters: types.BoolNull(),
	}

	cfg, diags := expandConfigurationWithIndex(ctx, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if cfg.IndexName != "suggestions_idx" {
		t.Errorf("expected index_name 'suggestions_idx', got %q", cfg.IndexName)
	}
	if len(cfg.SourceIndices) != 1 {
		t.Fatalf("expected 1 source index, got %d", len(cfg.SourceIndices))
	}
	si := cfg.SourceIndices[0]
	if si.Replicas != nil {
		t.Errorf("expected nil replicas, got %v", *si.Replicas)
	}
	if si.MinHits != nil {
		t.Errorf("expected nil min_hits, got %d", *si.MinHits)
	}
	if cfg.Languages != nil {
		t.Errorf("expected nil languages, got %v", cfg.Languages)
	}
	if cfg.EnablePersonalization != nil {
		t.Errorf("expected nil enable_personalization, got %v", cfg.EnablePersonalization)
	}
}

func TestExpandConfigurationWithIndex_languagesEnabled(t *testing.T) {
	ctx := context.Background()

	model := &QuerySuggestionsConfigResourceModel{
		IndexName:              types.StringValue("idx"),
		Region:                 types.StringValue("us"),
		SourceIndices:          nullSourceIndexList(),
		Languages:              types.ListNull(types.StringType),
		LanguagesEnabled:       types.BoolValue(true),
		Exclude:                types.ListNull(types.StringType),
		EnablePersonalization:  types.BoolNull(),
		AllowSpecialCharacters: types.BoolNull(),
	}

	cfg, diags := expandConfigurationWithIndex(ctx, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if cfg.Languages == nil {
		t.Fatal("expected languages to be set")
	}
	langs := cfg.GetLanguages()
	if langs.Bool == nil || !*langs.Bool {
		t.Errorf("expected languages_enabled=true (bool), got %v", langs)
	}
}

func TestExpandConfigurationWithIndex_languagesDisabled(t *testing.T) {
	ctx := context.Background()

	model := &QuerySuggestionsConfigResourceModel{
		IndexName:              types.StringValue("idx"),
		Region:                 types.StringValue("us"),
		SourceIndices:          nullSourceIndexList(),
		Languages:              types.ListNull(types.StringType),
		LanguagesEnabled:       types.BoolValue(false),
		Exclude:                types.ListNull(types.StringType),
		EnablePersonalization:  types.BoolNull(),
		AllowSpecialCharacters: types.BoolNull(),
	}

	cfg, diags := expandConfigurationWithIndex(ctx, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if cfg.Languages == nil {
		t.Fatal("expected languages to be set")
	}
	langs := cfg.GetLanguages()
	if langs.Bool == nil || *langs.Bool {
		t.Errorf("expected languages_enabled=false (bool), got %v", langs)
	}
}

func TestExpandConfigurationWithIndex_invalidGenerateJSON(t *testing.T) {
	ctx := context.Background()

	sourceList := buildSourceIndexList(t, SourceIndexModel{
		IndexName:     types.StringValue("idx"),
		Replicas:      types.BoolNull(),
		AnalyticsTags: emptyStringList(),
		Facets:        emptyFacetList(),
		MinHits:       types.Int64Null(),
		MinLetters:    types.Int64Null(),
		Generate:      types.StringValue(`not-valid-json`),
		External:      emptyStringList(),
	})

	model := &QuerySuggestionsConfigResourceModel{
		IndexName:     types.StringValue("idx"),
		Region:        types.StringValue("us"),
		SourceIndices: sourceList,
		Languages:     types.ListNull(types.StringType),
	}

	_, diags := expandConfigurationWithIndex(ctx, model)
	if !diags.HasError() {
		t.Error("expected error for invalid generate JSON, got none")
	}
}

// ---- expandConfiguration (update path) ----

func TestExpandConfiguration_forUpdate(t *testing.T) {
	ctx := context.Background()

	sourceList := buildSourceIndexList(t, SourceIndexModel{
		IndexName:     types.StringValue("products"),
		Replicas:      types.BoolNull(),
		AnalyticsTags: emptyStringList(),
		Facets:        emptyFacetList(),
		MinHits:       types.Int64Null(),
		MinLetters:    types.Int64Null(),
		Generate:      types.StringNull(),
		External:      emptyStringList(),
	})

	model := &QuerySuggestionsConfigResourceModel{
		IndexName:              types.StringValue("my_suggestions"),
		Region:                 types.StringValue("us"),
		SourceIndices:          sourceList,
		Languages:              types.ListNull(types.StringType),
		LanguagesEnabled:       types.BoolNull(),
		Exclude:                types.ListValueMust(types.StringType, []attr.Value{types.StringValue("bad_query")}),
		EnablePersonalization:  types.BoolValue(false),
		AllowSpecialCharacters: types.BoolValue(true),
	}

	cfg, diags := expandConfiguration(ctx, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if len(cfg.SourceIndices) != 1 {
		t.Fatalf("expected 1 source index, got %d", len(cfg.SourceIndices))
	}
	if len(cfg.Exclude) != 1 || cfg.Exclude[0] != "bad_query" {
		t.Errorf("unexpected exclude: %v", cfg.Exclude)
	}
	if cfg.EnablePersonalization == nil || *cfg.EnablePersonalization {
		t.Error("expected enable_personalization=false")
	}
	if cfg.AllowSpecialCharacters == nil || !*cfg.AllowSpecialCharacters {
		t.Error("expected allow_special_characters=true")
	}
}

// ---- expandLanguages ----

func TestExpandLanguages_listMode(t *testing.T) {
	ctx := context.Background()

	model := &QuerySuggestionsConfigResourceModel{
		Languages:        types.ListValueMust(types.StringType, []attr.Value{types.StringValue("en"), types.StringValue("de")}),
		LanguagesEnabled: types.BoolNull(),
	}

	langs, diags := expandLanguages(ctx, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if langs == nil || langs.ArrayOfString == nil {
		t.Fatal("expected ArrayOfString languages")
	}
	if len(*langs.ArrayOfString) != 2 {
		t.Errorf("expected 2 language codes, got %d", len(*langs.ArrayOfString))
	}
}

func TestExpandLanguages_boolMode(t *testing.T) {
	ctx := context.Background()

	model := &QuerySuggestionsConfigResourceModel{
		Languages:        types.ListNull(types.StringType),
		LanguagesEnabled: types.BoolValue(true),
	}

	langs, diags := expandLanguages(ctx, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if langs == nil || langs.Bool == nil || !*langs.Bool {
		t.Error("expected Bool=true languages")
	}
}

func TestExpandLanguages_neitherSet(t *testing.T) {
	ctx := context.Background()

	model := &QuerySuggestionsConfigResourceModel{
		Languages:        types.ListNull(types.StringType),
		LanguagesEnabled: types.BoolNull(),
	}

	langs, diags := expandLanguages(ctx, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if langs != nil {
		t.Errorf("expected nil languages, got %v", langs)
	}
}
