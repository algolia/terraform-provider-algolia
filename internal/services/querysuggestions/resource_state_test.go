package querysuggestions

import (
	"testing"

	suggestions "github.com/algolia/algoliasearch-client-go/v4/algolia/query-suggestions"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildConfigurationWithIndex(t *testing.T) {
	model := QuerySuggestionsResourceModel{
		IndexName: types.StringValue("qs_products"),
		Region:    types.StringValue("us"),
		SourceIndices: types.ListValueMust(sourceIndexModelType, []attr.Value{
			types.ObjectValueMust(sourceIndexModelAttrTypes, map[string]attr.Value{
				"index_name":     types.StringValue("products"),
				"analytics_tags": types.SetValueMust(types.StringType, []attr.Value{types.StringValue("mobile")}),
				"facets": types.ListValueMust(facetModelType, []attr.Value{
					types.ObjectValueMust(facetModelAttrTypes, map[string]attr.Value{
						"attribute": types.StringValue("brand"),
						"amount":    types.Int64Value(5),
					}),
				}),
				"min_hits":    types.Int64Value(10),
				"min_letters": types.Int64Value(3),
				"generate": types.ListValueMust(types.ListType{ElemType: types.StringType}, []attr.Value{
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

	model := QuerySuggestionsResourceModel{Region: types.StringValue("us")}
	diags := hydrateQuerySuggestionsModel(resp, "us", &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.ID.ValueString(); got != "us/qs_products" {
		t.Fatalf("id = %q, want composite id", got)
	}
	if got := model.IndexName.ValueString(); got != "qs_products" {
		t.Fatalf("index_name = %q, want qs_products", got)
	}
}

