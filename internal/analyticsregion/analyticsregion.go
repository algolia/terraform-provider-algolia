package analyticsregion

import (
	"errors"
	"fmt"
	"strings"

	abtesting "github.com/algolia/algoliasearch-client-go/v4/algolia/abtesting-v3"
	ingestion "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	personalization "github.com/algolia/algoliasearch-client-go/v4/algolia/personalization"
	suggestions "github.com/algolia/algoliasearch-client-go/v4/algolia/query-suggestions"
)

const (
	EnvVar              = "ALGOLIA_ANALYTICS_REGION"
	MissingRegionDetail = "set analytics_region in the provider configuration or use the ALGOLIA_ANALYTICS_REGION environment variable"
)

func Normalize(region string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(region))
	switch normalized {
	case "":
		return "", nil
	case "us", "eu":
		return normalized, nil
	default:
		return "", fmt.Errorf("analytics region must be one of: us, eu")
	}
}

func Require(region string) (string, error) {
	normalized, err := Normalize(region)
	if err != nil {
		return "", err
	}
	if normalized == "" {
		return "", errors.New(MissingRegionDetail)
	}
	return normalized, nil
}

func NewQuerySuggestionsClient(appID, apiKey, region string) (*suggestions.APIClient, error) {
	normalized, err := Require(region)
	if err != nil {
		return nil, err
	}

	var apiRegion suggestions.Region
	switch normalized {
	case "us":
		apiRegion = suggestions.US
	case "eu":
		apiRegion = suggestions.EU
	default:
		return nil, fmt.Errorf("analytics region must be one of: us, eu")
	}

	return suggestions.NewClient(appID, apiKey, apiRegion)
}

func NewPersonalizationClient(appID, apiKey, region string) (*personalization.APIClient, error) {
	normalized, err := Require(region)
	if err != nil {
		return nil, err
	}

	var apiRegion personalization.Region
	switch normalized {
	case "us":
		apiRegion = personalization.US
	case "eu":
		apiRegion = personalization.EU
	default:
		return nil, fmt.Errorf("analytics region must be one of: us, eu")
	}

	return personalization.NewClient(appID, apiKey, apiRegion)
}

func NewIngestionClient(appID, apiKey, region string) (*ingestion.APIClient, error) {
	normalized, err := Require(region)
	if err != nil {
		return nil, err
	}

	var apiRegion ingestion.Region
	switch normalized {
	case "us":
		apiRegion = ingestion.US
	case "eu":
		apiRegion = ingestion.EU
	default:
		return nil, fmt.Errorf("analytics region must be one of: us, eu")
	}

	return ingestion.NewClient(appID, apiKey, apiRegion)
}

// NewABTestingClient builds an abtesting-v3 client for the given normalized
// analytics region.
//
// Unlike every other region-routed client in this provider (Query
// Suggestions, Personalization, Ingestion - all of which expose a Region
// enum of exactly "eu"/"us"), the A/B Testing API's Region enum is "de"/
// "us": its EU-hosted cluster lives in Germany and the client models that
// as abtesting.DE, not an abtesting.EU that does not exist. To keep this
// provider's public surface consistent - a single `analytics_region` of
// "us"/"eu" governs every region-routed resource - "eu" is mapped to
// abtesting.DE here rather than surfacing "de" as a third region value.
func NewABTestingClient(appID, apiKey, region string) (*abtesting.APIClient, error) {
	normalized, err := Require(region)
	if err != nil {
		return nil, err
	}

	var apiRegion abtesting.Region
	switch normalized {
	case "us":
		apiRegion = abtesting.US
	case "eu":
		apiRegion = abtesting.DE
	default:
		return nil, fmt.Errorf("analytics region must be one of: us, eu")
	}

	return abtesting.NewClient(appID, apiKey, apiRegion)
}
