package personalization

import (
	"reflect"
	"strconv"
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

// eventsScoringList builds an events_scoring value in the given order.
func eventsScoringList(t *testing.T, entries ...[3]string) types.List {
	t.Helper()

	values := make([]attr.Value, 0, len(entries))
	for _, entry := range entries {
		values = append(values, types.ObjectValueMust(eventsScoringModelAttrTypes, map[string]attr.Value{
			"event_name": types.StringValue(entry[0]),
			"event_type": types.StringValue(entry[1]),
			"score":      types.Int64Value(mustScore(t, entry[2])),
		}))
	}

	return types.ListValueMust(eventsScoringModelType, values)
}

func facetsScoringList(t *testing.T, entries ...[2]string) types.List {
	t.Helper()

	values := make([]attr.Value, 0, len(entries))
	for _, entry := range entries {
		values = append(values, types.ObjectValueMust(facetsScoringModelAttrTypes, map[string]attr.Value{
			"facet_name": types.StringValue(entry[0]),
			"score":      types.Int64Value(mustScore(t, entry[1])),
		}))
	}

	return types.ListValueMust(facetsScoringModelType, values)
}

func mustScore(t *testing.T, raw string) int64 {
	t.Helper()

	score, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		t.Fatalf("invalid score %q: %v", raw, err)
	}

	return score
}

func eventNames(t *testing.T, list types.List) []string {
	t.Helper()

	names := make([]string, 0, len(list.Elements()))
	for _, value := range list.Elements() {
		object, ok := value.(types.Object)
		if !ok {
			t.Fatalf("expected an object element, got %#v", value)
		}
		names = append(names, object.Attributes()["event_name"].(types.String).ValueString())
	}

	return names
}

func facetNames(t *testing.T, list types.List) []string {
	t.Helper()

	names := make([]string, 0, len(list.Elements()))
	for _, value := range list.Elements() {
		object, ok := value.(types.Object)
		if !ok {
			t.Fatalf("expected an object element, got %#v", value)
		}
		names = append(names, object.Attributes()["facet_name"].(types.String).ValueString())
	}

	return names
}

// TestHydratePersonalizationStrategyModel_PreservesConfiguredOrder is the
// regression test for a hydrate that sorted both scoring blocks: they are list
// blocks, so a configuration whose order differs from sorted order used to fail
// the apply with "Provider produced inconsistent result after apply".
func TestHydratePersonalizationStrategyModel_PreservesConfiguredOrder(t *testing.T) {
	// Deliberately not in sorted order, and not in the order the API returns.
	prior := PersonalizationStrategyResourceModel{
		EventsScoring: eventsScoringList(t,
			[3]string{"Zebra Viewed", "view", "10"},
			[3]string{"Product Clicked", "click", "50"},
			[3]string{"Alpha Converted", "conversion", "20"},
		),
		FacetsScoring: facetsScoringList(t,
			[2]string{"zebra", "5"},
			[2]string{"brand", "30"},
			[2]string{"alpha", "10"},
		),
	}

	resp := api.NewEmptyPersonalizationStrategyParams()
	resp.SetPersonalizationImpact(80)
	resp.SetEventsScoring([]api.EventsScoring{
		*api.NewEventsScoring(20, "Alpha Converted", api.EVENT_TYPE_CONVERSION),
		*api.NewEventsScoring(50, "Product Clicked", api.EVENT_TYPE_CLICK),
		*api.NewEventsScoring(10, "Zebra Viewed", api.EVENT_TYPE_VIEW),
	})
	resp.SetFacetsScoring([]api.FacetsScoring{
		*api.NewFacetsScoring(10, "alpha"),
		*api.NewFacetsScoring(30, "brand"),
		*api.NewFacetsScoring(5, "zebra"),
	})

	model := prior
	if diags := hydratePersonalizationStrategyModel(resp, &model); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	wantEvents := []string{"Zebra Viewed", "Product Clicked", "Alpha Converted"}
	if got := eventNames(t, model.EventsScoring); !reflect.DeepEqual(got, wantEvents) {
		t.Fatalf("events_scoring order = %v, want the configured order %v", got, wantEvents)
	}

	wantFacets := []string{"zebra", "brand", "alpha"}
	if got := facetNames(t, model.FacetsScoring); !reflect.DeepEqual(got, wantFacets) {
		t.Fatalf("facets_scoring order = %v, want the configured order %v", got, wantFacets)
	}

	// The API's values still win over the prior ones, only the order is taken
	// from the configuration.
	if got := model.EventsScoring.Elements()[0].(types.Object).Attributes()["score"].(types.Int64).ValueInt64(); got != 10 {
		t.Fatalf("events_scoring[0].score = %d, want the API value 10", got)
	}
}

// TestHydratePersonalizationStrategyModel_RoundTripsUnchanged checks the full
// build -> API -> hydrate loop leaves an unsorted configuration byte-identical.
func TestHydratePersonalizationStrategyModel_RoundTripsUnchanged(t *testing.T) {
	model := PersonalizationStrategyResourceModel{
		PersonalizationImpact: types.Int64Value(80),
		EventsScoring: eventsScoringList(t,
			[3]string{"Zebra Viewed", "view", "10"},
			[3]string{"Product Clicked", "click", "50"},
		),
		FacetsScoring: facetsScoringList(t,
			[2]string{"zebra", "5"},
			[2]string{"brand", "30"},
		),
	}
	planned := model

	request, diags := buildPersonalizationStrategyRequest(&model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	// Stand in for an API that returns the rules in its own order.
	response := api.NewEmptyPersonalizationStrategyParams()
	response.SetPersonalizationImpact(request.GetPersonalizationImpact())
	response.SetEventsScoring([]api.EventsScoring{request.GetEventsScoring()[1], request.GetEventsScoring()[0]})
	response.SetFacetsScoring([]api.FacetsScoring{request.GetFacetsScoring()[1], request.GetFacetsScoring()[0]})

	if diags := hydratePersonalizationStrategyModel(response, &model); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.EventsScoring.Equal(planned.EventsScoring) {
		t.Fatalf("events_scoring = %s, want the planned value %s", model.EventsScoring, planned.EventsScoring)
	}
	if !model.FacetsScoring.Equal(planned.FacetsScoring) {
		t.Fatalf("facets_scoring = %s, want the planned value %s", model.FacetsScoring, planned.FacetsScoring)
	}
}

// TestHydratePersonalizationStrategyModel_AppendsUnknownRules covers reads where
// the API reports rules the configuration never mentioned - added through the
// dashboard, or every rule at all during an import.
func TestHydratePersonalizationStrategyModel_AppendsUnknownRules(t *testing.T) {
	resp := api.NewEmptyPersonalizationStrategyParams()
	resp.SetPersonalizationImpact(80)
	resp.SetEventsScoring([]api.EventsScoring{
		*api.NewEventsScoring(10, "Zebra Viewed", api.EVENT_TYPE_VIEW),
		*api.NewEventsScoring(50, "Product Clicked", api.EVENT_TYPE_CLICK),
	})
	resp.SetFacetsScoring([]api.FacetsScoring{
		*api.NewFacetsScoring(5, "zebra"),
		*api.NewFacetsScoring(30, "brand"),
	})

	// Import: no prior value at all, so the order is the deterministic sorted
	// one rather than the API's.
	var imported PersonalizationStrategyResourceModel
	if diags := hydratePersonalizationStrategyModel(resp, &imported); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := eventNames(t, imported.EventsScoring); !reflect.DeepEqual(got, []string{"Product Clicked", "Zebra Viewed"}) {
		t.Fatalf("imported events_scoring order = %v, want a deterministic sorted order", got)
	}
	if got := facetNames(t, imported.FacetsScoring); !reflect.DeepEqual(got, []string{"brand", "zebra"}) {
		t.Fatalf("imported facets_scoring order = %v, want a deterministic sorted order", got)
	}

	// Refresh: the configured rule keeps its position, the out-of-band one is
	// appended.
	refreshed := PersonalizationStrategyResourceModel{
		EventsScoring: eventsScoringList(t, [3]string{"Zebra Viewed", "view", "10"}),
		FacetsScoring: facetsScoringList(t, [2]string{"zebra", "5"}),
	}
	if diags := hydratePersonalizationStrategyModel(resp, &refreshed); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := eventNames(t, refreshed.EventsScoring); !reflect.DeepEqual(got, []string{"Zebra Viewed", "Product Clicked"}) {
		t.Fatalf("refreshed events_scoring order = %v, want the configured rule first", got)
	}
	if got := facetNames(t, refreshed.FacetsScoring); !reflect.DeepEqual(got, []string{"zebra", "brand"}) {
		t.Fatalf("refreshed facets_scoring order = %v, want the configured rule first", got)
	}
}

func TestBuildPersonalizationStrategyRequest_RejectsMistypedAttributes(t *testing.T) {
	model := PersonalizationStrategyResourceModel{
		PersonalizationImpact: types.Int64Value(80),
		EventsScoring: types.ListValueMust(types.ObjectType{AttrTypes: map[string]attr.Type{"event_name": types.Int64Type}}, []attr.Value{
			types.ObjectValueMust(map[string]attr.Type{"event_name": types.Int64Type}, map[string]attr.Value{
				"event_name": types.Int64Value(1),
			}),
		}),
	}

	if _, diags := buildPersonalizationStrategyRequest(&model); !diags.HasError() {
		t.Fatal("expected a diagnostic for a mistyped event_name, got none")
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
