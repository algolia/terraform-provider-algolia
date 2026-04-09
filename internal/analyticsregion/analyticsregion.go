package analyticsregion

import (
	"errors"
	"fmt"
	"strings"

	personalization "github.com/algolia/algoliasearch-client-go/v4/algolia/personalization"
	suggestions "github.com/algolia/algoliasearch-client-go/v4/algolia/query-suggestions"
)

const (
	QuerySuggestionsEnvVar    = "ALGOLIA_QUERY_SUGGESTIONS_REGION"
	QuerySuggestionsAttribute = "query_suggestions_region"
	PersonalizationEnvVar     = "ALGOLIA_PERSONALIZATION_REGION"
	PersonalizationAttribute  = "personalization_region"
	validRegionValues         = "us, eu"
)

type serviceConfig struct {
	name          string
	attributeName string
	envVar        string
}

var (
	querySuggestionsConfig = serviceConfig{
		name:          "Query Suggestions",
		attributeName: QuerySuggestionsAttribute,
		envVar:        QuerySuggestionsEnvVar,
	}
	personalizationConfig = serviceConfig{
		name:          "Personalization",
		attributeName: PersonalizationAttribute,
		envVar:        PersonalizationEnvVar,
	}
)

func Normalize(region string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(region))
	switch normalized {
	case "":
		return "", nil
	case "us", "eu":
		return normalized, nil
	default:
		return "", fmt.Errorf("region must be one of: %s", validRegionValues)
	}
}

func Require(region string, attributeName, envVar string) (string, error) {
	normalized, err := Normalize(region)
	if err != nil {
		return "", err
	}
	if normalized == "" {
		return "", errors.New(missingRegionDetail(attributeName, envVar))
	}
	return normalized, nil
}

func NewQuerySuggestionsClient(appID, apiKey, region string) (*suggestions.APIClient, error) {
	normalized, err := Require(region, querySuggestionsConfig.attributeName, querySuggestionsConfig.envVar)
	if err != nil {
		return nil, fmt.Errorf("%s client requires a region: %w", querySuggestionsConfig.name, err)
	}

	var apiRegion suggestions.Region
	switch normalized {
	case "us":
		apiRegion = suggestions.US
	case "eu":
		apiRegion = suggestions.EU
	default:
		return nil, fmt.Errorf("%s region must be one of: %s", querySuggestionsConfig.name, validRegionValues)
	}

	return suggestions.NewClient(appID, apiKey, apiRegion)
}

func NewPersonalizationClient(appID, apiKey, region string) (*personalization.APIClient, error) {
	normalized, err := Require(region, personalizationConfig.attributeName, personalizationConfig.envVar)
	if err != nil {
		return nil, fmt.Errorf("%s client requires a region: %w", personalizationConfig.name, err)
	}

	var apiRegion personalization.Region
	switch normalized {
	case "us":
		apiRegion = personalization.US
	case "eu":
		apiRegion = personalization.EU
	default:
		return nil, fmt.Errorf("%s region must be one of: %s", personalizationConfig.name, validRegionValues)
	}

	return personalization.NewClient(appID, apiKey, apiRegion)
}

func missingRegionDetail(attributeName, envVar string) string {
	return fmt.Sprintf("set %s in the provider configuration or use the %s environment variable", attributeName, envVar)
}
