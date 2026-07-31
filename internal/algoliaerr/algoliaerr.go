// Package algoliaerr classifies errors returned by the Algolia Go client.
//
// The client generates one APIError type per API surface. The types are
// structurally identical (a message and an HTTP status) but nominally distinct,
// so there is no single type a caller can match against. Every resource in this
// provider needs the same handful of decisions -- "is this a 404, so should the
// resource be dropped from state?", "is this worth retrying?" -- and without a
// shared helper each one hand-rolls its own errors.As over whichever type its
// own client happens to return. This package holds that knowledge in one place.
//
// For the same reason it also owns the wording of the diagnostics a resource
// raises when an operation fails (see Subject): the sentence was hand-copied
// across sibling resources, which is how a fix to one of them stops short of the
// rest.
package algoliaerr

import (
	"errors"
	"net/http"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/abtesting"
	abtestingV3 "github.com/algolia/algoliasearch-client-go/v4/algolia/abtesting-v3"
	agentStudio "github.com/algolia/algoliasearch-client-go/v4/algolia/agent-studio"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/analytics"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/composition"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/insights"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/monitoring"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/personalization"
	suggestions "github.com/algolia/algoliasearch-client-go/v4/algolia/query-suggestions"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/recommend"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
)

// Status returns the HTTP status code reported by an Algolia API error anywhere
// in err's chain. The second return value is false when err is nil or wraps no
// Algolia API error, in which case no status is available -- callers must not
// read the zero status as meaningful.
func Status(err error) (int, bool) {
	status, _, ok := apiError(err)
	return status, ok
}

// extras returns the properties an Algolia API error carries beyond its message,
// which is where the API puts anything structured it has to say about a failure -
// see Explain. Nil when err wraps no Algolia API error, or carries nothing extra.
func extras(err error) map[string]any {
	_, properties, _ := apiError(err)
	return properties
}

// apiError finds an Algolia API error anywhere in err's chain and reports its
// status and its additional properties.
//
// Each API surface needs its own concrete target type, so this is a dispatch
// table rather than a loop: add a pair of cases when the client gains a surface.
// Both the pointer and the value form of each type are matched, because the
// generated APIError declares Error() on its value receiver and so satisfies the
// error interface either way, even though the clients all return pointers. The
// pointer cases guard against a typed nil, which errors.As happily matches.
func apiError(err error) (int, map[string]any, bool) {
	if err == nil {
		return 0, nil, false
	}

	var (
		searchPtr, searchVal                   = (*search.APIError)(nil), search.APIError{}
		recommendPtr, recommendVal             = (*recommend.APIError)(nil), recommend.APIError{}
		ingestionPtr, ingestionVal             = (*ingestion.APIError)(nil), ingestion.APIError{}
		compositionPtr, compositionVal         = (*composition.APIError)(nil), composition.APIError{}
		personalizationPtr, personalizationVal = (*personalization.APIError)(nil), personalization.APIError{}
		suggestionsPtr, suggestionsVal         = (*suggestions.APIError)(nil), suggestions.APIError{}
		abtestingPtr, abtestingVal             = (*abtesting.APIError)(nil), abtesting.APIError{}
		abtestingV3Ptr, abtestingV3Val         = (*abtestingV3.APIError)(nil), abtestingV3.APIError{}
		agentStudioPtr, agentStudioVal         = (*agentStudio.APIError)(nil), agentStudio.APIError{}
		analyticsPtr, analyticsVal             = (*analytics.APIError)(nil), analytics.APIError{}
		insightsPtr, insightsVal               = (*insights.APIError)(nil), insights.APIError{}
		monitoringPtr, monitoringVal           = (*monitoring.APIError)(nil), monitoring.APIError{}
	)

	switch {
	case errors.As(err, &searchPtr) && searchPtr != nil:
		return searchPtr.Status, searchPtr.AdditionalProperties, true
	case errors.As(err, &searchVal):
		return searchVal.Status, searchVal.AdditionalProperties, true

	case errors.As(err, &recommendPtr) && recommendPtr != nil:
		return recommendPtr.Status, recommendPtr.AdditionalProperties, true
	case errors.As(err, &recommendVal):
		return recommendVal.Status, recommendVal.AdditionalProperties, true

	case errors.As(err, &ingestionPtr) && ingestionPtr != nil:
		return ingestionPtr.Status, ingestionPtr.AdditionalProperties, true
	case errors.As(err, &ingestionVal):
		return ingestionVal.Status, ingestionVal.AdditionalProperties, true

	case errors.As(err, &compositionPtr) && compositionPtr != nil:
		return compositionPtr.Status, compositionPtr.AdditionalProperties, true
	case errors.As(err, &compositionVal):
		return compositionVal.Status, compositionVal.AdditionalProperties, true

	case errors.As(err, &personalizationPtr) && personalizationPtr != nil:
		return personalizationPtr.Status, personalizationPtr.AdditionalProperties, true
	case errors.As(err, &personalizationVal):
		return personalizationVal.Status, personalizationVal.AdditionalProperties, true

	case errors.As(err, &suggestionsPtr) && suggestionsPtr != nil:
		return suggestionsPtr.Status, suggestionsPtr.AdditionalProperties, true
	case errors.As(err, &suggestionsVal):
		return suggestionsVal.Status, suggestionsVal.AdditionalProperties, true

	case errors.As(err, &abtestingPtr) && abtestingPtr != nil:
		return abtestingPtr.Status, abtestingPtr.AdditionalProperties, true
	case errors.As(err, &abtestingVal):
		return abtestingVal.Status, abtestingVal.AdditionalProperties, true

	case errors.As(err, &abtestingV3Ptr) && abtestingV3Ptr != nil:
		return abtestingV3Ptr.Status, abtestingV3Ptr.AdditionalProperties, true
	case errors.As(err, &abtestingV3Val):
		return abtestingV3Val.Status, abtestingV3Val.AdditionalProperties, true

	case errors.As(err, &agentStudioPtr) && agentStudioPtr != nil:
		return agentStudioPtr.Status, agentStudioPtr.AdditionalProperties, true
	case errors.As(err, &agentStudioVal):
		return agentStudioVal.Status, agentStudioVal.AdditionalProperties, true

	case errors.As(err, &analyticsPtr) && analyticsPtr != nil:
		return analyticsPtr.Status, analyticsPtr.AdditionalProperties, true
	case errors.As(err, &analyticsVal):
		return analyticsVal.Status, analyticsVal.AdditionalProperties, true

	case errors.As(err, &insightsPtr) && insightsPtr != nil:
		return insightsPtr.Status, insightsPtr.AdditionalProperties, true
	case errors.As(err, &insightsVal):
		return insightsVal.Status, insightsVal.AdditionalProperties, true

	case errors.As(err, &monitoringPtr) && monitoringPtr != nil:
		return monitoringPtr.Status, monitoringPtr.AdditionalProperties, true
	case errors.As(err, &monitoringVal):
		return monitoringVal.Status, monitoringVal.AdditionalProperties, true
	}

	return 0, nil, false
}

// IsNotFound reports whether err is an Algolia API error with status 404.
//
// Resource Read implementations use this to tell "deleted out of band", which
// must remove the resource from Terraform state so the next plan recreates it,
// apart from a genuine failure that has to surface as a diagnostic. Note that
// ImportState wants the opposite: importing an absent resource must fail.
func IsNotFound(err error) bool {
	status, ok := Status(err)
	return ok && status == http.StatusNotFound
}

// IsRetryable reports whether err is an Algolia API error whose status suggests
// the same request may succeed later: 429 (rate limited) or any 5xx.
func IsRetryable(err error) bool {
	status, ok := Status(err)
	if !ok {
		return false
	}
	return status == http.StatusTooManyRequests || (status >= 500 && status <= 599)
}
