package personalization

import (
	"testing"

	api "github.com/algolia/algoliasearch-client-go/v4/algolia/personalization"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildPersonalizationStrategyRequest(t *testing.T) {
	model := PersonalizationStrategyResourceModel{
		PersonalizationImpact: types.Int64Value(80),
		EventsScoring: types.ListValueMust(eventsScoringModelType, []attr.Value{
			types.ObjectValueMust(eventsScoringModelAttrTypes, map[string]attr.Value{
				"event_name": types.StringValue("Product Clicked"),
				"event_type": types.StringValue("click"),
				"score":      types.Int64Value(50),
			}),
		}),
		FacetsScoring: types.ListValueMust(facetsScoringModelType, []attr.Value{
			types.ObjectValueMust(facetsScoringModelAttrTypes, map[string]attr.Value{
				"facet_name": types.StringValue("brand"),
				"score":      types.Int64Value(30),
			}),
		}),
	}

	req, diags := buildPersonalizationStrategyRequest(&model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := req.GetPersonalizationImpact(); got != 80 {
		t.Fatalf("personalization_impact = %d, want 80", got)
	}
	if got := req.GetEventsScoring(); len(got) != 1 || got[0].GetEventName() != "Product Clicked" {
		t.Fatalf("events_scoring = %#v, want one Product Clicked entry", got)
	}
	if got := req.GetFacetsScoring(); len(got) != 1 || got[0].GetFacetName() != "brand" {
		t.Fatalf("facets_scoring = %#v, want one brand entry", got)
	}
}

func TestHydratePersonalizationStrategyModel(t *testing.T) {
	resp := api.NewEmptyPersonalizationStrategyParams()
	resp.SetPersonalizationImpact(70)
	resp.SetEventsScoring([]api.EventsScoring{
		*api.NewEventsScoring(40, "Product Clicked", api.EVENT_TYPE_CLICK),
	})
	resp.SetFacetsScoring([]api.FacetsScoring{
		*api.NewFacetsScoring(20, "category"),
	})

	var model PersonalizationStrategyResourceModel
	diags := hydratePersonalizationStrategyModel(resp, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.ID.ValueString(); got != personalizationStrategyID {
		t.Fatalf("id = %q, want %q", got, personalizationStrategyID)
	}
	if got := model.PersonalizationImpact.ValueInt64(); got != 70 {
		t.Fatalf("personalization_impact = %d, want 70", got)
	}
}

func TestPersonalizationStrategySchemas_RegisterExpectedAttributes(t *testing.T) {
	resourceSchema := personalizationStrategyResourceSchema()
	idAttr, ok := resourceSchema.Attributes["id"].(resourceschema.StringAttribute)
	if !ok || !idAttr.Computed {
		t.Fatal("expected id to be a computed string attribute")
	}

	impactAttr, ok := resourceSchema.Attributes["personalization_impact"].(resourceschema.Int64Attribute)
	if !ok || !impactAttr.Required {
		t.Fatal("expected personalization_impact to be a required int64 attribute")
	}

	dataSourceSchema := personalizationStrategyDataSourceSchema()
	dsIDAttr, ok := dataSourceSchema.Attributes["id"].(datasourceschema.StringAttribute)
	if !ok || !dsIDAttr.Computed {
		t.Fatal("expected data source id to be computed")
	}

	if _, ok := dataSourceSchema.Blocks["events_scoring"]; !ok {
		t.Fatal("expected events_scoring block on personalization strategy data source")
	}
	if _, ok := dataSourceSchema.Blocks["facets_scoring"]; !ok {
		t.Fatal("expected facets_scoring block on personalization strategy data source")
	}
}
