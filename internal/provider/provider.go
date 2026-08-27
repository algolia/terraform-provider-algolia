package provider

import (
	"context"
	"os"

	agentStudio "github.com/algolia/algoliasearch-client-go/v4/algolia/agent-studio"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	frameworkschema "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/algolia/terraform-provider-algolia/internal/analyticsregion"
	"github.com/algolia/terraform-provider-algolia/internal/services/abtest"
	"github.com/algolia/terraform-provider-algolia/internal/services/agent"
	"github.com/algolia/terraform-provider-algolia/internal/services/agentprovider"
	"github.com/algolia/terraform-provider-algolia/internal/services/allowedsources"
	"github.com/algolia/terraform-provider-algolia/internal/services/apikey"
	"github.com/algolia/terraform-provider-algolia/internal/services/composition"
	"github.com/algolia/terraform-provider-algolia/internal/services/crawler"
	"github.com/algolia/terraform-provider-algolia/internal/services/dictionary"
	"github.com/algolia/terraform-provider-algolia/internal/services/index"
	"github.com/algolia/terraform-provider-algolia/internal/services/ingestion"
	"github.com/algolia/terraform-provider-algolia/internal/services/mcm"
	"github.com/algolia/terraform-provider-algolia/internal/services/personalization"
	"github.com/algolia/terraform-provider-algolia/internal/services/querysuggestions"
	"github.com/algolia/terraform-provider-algolia/internal/services/recommend"
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
	CrawlerUserID   types.String `tfsdk:"crawler_user_id"`
	CrawlerAPIKey   types.String `tfsdk:"crawler_api_key"`
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
		Description: "The Algolia provider allows you to manage Algolia resources.\n\n" +
			"Some resources handle credentials. Terraform can retain sensitive values in state " +
			"and saved plans, so protect those files and related CI artifacts.",
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
			// No crawler resource or data source exists, and none is planned: the
			// crawler was descoped on 2026-07-18 (see ROADMAP.md). These two
			// attributes therefore configure nothing. They are kept, deprecated
			// rather than removed, so that a configuration which already sets them
			// keeps planning instead of failing on an unknown attribute.
			"crawler_user_id": schema.StringAttribute{
				Description:        "Deprecated and unused. No crawler resource or data source exists, so setting this has no effect. Can also be set via the ALGOLIA_CRAWLER_USER_ID environment variable.",
				DeprecationMessage: "The crawler was descoped and no crawler resource or data source will be built, so crawler_user_id configures nothing. Remove it from your provider block.",
				Optional:           true,
			},
			"crawler_api_key": schema.StringAttribute{
				Description:        "Deprecated and unused. No crawler resource or data source exists, so setting this has no effect. Can also be set via the ALGOLIA_CRAWLER_API_KEY environment variable.",
				DeprecationMessage: "The crawler was descoped and no crawler resource or data source will be built, so crawler_api_key configures nothing. Remove it from your provider block.",
				Optional:           true,
				Sensitive:          true,
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
	crawlerUserID := os.Getenv("ALGOLIA_CRAWLER_USER_ID")
	crawlerAPIKey := os.Getenv("ALGOLIA_CRAWLER_API_KEY")

	if !config.AppID.IsNull() {
		appID = config.AppID.ValueString()
	}
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}
	if !config.AnalyticsRegion.IsNull() {
		analyticsRegion = config.AnalyticsRegion.ValueString()
	}
	if !config.CrawlerUserID.IsNull() {
		crawlerUserID = config.CrawlerUserID.ValueString()
	}
	if !config.CrawlerAPIKey.IsNull() {
		crawlerAPIKey = config.CrawlerAPIKey.ValueString()
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

	agentClient, err := agentStudio.NewClient(appID, apiKey)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create Algolia Agent Studio client",
			"An unexpected error occurred when creating the Algolia Agent Studio client: "+err.Error(),
		)
		return
	}

	// The crawler is optional: only the (future) crawler resource/data source needs these
	// credentials, so an empty crawler_user_id/crawler_api_key is not an error here — the
	// client is simply left nil and only constructed when both are set.
	var crawlerClient interface{}
	if crawlerUserID != "" && crawlerAPIKey != "" {
		crawlerClient = crawler.NewClient(crawlerUserID, crawlerAPIKey)
	}

	data := &providertypes.ProviderData{
		AppID:           appID,
		APIKey:          apiKey,
		AnalyticsRegion: normalizedAnalyticsRegion,
		Client:          client,
		AgentClient:     agentClient,
		CrawlerClient:   crawlerClient,
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
		allowedsources.NewResource,
		ingestion.NewAuthenticationResource,
		ingestion.NewSourceResource,
		ingestion.NewDestinationResource,
		ingestion.NewTransformationResource,
		ingestion.NewTaskResource,
		abtest.NewResource,
		recommend.NewResource,
		composition.NewResource,
		composition.NewRuleResource,
	}
}

func (p *algoliaProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		index.NewDataSource,
		index.NewVirtualDataSource,
		index.NewIndicesDataSource,
		agent.NewDataSource,
		agentprovider.NewDataSource,
		agentprovider.NewModelsDataSource,
		apikey.NewDataSource,
		apikey.NewKeysDataSource,
		rule.NewDataSource,
		synonym.NewDataSource,
		dictionary.NewDataSource,
		dictionary.NewSettingsDataSource,
		querysuggestions.NewDataSource,
		personalization.NewDataSource,
		allowedsources.NewDataSource,
		ingestion.NewAuthenticationDataSource,
		ingestion.NewSourceDataSource,
		ingestion.NewDestinationDataSource,
		ingestion.NewTransformationDataSource,
		ingestion.NewTaskDataSource,
		abtest.NewDataSource,
		recommend.NewDataSource,
		composition.NewDataSource,
		composition.NewRuleDataSource,
		mcm.NewClustersDataSource,
		mcm.NewUserIdsDataSource,
	}
}
