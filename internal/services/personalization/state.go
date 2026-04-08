package personalization

import (
	"context"
	"fmt"
	"sort"
	"time"

	api "github.com/algolia/algoliasearch-client-go/v4/algolia/personalization"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func buildPersonalizationStrategyRequest(model *PersonalizationStrategyResourceModel) (*api.PersonalizationStrategyParams, diag.Diagnostics) {
	var diags diag.Diagnostics

	request := api.NewEmptyPersonalizationStrategyParams()
	request.SetPersonalizationImpact(int32(model.PersonalizationImpact.ValueInt64()))

	if !model.EventsScoring.IsNull() && !model.EventsScoring.IsUnknown() {
		events := make([]api.EventsScoring, 0, len(model.EventsScoring.Elements()))
		for _, value := range model.EventsScoring.Elements() {
			object := value.(types.Object)
			attrs := object.Attributes()
			eventName := attrs["event_name"].(types.String).ValueString()
			eventType := api.EventType(attrs["event_type"].(types.String).ValueString())
			score := int32(attrs["score"].(types.Int64).ValueInt64())
			events = append(events, *api.NewEventsScoring(score, eventName, eventType))
		}
		request.SetEventsScoring(events)
	}

	if !model.FacetsScoring.IsNull() && !model.FacetsScoring.IsUnknown() {
		facets := make([]api.FacetsScoring, 0, len(model.FacetsScoring.Elements()))
		for _, value := range model.FacetsScoring.Elements() {
			object := value.(types.Object)
			attrs := object.Attributes()
			facetName := attrs["facet_name"].(types.String).ValueString()
			score := int32(attrs["score"].(types.Int64).ValueInt64())
			facets = append(facets, *api.NewFacetsScoring(score, facetName))
		}
		request.SetFacetsScoring(facets)
	}

	return request, diags
}

func hydratePersonalizationStrategyModel(resp *api.PersonalizationStrategyParams, model *PersonalizationStrategyResourceModel) diag.Diagnostics {
	eventsValues := make([]attr.Value, 0, len(resp.GetEventsScoring()))
	events := append([]api.EventsScoring(nil), resp.GetEventsScoring()...)
	sort.Slice(events, func(i, j int) bool {
		if events[i].GetEventType() != events[j].GetEventType() {
			return events[i].GetEventType() < events[j].GetEventType()
		}
		if events[i].GetEventName() != events[j].GetEventName() {
			return events[i].GetEventName() < events[j].GetEventName()
		}
		return events[i].GetScore() < events[j].GetScore()
	})
	for _, event := range events {
		eventsValues = append(eventsValues, types.ObjectValueMust(eventsScoringModelAttrTypes, map[string]attr.Value{
			"event_name": types.StringValue(event.GetEventName()),
			"event_type": types.StringValue(string(event.GetEventType())),
			"score":      types.Int64Value(int64(event.GetScore())),
		}))
	}

	facetsValues := make([]attr.Value, 0, len(resp.GetFacetsScoring()))
	facets := append([]api.FacetsScoring(nil), resp.GetFacetsScoring()...)
	sort.Slice(facets, func(i, j int) bool {
		if facets[i].GetFacetName() != facets[j].GetFacetName() {
			return facets[i].GetFacetName() < facets[j].GetFacetName()
		}
		return facets[i].GetScore() < facets[j].GetScore()
	})
	for _, facet := range facets {
		facetsValues = append(facetsValues, types.ObjectValueMust(facetsScoringModelAttrTypes, map[string]attr.Value{
			"facet_name": types.StringValue(facet.GetFacetName()),
			"score":      types.Int64Value(int64(facet.GetScore())),
		}))
	}

	model.ID = types.StringValue(personalizationStrategyID)
	model.PersonalizationImpact = types.Int64Value(int64(resp.GetPersonalizationImpact()))
	model.EventsScoring = types.ListValueMust(eventsScoringModelType, eventsValues)
	model.FacetsScoring = types.ListValueMust(facetsScoringModelType, facetsValues)

	return nil
}

func disabledPersonalizationStrategy(current *api.PersonalizationStrategyParams) *api.PersonalizationStrategyParams {
	request := api.NewEmptyPersonalizationStrategyParams()
	request.SetPersonalizationImpact(0)

	events := make([]api.EventsScoring, 0, len(current.GetEventsScoring()))
	for _, event := range current.GetEventsScoring() {
		events = append(events, *api.NewEventsScoring(0, event.GetEventName(), event.GetEventType()))
	}
	request.SetEventsScoring(events)

	facets := make([]api.FacetsScoring, 0, len(current.GetFacetsScoring()))
	for _, facet := range current.GetFacetsScoring() {
		facets = append(facets, *api.NewFacetsScoring(0, facet.GetFacetName()))
	}
	request.SetFacetsScoring(facets)

	return request
}

func personalizationStrategyMatches(actual, expected *api.PersonalizationStrategyParams) bool {
	if actual.GetPersonalizationImpact() != expected.GetPersonalizationImpact() {
		return false
	}

	actualEvents := append([]api.EventsScoring(nil), actual.GetEventsScoring()...)
	expectedEvents := append([]api.EventsScoring(nil), expected.GetEventsScoring()...)
	sort.Slice(actualEvents, func(i, j int) bool {
		if actualEvents[i].GetEventType() != actualEvents[j].GetEventType() {
			return actualEvents[i].GetEventType() < actualEvents[j].GetEventType()
		}
		if actualEvents[i].GetEventName() != actualEvents[j].GetEventName() {
			return actualEvents[i].GetEventName() < actualEvents[j].GetEventName()
		}
		return actualEvents[i].GetScore() < actualEvents[j].GetScore()
	})
	sort.Slice(expectedEvents, func(i, j int) bool {
		if expectedEvents[i].GetEventType() != expectedEvents[j].GetEventType() {
			return expectedEvents[i].GetEventType() < expectedEvents[j].GetEventType()
		}
		if expectedEvents[i].GetEventName() != expectedEvents[j].GetEventName() {
			return expectedEvents[i].GetEventName() < expectedEvents[j].GetEventName()
		}
		return expectedEvents[i].GetScore() < expectedEvents[j].GetScore()
	})
	if len(actualEvents) != len(expectedEvents) {
		return false
	}
	for i := range actualEvents {
		if actualEvents[i].GetEventName() != expectedEvents[i].GetEventName() ||
			actualEvents[i].GetEventType() != expectedEvents[i].GetEventType() ||
			actualEvents[i].GetScore() != expectedEvents[i].GetScore() {
			return false
		}
	}

	actualFacets := append([]api.FacetsScoring(nil), actual.GetFacetsScoring()...)
	expectedFacets := append([]api.FacetsScoring(nil), expected.GetFacetsScoring()...)
	sort.Slice(actualFacets, func(i, j int) bool {
		if actualFacets[i].GetFacetName() != actualFacets[j].GetFacetName() {
			return actualFacets[i].GetFacetName() < actualFacets[j].GetFacetName()
		}
		return actualFacets[i].GetScore() < actualFacets[j].GetScore()
	})
	sort.Slice(expectedFacets, func(i, j int) bool {
		if expectedFacets[i].GetFacetName() != expectedFacets[j].GetFacetName() {
			return expectedFacets[i].GetFacetName() < expectedFacets[j].GetFacetName()
		}
		return expectedFacets[i].GetScore() < expectedFacets[j].GetScore()
	})
	if len(actualFacets) != len(expectedFacets) {
		return false
	}
	for i := range actualFacets {
		if actualFacets[i].GetFacetName() != expectedFacets[i].GetFacetName() ||
			actualFacets[i].GetScore() != expectedFacets[i].GetScore() {
			return false
		}
	}

	return true
}

func waitForPersonalizationStrategy(ctx context.Context, client *api.APIClient, expected *api.PersonalizationStrategyParams) (*api.PersonalizationStrategyParams, error) {
	deadline := time.Now().Add(2 * time.Minute)
	interval := 2 * time.Second

	for time.Now().Before(deadline) {
		resp, err := client.GetPersonalizationStrategy()
		if err == nil && personalizationStrategyMatches(resp, expected) {
			return resp, nil
		}
		if err != nil {
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}

		if interval < 10*time.Second {
			interval += time.Second
		}
	}

	return nil, fmt.Errorf("personalization strategy did not converge within 2 minutes")
}
