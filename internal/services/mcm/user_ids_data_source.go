package mcm

import (
	"context"
	"fmt"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &userIdsDataSource{}
	_ datasource.DataSourceWithConfigure = &userIdsDataSource{}
)

type userIdsDataSource struct {
	client *search.APIClient
	appID  string
}

// NewUserIdsDataSource returns the algolia_user_ids data source.
func NewUserIdsDataSource() datasource.DataSource {
	return &userIdsDataSource{}
}

func (d *userIdsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_ids"
}

func (d *userIdsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = userIdsDataSourceSchema()
}

func (d *userIdsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *userIdsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading user IDs data source", map[string]any{"app_id": d.appID})

	items, err := fetchAllUserIds(ctx, d.client)
	if err != nil {
		resp.Diagnostics.AddError("Error listing user IDs", "Could not list user IDs: "+err.Error())
		return
	}

	var model UserIdsDataSourceModel
	resp.Diagnostics.Append(flattenUserIdsDataSource(ctx, items, d.appID, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
