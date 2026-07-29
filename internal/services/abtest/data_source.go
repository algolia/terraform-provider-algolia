package abtest

import (
	"context"

	abtestingapi "github.com/algolia/algoliasearch-client-go/v4/algolia/abtesting-v3"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &abTestDataSource{}
	_ datasource.DataSourceWithConfigure = &abTestDataSource{}
)

// abTestDataSource reads an algolia_ab_test's current, enriched state
// (including runtime results). It embeds base (see client.go) for
// Configure-time wiring and an on-demand region-routed A/B Testing client.
type abTestDataSource struct {
	base
}

// NewDataSource returns the algolia_ab_test data source.
func NewDataSource() datasource.DataSource {
	return &abTestDataSource{}
}

func (d *abTestDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ab_test"
}

func (d *abTestDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = abTestDataSourceSchema()
}

func (d *abTestDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(d.configure(req.ProviderData)...)
}

func (d *abTestDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model ABTestDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := d.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	abTestID := int32(model.ABTestID.ValueInt64())
	tflog.Debug(ctx, "Reading A/B test data source", map[string]any{"ab_test_id": abTestID})

	apiResp, err := client.GetABTest(client.NewApiGetABTestRequest(abTestID), abtestingapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error reading A/B test", "Could not read A/B test: "+err.Error())
		return
	}

	resp.Diagnostics.Append(flattenABTestDataSource(apiResp, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
