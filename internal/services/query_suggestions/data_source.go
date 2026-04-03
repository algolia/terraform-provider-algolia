package query_suggestions

import (
	"context"
	"fmt"

	suggestions "github.com/algolia/algoliasearch-client-go/v4/algolia/query-suggestions"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &querySuggestionsConfigDataSource{}

type querySuggestionsConfigDataSource struct {
	appID  string
	apiKey string
}

func NewDataSource() datasource.DataSource {
	return &querySuggestionsConfigDataSource{}
}

func (d *querySuggestionsConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_query_suggestions_config"
}

func (d *querySuggestionsConfigDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = querySuggestionsConfigDataSourceSchema()
}

func (d *querySuggestionsConfigDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*providertypes.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *providertypes.ProviderData, got: %T", req.ProviderData),
		)
		return
	}

	d.appID = data.AppID
	d.apiKey = data.APIKey
}

func (d *querySuggestionsConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model QuerySuggestionsConfigResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := model.IndexName.ValueString()
	region := model.Region.ValueString()
	tflog.Debug(ctx, "Reading query suggestions config data source", map[string]interface{}{"index_name": indexName, "region": region})

	client, err := suggestions.NewClient(d.appID, d.apiKey, suggestions.Region(region))
	if err != nil {
		resp.Diagnostics.AddError("Error creating Query Suggestions client", err.Error())
		return
	}

	apiResp, err := client.GetConfig(client.NewApiGetConfigRequest(indexName))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Query Suggestions config", "Could not read config for index "+indexName+": "+err.Error())
		return
	}

	savedRegion := model.Region

	resp.Diagnostics.Append(flattenConfigurationResponse(ctx, apiResp, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	model.Region = savedRegion
	model.DeletionProtection = types.BoolValue(true)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
