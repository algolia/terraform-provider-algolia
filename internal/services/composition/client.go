// Package composition implements the algolia_composition and
// algolia_composition_rule resources and data sources, backed by the
// composition client.
//
// Like Recommend, the Composition API is not region-routed:
// compositionapi.NewClient(appID, apiKey) takes no region argument.
// Following the same on-demand-client convention as recommend (see
// internal/services/recommend/client.go; internal/services/abtest/client.go
// is the region-routed counterpart), no *compositionapi.APIClient is stored
// on ProviderData - the resources and data sources keep the app ID and API
// key from *providertypes.ProviderData and build a client on demand via
// base.client().
package composition

import (
	"fmt"

	compositionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/composition"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// base holds the provider configuration shared by the algolia_composition
// and algolia_composition_rule resources/data sources. Embed it to inherit
// Configure-time wiring (configure) and an on-demand client (client).
type base struct {
	appID  string
	apiKey string
}

// configure extracts appID/apiKey from a Configure request's ProviderData.
func (b *base) configure(providerData any) diag.Diagnostics {
	var diags diag.Diagnostics

	if providerData == nil {
		return diags
	}

	data, ok := providerData.(*providertypes.ProviderData)
	if !ok {
		diags.AddError(
			"Unexpected Configure Type",
			fmt.Sprintf("Expected *providertypes.ProviderData, got: %T", providerData),
		)
		return diags
	}

	b.appID = data.AppID
	b.apiKey = data.APIKey

	return diags
}

// client builds a Composition API client on demand from the configuration
// captured by configure.
func (b *base) client() (*compositionapi.APIClient, diag.Diagnostics) {
	var diags diag.Diagnostics

	client, err := compositionapi.NewClient(b.appID, b.apiKey)
	if err != nil {
		diags.AddError("Unable to create Composition client", err.Error())
		return nil, diags
	}

	return client, diags
}
