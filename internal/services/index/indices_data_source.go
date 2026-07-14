package index

import (
	"context"
	"fmt"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &indicesDataSource{}
	_ datasource.DataSourceWithConfigure = &indicesDataSource{}
)

type indicesDataSource struct {
	client *search.APIClient
	appID  string
}

// NewIndicesDataSource returns the algolia_indices data source.
func NewIndicesDataSource() datasource.DataSource {
	return &indicesDataSource{}
}

func (d *indicesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_indices"
}

func (d *indicesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = indicesDataSourceSchema()
}

func (d *indicesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = data.Client
	d.appID = data.AppID
}

func (d *indicesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading indices data source", map[string]any{"app_id": d.appID})

	items, err := fetchAllIndices(d.client)
	if err != nil {
		resp.Diagnostics.AddError("Error listing indices", "Could not list indices: "+err.Error())
		return
	}

	var model IndicesDataSourceModel
	resp.Diagnostics.Append(flattenIndicesDataSource(ctx, items, d.appID, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
