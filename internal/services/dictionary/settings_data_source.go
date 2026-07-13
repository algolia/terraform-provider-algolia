package dictionary

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
	_ datasource.DataSource              = &dictionarySettingsDataSource{}
	_ datasource.DataSourceWithConfigure = &dictionarySettingsDataSource{}
)

type dictionarySettingsDataSource struct {
	client *search.APIClient
	appID  string
}

// NewSettingsDataSource returns the algolia_dictionary_settings data source.
func NewSettingsDataSource() datasource.DataSource {
	return &dictionarySettingsDataSource{}
}

func (d *dictionarySettingsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dictionary_settings"
}

func (d *dictionarySettingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dictionarySettingsDataSourceSchema()
}

func (d *dictionarySettingsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *dictionarySettingsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading dictionary settings data source", map[string]any{"app_id": d.appID})

	current, err := d.client.GetDictionarySettings()
	if err != nil {
		resp.Diagnostics.AddError("Error reading dictionary settings", err.Error())
		return
	}

	var model DictionarySettingsDataSourceModel
	model.ID = types.StringValue(d.appID)
	resp.Diagnostics.Append(flattenDictionarySettings(ctx, current.GetDisableStandardEntries(), &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
