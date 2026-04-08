package analyticsregion

import (
	"errors"
	"fmt"
	"strings"

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
