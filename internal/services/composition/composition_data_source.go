package composition

import (
	"context"

	compositionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/composition"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &compositionDataSource{}
	_ datasource.DataSourceWithConfigure = &compositionDataSource{}
)

// compositionDataSource reads an algolia_composition. It embeds base (see
// client.go) for Configure-time wiring and an on-demand Composition client.
type compositionDataSource struct {
	base
}

// NewDataSource returns the algolia_composition data source.
func NewDataSource() datasource.DataSource {
	return &compositionDataSource{}
}

func (d *compositionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_composition"
}

func (d *compositionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = compositionDataSourceSchema()
}

func (d *compositionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(d.configure(req.ProviderData)...)
}

func (d *compositionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model CompositionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := d.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	objectID := model.ObjectID.ValueString()
	tflog.Debug(ctx, "Reading composition data source", map[string]any{"object_id": objectID})

	apiResp, err := client.GetComposition(client.NewApiGetCompositionRequest(objectID), compositionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(algoliaerr.Object(compositionKind, objectID).Message(algoliaerr.Read, err))
		return
	}

	resp.Diagnostics.Append(flattenComposition(apiResp, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
