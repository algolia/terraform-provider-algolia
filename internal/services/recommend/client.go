// Package recommend implements the algolia_recommend_rule resource and data
// source, backed by the recommend client.
//
// Unlike Query Suggestions, Personalization, Ingestion, and A/B Testing, the
// Recommend API is not region-routed: recommendapi.NewClient(appID, apiKey)
// takes no region argument. Following the same on-demand-client convention
// as those packages (see internal/services/abtest/client.go), no
// *recommendapi.APIClient is stored on ProviderData - the resource and data
// source keep the app ID and API key from *providertypes.ProviderData and
// build a client on demand via base.client().
package recommend

import (
	"fmt"

	recommendapi "github.com/algolia/algoliasearch-client-go/v4/algolia/recommend"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// base holds the provider configuration shared by the algolia_recommend_rule
// resource and data source. Embed it to inherit Configure-time wiring
// (configure) and an on-demand client (client).
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

// client builds a Recommend API client on demand from the configuration
// captured by configure.
func (b *base) client() (*recommendapi.APIClient, diag.Diagnostics) {
	var diags diag.Diagnostics

	client, err := recommendapi.NewClient(b.appID, b.apiKey)
	if err != nil {
		diags.AddError("Unable to create Recommend client", err.Error())
		return nil, diags
	}

	return client, diags
}
