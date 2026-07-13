package provider

import (
	"context"
	"os"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	frameworkschema "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/algolia/terraform-provider-algolia/internal/analyticsregion"
	"github.com/algolia/terraform-provider-algolia/internal/services/agent"
	"github.com/algolia/terraform-provider-algolia/internal/services/agentprovider"
	"github.com/algolia/terraform-provider-algolia/internal/services/apikey"
	"github.com/algolia/terraform-provider-algolia/internal/services/dictionary"
	"github.com/algolia/terraform-provider-algolia/internal/services/index"
	"github.com/algolia/terraform-provider-algolia/internal/services/personalization"
	"github.com/algolia/terraform-provider-algolia/internal/services/querysuggestions"
	"github.com/algolia/terraform-provider-algolia/internal/services/rule"
	"github.com/algolia/terraform-provider-algolia/internal/services/synonym"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
)

var _ provider.Provider = &algoliaProvider{}

type algoliaProvider struct {
	version string
}

type algoliaProviderModel struct {
	AppID           types.String `tfsdk:"app_id"`
	APIKey          types.String `tfsdk:"api_key"`
	AnalyticsRegion types.String `tfsdk:"analytics_region"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &algoliaProvider{
			version: version,
		}
	}
}

func (p *algoliaProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "algolia"
	resp.Version = p.version
}

func (p *algoliaProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Algolia provider allows you to manage Algolia resources.",
		Attributes: map[string]schema.Attribute{
			"app_id": schema.StringAttribute{
				Description: "The Algolia Application ID. Can also be set via the ALGOLIA_APP_ID environment variable.",
				Optional:    true,
			},
			"api_key": schema.StringAttribute{
				Description: "The Algolia Admin API Key. Can also be set via the ALGOLIA_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"analytics_region": schema.StringAttribute{
				Description: "Analytics region for Algolia APIs that require regional routing, such as Query Suggestions and Personalization. Can also be set via the ALGOLIA_ANALYTICS_REGION environment variable.",
				Optional:    true,
				Validators: []frameworkschema.String{
					stringvalidator.OneOf("us", "eu"),
				},
			},
		},
	}
}

func (p *algoliaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config algoliaProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := os.Getenv("ALGOLIA_APP_ID")
	apiKey := os.Getenv("ALGOLIA_API_KEY")
	analyticsRegion := os.Getenv(analyticsregion.EnvVar)

	if !config.AppID.IsNull() {
		appID = config.AppID.ValueString()
	}
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}
	if !config.AnalyticsRegion.IsNull() {
		analyticsRegion = config.AnalyticsRegion.ValueString()
	}

	if appID == "" {
		resp.Diagnostics.AddError(
			"Missing Algolia App ID",
			"The provider cannot create the Algolia client as there is a missing or empty value for the Algolia App ID. "+
				"Set the app_id value in the configuration or use the ALGOLIA_APP_ID environment variable.",
		)
	}
	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing Algolia API Key",
			"The provider cannot create the Algolia client as there is a missing or empty value for the Algolia API Key. "+
				"Set the api_key value in the configuration or use the ALGOLIA_API_KEY environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	normalizedAnalyticsRegion, err := analyticsregion.Normalize(analyticsRegion)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid analytics region",
			err.Error(),
		)
		return
	}

	client, err := search.NewClient(appID, apiKey)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create Algolia client",
			"An unexpected error occurred when creating the Algolia client: "+err.Error(),
		)
		return
	}

	agentClient := agent.NewClient(appID, apiKey)

	data := &providertypes.ProviderData{
		AppID:           appID,
		APIKey:          apiKey,
		AnalyticsRegion: normalizedAnalyticsRegion,
		Client:          client,
		AgentClient:     agentClient,
	}

	resp.ResourceData = data
	resp.DataSourceData = data
}

func (p *algoliaProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		index.NewResource,
		agent.NewResource,
		agentprovider.NewResource,
		apikey.NewResource,
		rule.NewResource,
		synonym.NewResource,
		dictionary.NewResource,
		dictionary.NewSettingsResource,
		index.NewVirtualResource,
		querysuggestions.NewResource,
		personalization.NewResource,
	}
}

func (p *algoliaProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		index.NewDataSource,
		index.NewVirtualDataSource,
		agent.NewDataSource,
		agentprovider.NewDataSource,
		agentprovider.NewModelsDataSource,
		rule.NewDataSource,
		synonym.NewDataSource,
		dictionary.NewDataSource,
		dictionary.NewSettingsDataSource,
		querysuggestions.NewDataSource,
		personalization.NewDataSource,
	}
}
