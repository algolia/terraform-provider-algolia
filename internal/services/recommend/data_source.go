package recommend

import (
	"context"

	recommendapi "github.com/algolia/algoliasearch-client-go/v4/algolia/recommend"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &recommendRuleDataSource{}
	_ datasource.DataSourceWithConfigure = &recommendRuleDataSource{}
)

// recommendRuleDataSource reads an algolia_recommend_rule. It embeds base
// (see client.go) for Configure-time wiring and an on-demand Recommend
// client.
type recommendRuleDataSource struct {
	base
}

// NewDataSource returns the algolia_recommend_rule data source.
func NewDataSource() datasource.DataSource {
	return &recommendRuleDataSource{}
}

func (d *recommendRuleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_recommend_rule"
}

func (d *recommendRuleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = recommendRuleDataSourceSchema()
}

func (d *recommendRuleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(d.configure(req.ProviderData)...)
}

func (d *recommendRuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model RecommendRuleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := d.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := model.IndexName.ValueString()
	recommendModel := recommendapi.RecommendModels(model.Model.ValueString())
	objectID := model.ObjectID.ValueString()
	tflog.Debug(ctx, "Reading Recommend rule data source", map[string]any{
		"index_name": indexName,
		"model":      string(recommendModel),
		"object_id":  objectID,
	})

	apiResp, err := client.GetRecommendRule(client.NewApiGetRecommendRuleRequest(indexName, recommendModel, objectID))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Recommend rule", "Could not read Recommend rule "+objectID+" on index "+indexName+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(flattenRecommendRule(indexName, string(recommendModel), apiResp, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
