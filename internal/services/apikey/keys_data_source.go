package apikey

import (
	"context"
	"fmt"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &apiKeysDataSource{}
	_ datasource.DataSourceWithConfigure = &apiKeysDataSource{}
)

type apiKeysDataSource struct {
	client *search.APIClient
	appID  string
}

// NewKeysDataSource returns the algolia_api_keys data source.
func NewKeysDataSource() datasource.DataSource {
	return &apiKeysDataSource{}
}

func (d *apiKeysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_keys"
}

func (d *apiKeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = apiKeysDataSourceSchema()
}

func (d *apiKeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*providertypes.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *providertypes.ProviderData, got: %T", req.ProviderData),
		)
		return
	}

	d.client = data.Client
	d.appID = data.AppID
}

func (d *apiKeysDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading API keys data source", map[string]any{"app_id": d.appID})

	apiResp, err := d.client.ListApiKeys()
	if err != nil {
		resp.Diagnostics.AddError("Error listing API keys", "Could not list API keys: "+err.Error())
		return
	}

	var model APIKeysDataSourceModel
	resp.Diagnostics.Append(flattenAPIKeysDataSource(ctx, apiResp, d.appID, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
