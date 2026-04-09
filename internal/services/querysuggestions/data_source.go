package querysuggestions

import (
	"context"
	"fmt"

	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &querySuggestionsDataSource{}
	_ datasource.DataSourceWithConfigure = &querySuggestionsDataSource{}
)

type querySuggestionsDataSource struct {
	appID                  string
	apiKey                 string
	querySuggestionsRegion string
}

func NewDataSource() datasource.DataSource {
	return &querySuggestionsDataSource{}
}

func (d *querySuggestionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_query_suggestions"
}

func (d *querySuggestionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = querySuggestionsDataSourceSchema()
}

func (d *querySuggestionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.appID = data.AppID
	d.apiKey = data.APIKey
	d.querySuggestionsRegion = data.QuerySuggestionsRegion
}

func (d *querySuggestionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model QuerySuggestionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resource := &querySuggestionsResource{
		appID:                  d.appID,
		apiKey:                 d.apiKey,
		querySuggestionsRegion: d.querySuggestionsRegion,
	}

	client, diags := resource.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := model.IndexName.ValueString()
	tflog.Debug(ctx, "Reading Query Suggestions data source", map[string]any{"region": d.querySuggestionsRegion, "index_name": indexName})

	apiResp, err := client.GetConfig(client.NewApiGetConfigRequest(indexName))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Query Suggestions config", "Could not read Query Suggestions config "+indexName+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(hydrateQuerySuggestionsModel(apiResp, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
