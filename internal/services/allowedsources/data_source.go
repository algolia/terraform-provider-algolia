package allowedsources

import (
	"context"
	"fmt"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &allowedSourcesDataSource{}
	_ datasource.DataSourceWithConfigure = &allowedSourcesDataSource{}
)

type allowedSourcesDataSource struct {
	client *search.APIClient
	appID  string
}

// NewDataSource returns the algolia_allowed_sources data source.
func NewDataSource() datasource.DataSource {
	return &allowedSourcesDataSource{}
}

func (d *allowedSourcesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_allowed_sources"
}

func (d *allowedSourcesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = allowedSourcesDataSourceSchema()
}

func (d *allowedSourcesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *allowedSourcesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading allowed sources data source", map[string]any{"app_id": d.appID})

	current, err := d.client.GetSources(search.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error reading allowed sources", err.Error())
		return
	}

	var model AllowedSourcesDataSourceModel
	model.ID = types.StringValue(d.appID)
	resp.Diagnostics.Append(flattenSources(ctx, current, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
