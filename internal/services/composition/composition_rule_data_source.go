package composition

import (
	"context"

	compositionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/composition"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &compositionRuleDataSource{}
	_ datasource.DataSourceWithConfigure = &compositionRuleDataSource{}
)

// compositionRuleDataSource reads an algolia_composition_rule. It embeds
// base (see client.go) for Configure-time wiring and an on-demand
// Composition client.
type compositionRuleDataSource struct {
	base
}

// NewRuleDataSource returns the algolia_composition_rule data source.
func NewRuleDataSource() datasource.DataSource {
	return &compositionRuleDataSource{}
}

func (d *compositionRuleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_composition_rule"
}

func (d *compositionRuleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = compositionRuleDataSourceSchema()
}

func (d *compositionRuleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(d.configure(req.ProviderData)...)
}

func (d *compositionRuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model CompositionRuleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := d.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	compositionID := model.CompositionID.ValueString()
	objectID := model.ObjectID.ValueString()
	tflog.Debug(ctx, "Reading composition rule data source", map[string]any{"composition_id": compositionID, "object_id": objectID})

	apiResp, err := client.GetRule(client.NewApiGetRuleRequest(compositionID, objectID), compositionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(compositionRuleSubject(compositionID, objectID).Message(algoliaerr.Read, err))
		return
	}

	resp.Diagnostics.Append(flattenCompositionRule(compositionID, apiResp, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
