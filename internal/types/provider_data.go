package types

import (
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
)

// ProviderData holds the configured Algolia clients for use by resources and data sources.
type ProviderData struct {
	Client      *search.APIClient
	AgentClient interface{} // *agent.Client — stored as interface{} to break import cycle
}
