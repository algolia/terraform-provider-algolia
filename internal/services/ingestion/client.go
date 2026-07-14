// Package ingestion contains the shared plumbing for the Algolia Ingestion
// API (authentications, sources, destinations, transformations, and tasks).
//
// The Ingestion API is region-routed, like Query Suggestions and
// Personalization (see internal/analyticsregion). Following that same
// convention, no *ingestion.APIClient is stored on ProviderData: each
// resource/data source keeps the app ID, API key, and analytics region from
// *providertypes.ProviderData and builds a client on demand via base.client().
package ingestion

import (
	"fmt"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/algolia/terraform-provider-algolia/internal/analyticsregion"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// base holds the provider configuration shared by every Ingestion resource
// and data source. Embed it to inherit Configure-time wiring (configure) and
// an on-demand client (client), instead of re-declaring appID/apiKey/
// analyticsRegion and their plumbing in each of the five resources.
type base struct {
	appID           string
	apiKey          string
	analyticsRegion string
}

// configure extracts appID/apiKey/analyticsRegion from a Configure request's
// ProviderData. It mirrors the Configure method used by every other
// region-routed resource/data source (see querysuggestions, personalization),
// factored out so each Ingestion resource/data source only needs:
//
//	func (r *fooResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
//	    resp.Diagnostics.Append(r.configure(req.ProviderData)...)
//	}
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

// client builds an Ingestion API client on demand from the configuration
// captured by configure, validating the configured analytics region.
func (b *base) client() (*ingestionapi.APIClient, diag.Diagnostics) {
	var diags diag.Diagnostics

	client, err := analyticsregion.NewIngestionClient(b.appID, b.apiKey, b.analyticsRegion)
	if err != nil {
		diags.AddError("Unable to create Ingestion client", err.Error())
		return nil, diags
	}

	return client, diags
}
