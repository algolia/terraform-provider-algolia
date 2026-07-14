// Package abtest implements the algolia_ab_test resource and data source,
// backed by the abtesting-v3 client.
//
// The A/B Testing API is region-routed, like Query Suggestions,
// Personalization, and Ingestion (see internal/analyticsregion). Following
// that same convention, no *abtestingV3.APIClient is stored on
// ProviderData: the resource and data source keep the app ID, API key, and
// analytics region from *providertypes.ProviderData and build a client on
// demand via base.client().
package abtest

import (
	"fmt"

	abtestingapi "github.com/algolia/algoliasearch-client-go/v4/algolia/abtesting-v3"
	"github.com/algolia/terraform-provider-algolia/internal/analyticsregion"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// base holds the provider configuration shared by the algolia_ab_test
// resource and data source. Embed it to inherit Configure-time wiring
// (configure) and an on-demand client (client).
type base struct {
	appID           string
	apiKey          string
	analyticsRegion string
}

// configure extracts appID/apiKey/analyticsRegion from a Configure request's
// ProviderData, mirroring every other region-routed resource/data source
// (see querysuggestions, personalization, ingestion).
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
	b.analyticsRegion = data.AnalyticsRegion

	return diags
}

// client builds an A/B Testing API client on demand from the configuration
// captured by configure, validating the configured analytics region.
func (b *base) client() (*abtestingapi.APIClient, diag.Diagnostics) {
	var diags diag.Diagnostics

	client, err := analyticsregion.NewABTestingClient(b.appID, b.apiKey, b.analyticsRegion)
	if err != nil {
		diags.AddError("Unable to create A/B Testing client", err.Error())
		return nil, diags
	}

	return client, diags
}
