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
		for i, value := range model.EventsScoring.Elements() {
			object, ok := value.(types.Object)
			if !ok {
				diags.AddError("Invalid events_scoring value", fmt.Sprintf("events_scoring[%d] is not an object.", i))
				return nil, diags
			}

			attrs := object.Attributes()
			eventName, ok := attrs["event_name"].(types.String)
			if !ok {
				diags.AddError("Invalid events_scoring value", fmt.Sprintf("events_scoring[%d].event_name is not a string.", i))
				return nil, diags
			}
			eventType, ok := attrs["event_type"].(types.String)
			if !ok {
				diags.AddError("Invalid events_scoring value", fmt.Sprintf("events_scoring[%d].event_type is not a string.", i))
				return nil, diags
			}
			score, ok := attrs["score"].(types.Int64)
			if !ok {
				diags.AddError("Invalid events_scoring value", fmt.Sprintf("events_scoring[%d].score is not a number.", i))
				return nil, diags
			}

			events = append(events, *api.NewEventsScoring(
				int32(score.ValueInt64()),
				eventName.ValueString(),
				api.EventType(eventType.ValueString()),
			))
		}
		request.SetEventsScoring(events)
	}

	if !model.FacetsScoring.IsNull() && !model.FacetsScoring.IsUnknown() {
		facets := make([]api.FacetsScoring, 0, len(model.FacetsScoring.Elements()))
		for i, value := range model.FacetsScoring.Elements() {
			object, ok := value.(types.Object)
			if !ok {
				diags.AddError("Invalid facets_scoring value", fmt.Sprintf("facets_scoring[%d] is not an object.", i))
				return nil, diags
			}

			attrs := object.Attributes()
			facetName, ok := attrs["facet_name"].(types.String)
			if !ok {
				diags.AddError("Invalid facets_scoring value", fmt.Sprintf("facets_scoring[%d].facet_name is not a string.", i))
				return nil, diags
			}
			score, ok := attrs["score"].(types.Int64)
			if !ok {
				diags.AddError("Invalid facets_scoring value", fmt.Sprintf("facets_scoring[%d].score is not a number.", i))
				return nil, diags
			}

			facets = append(facets, *api.NewFacetsScoring(int32(score.ValueInt64()), facetName.ValueString()))
		}
		request.SetFacetsScoring(facets)
	}

	return request, diags
}

// hydratePersonalizationStrategyModel writes the API response into model. model
// must already hold the prior value (the plan during Create/Update, the state
// during Read): events_scoring and facets_scoring are list blocks, so element
// order is part of the value Terraform compares between plan and apply, and the
// response has to be put back into the order the configuration used. See
// orderEventsScoring.
func hydratePersonalizationStrategyModel(resp *api.PersonalizationStrategyParams, model *PersonalizationStrategyResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	events := orderEventsScoring(resp.GetEventsScoring(), model.EventsScoring)
	eventsValues := make([]attr.Value, 0, len(events))
	for _, event := range events {
		value, valueDiags := types.ObjectValue(eventsScoringModelAttrTypes, map[string]attr.Value{
			"event_name": types.StringValue(event.GetEventName()),
			"event_type": types.StringValue(string(event.GetEventType())),
			"score":      types.Int64Value(int64(event.GetScore())),
		})
		diags.Append(valueDiags...)
		if diags.HasError() {
			return diags
		}
		eventsValues = append(eventsValues, value)
	}

	facets := orderFacetsScoring(resp.GetFacetsScoring(), model.FacetsScoring)
	facetsValues := make([]attr.Value, 0, len(facets))
	for _, facet := range facets {
		value, valueDiags := types.ObjectValue(facetsScoringModelAttrTypes, map[string]attr.Value{
			"facet_name": types.StringValue(facet.GetFacetName()),
			"score":      types.Int64Value(int64(facet.GetScore())),
		})
		diags.Append(valueDiags...)
		if diags.HasError() {
			return diags
		}
		facetsValues = append(facetsValues, value)
	}

	eventsList, eventsDiags := types.ListValue(eventsScoringModelType, eventsValues)
	diags.Append(eventsDiags...)
	facetsList, facetsDiags := types.ListValue(facetsScoringModelType, facetsValues)
	diags.Append(facetsDiags...)
	if diags.HasError() {
		return diags
	}

	model.ID = types.StringValue(personalizationStrategyID)
	model.PersonalizationImpact = types.Int64Value(int64(resp.GetPersonalizationImpact()))
	model.EventsScoring = eventsList
	model.FacetsScoring = facetsList

	return diags
}

// eventsScoringKey identifies one events_scoring rule. The Personalization API
// keys these rules by event name and type - the score is the value, and the
// position in the array carries no meaning.
func eventsScoringKey(name string, eventType api.EventType) string {
	return string(eventType) + "\x00" + name
}

// orderEventsScoring returns the API's scoring rules in the order the prior
// value listed them.
//
// events_scoring is a ListNestedBlock, so element order is part of the value
// Terraform compares between the plan and the applied result. Because order
// means nothing to the API, the API is free to return the rules in a different
// order than they were submitted - which is why this reorders the response
// rather than simply not sorting it. Imposing any provider-chosen order (the
// previous behaviour sorted the response) makes every configuration whose block
// order differs fail with "Provider produced inconsistent result after apply".
//
// Rules the prior value does not mention - added outside Terraform, or read
// during an import, where there is no prior value at all - are appended in a
// deterministic sorted order.
func orderEventsScoring(events []api.EventsScoring, prior types.List) []api.EventsScoring {
	used := make([]bool, len(events))
	ordered := make([]api.EventsScoring, 0, len(events))

	for _, key := range priorEventsScoringKeys(prior) {
		for i, event := range events {
			if used[i] || eventsScoringKey(event.GetEventName(), event.GetEventType()) != key {
				continue
			}

			used[i] = true
			ordered = append(ordered, event)
			break
		}
	}

	leftovers := make([]api.EventsScoring, 0, len(events))
	for i, event := range events {
		if !used[i] {
			leftovers = append(leftovers, event)
		}
	}
	sort.Slice(leftovers, func(i, j int) bool {
		if leftovers[i].GetEventType() != leftovers[j].GetEventType() {
			return leftovers[i].GetEventType() < leftovers[j].GetEventType()
		}
		if leftovers[i].GetEventName() != leftovers[j].GetEventName() {
			return leftovers[i].GetEventName() < leftovers[j].GetEventName()
		}
		return leftovers[i].GetScore() < leftovers[j].GetScore()
	})

	return append(ordered, leftovers...)
}

// orderFacetsScoring is orderEventsScoring for facets_scoring, whose rules the
// API keys by facet name.
func orderFacetsScoring(facets []api.FacetsScoring, prior types.List) []api.FacetsScoring {
	used := make([]bool, len(facets))
	ordered := make([]api.FacetsScoring, 0, len(facets))

	for _, key := range priorFacetsScoringKeys(prior) {
		for i, facet := range facets {
			if used[i] || facet.GetFacetName() != key {
				continue
			}

			used[i] = true
			ordered = append(ordered, facet)
			break
		}
	}

	leftovers := make([]api.FacetsScoring, 0, len(facets))
	for i, facet := range facets {
		if !used[i] {
			leftovers = append(leftovers, facet)
		}
	}
	sort.Slice(leftovers, func(i, j int) bool {
		if leftovers[i].GetFacetName() != leftovers[j].GetFacetName() {
			return leftovers[i].GetFacetName() < leftovers[j].GetFacetName()
		}
		return leftovers[i].GetScore() < leftovers[j].GetScore()
	})

	return append(ordered, leftovers...)
}

func priorEventsScoringKeys(prior types.List) []string {
	if prior.IsNull() || prior.IsUnknown() {
		return nil
	}

	keys := make([]string, 0, len(prior.Elements()))
	for _, value := range prior.Elements() {
		object, ok := value.(types.Object)
		if !ok || object.IsNull() || object.IsUnknown() {
			continue
		}

		attrs := object.Attributes()
		name, nameOk := attrs["event_name"].(types.String)
		eventType, typeOk := attrs["event_type"].(types.String)
		if !nameOk || !typeOk || name.IsUnknown() || eventType.IsUnknown() {
			continue
		}

		keys = append(keys, eventsScoringKey(name.ValueString(), api.EventType(eventType.ValueString())))
	}

	return keys
}

func priorFacetsScoringKeys(prior types.List) []string {
	if prior.IsNull() || prior.IsUnknown() {
		return nil
	}

	keys := make([]string, 0, len(prior.Elements()))
	for _, value := range prior.Elements() {
		object, ok := value.(types.Object)
		if !ok || object.IsNull() || object.IsUnknown() {
			continue
		}

		name, nameOk := object.Attributes()["facet_name"].(types.String)
		if !nameOk || name.IsUnknown() {
			continue
		}

		keys = append(keys, name.ValueString())
	}

	return keys
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
		resp, err := client.GetPersonalizationStrategy(api.WithContext(ctx))
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
