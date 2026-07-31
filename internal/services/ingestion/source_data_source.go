package ingestion

import (
	"context"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &sourceDataSource{}
	_ datasource.DataSourceWithConfigure = &sourceDataSource{}
)

// sourceDataSource reads an algolia_ingestion_source resource's
// configuration, including `input` in full: GetSource does not redact it
// (unlike GetAuthentication).
type sourceDataSource struct {
	base
}

// NewSourceDataSource returns the algolia_ingestion_source data source.
func NewSourceDataSource() datasource.DataSource {
	return &sourceDataSource{}
}

func (d *sourceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingestion_source"
}

func (d *sourceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = sourceDataSourceSchema()
}

func (d *sourceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(d.configure(req.ProviderData)...)
}

func (d *sourceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model SourceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := d.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourceID := model.SourceID.ValueString()
	tflog.Debug(ctx, "Reading Ingestion source data source", map[string]any{"source_id": sourceID})

	apiResp, err := client.GetSource(client.NewApiGetSourceRequest(sourceID), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ingestion source", "Could not read source "+sourceID+": "+algoliaerr.Explain(err))
		return
	}

	resp.Diagnostics.Append(flattenSourceDataSource(apiResp, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
