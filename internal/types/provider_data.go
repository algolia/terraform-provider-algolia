package types

import (
	agentStudio "github.com/algolia/algoliasearch-client-go/v4/algolia/agent-studio"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
)

// ProviderData holds the configured Algolia clients for use by resources and data sources.
type ProviderData struct {
	AppID           string
	APIKey          string
	AnalyticsRegion string
	Client          *search.APIClient
	AgentClient     *agentStudio.APIClient // shared Agent Studio SDK client
	CrawlerClient   interface{}            // *crawler.Client — shared Crawler API client stored as interface{} to break import cycles; nil when crawler credentials aren't configured
}
