package types

import (
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
)

// ProviderData holds the configured Algolia clients for use by resources and data sources.
type ProviderData struct {
	AppID             string
	APIKey            string
	AnalyticsRegion   string
	Client            *search.APIClient
	AgentClient       interface{} // *agent.Client — shared Agent Studio client stored as interface{} to break import cycles
	CollectionsClient interface{} // *collection.Client — shared Collections API client stored as interface{} to break import cycles
}
