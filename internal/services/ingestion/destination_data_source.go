package ingestion

import (
	"context"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &destinationDataSource{}
	_ datasource.DataSourceWithConfigure = &destinationDataSource{}
)

// destinationDataSource reads an algolia_ingestion_destination resource's
// configuration, including `input` in full: GetDestination does not redact
// it (like GetSource, unlike GetAuthentication).
type destinationDataSource struct {
	base
}

// NewDestinationDataSource returns the algolia_ingestion_destination data
// source.
func NewDestinationDataSource() datasource.DataSource {
	return &destinationDataSource{}
}

func (d *destinationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingestion_destination"
}

func (d *destinationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = destinationDataSourceSchema()
}

func (d *destinationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(d.configure(req.ProviderData)...)
}

func (d *destinationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model DestinationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := d.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	destinationID := model.DestinationID.ValueString()
	tflog.Debug(ctx, "Reading Ingestion destination data source", map[string]any{"destination_id": destinationID})

	apiResp, err := client.GetDestination(client.NewApiGetDestinationRequest(destinationID), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ingestion destination", "Could not read destination "+destinationID+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(flattenDestinationDataSource(apiResp, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
