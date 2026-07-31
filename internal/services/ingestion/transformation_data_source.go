package ingestion

import (
	"context"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &transformationDataSource{}
	_ datasource.DataSourceWithConfigure = &transformationDataSource{}
)

// transformationDataSource reads an algolia_ingestion_transformation
// resource's configuration, including `code` and `input` in full:
// GetTransformation does not redact either (like GetSource/GetDestination,
// unlike GetAuthentication).
type transformationDataSource struct {
	base
}

// NewTransformationDataSource returns the algolia_ingestion_transformation
// data source.
func NewTransformationDataSource() datasource.DataSource {
	return &transformationDataSource{}
}

func (d *transformationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingestion_transformation"
}

func (d *transformationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = transformationDataSourceSchema()
}

func (d *transformationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(d.configure(req.ProviderData)...)
}

func (d *transformationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model TransformationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := d.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	transformationID := model.TransformationID.ValueString()
	tflog.Debug(ctx, "Reading Ingestion transformation data source", map[string]any{"transformation_id": transformationID})

	apiResp, err := client.GetTransformation(client.NewApiGetTransformationRequest(transformationID), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ingestion transformation", "Could not read transformation "+transformationID+": "+algoliaerr.Explain(err))
		return
	}

	resp.Diagnostics.Append(flattenTransformationDataSource(apiResp, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
