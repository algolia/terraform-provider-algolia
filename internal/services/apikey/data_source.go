package apikey

import (
	"context"
	"errors"
	"fmt"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &apiKeyDataSource{}
	_ datasource.DataSourceWithConfigure = &apiKeyDataSource{}
)

type apiKeyDataSource struct {
	client *search.APIClient
}

// NewDataSource returns the algolia_api_key data source.
func NewDataSource() datasource.DataSource {
	return &apiKeyDataSource{}
}

func (d *apiKeyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (d *apiKeyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = apiKeyDataSourceSchema()
}

func (d *apiKeyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
}

func (d *apiKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model APIKeyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := model.Key.ValueString()
	// The key value is Sensitive; do not include it in logs.
	tflog.Debug(ctx, "Reading API key data source")

	apiResp, err := d.client.GetApiKey(d.client.NewApiGetApiKeyRequest(key))
	if err != nil {
		var apiErr *search.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			resp.Diagnostics.AddError(
				"API key not found",
				"No API key found for the provided value. Check that the key is correct and that the "+
					"credentials configured for the provider are allowed to read it.",
			)
			return
		}

		resp.Diagnostics.AddError("Error reading API key", "Could not read the API key: "+err.Error())
		return
	}

	resp.Diagnostics.Append(flattenAPIKeyDataSource(ctx, apiResp, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
