package synonym

import (
	"context"
	"fmt"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &synonymDataSource{}
	_ datasource.DataSourceWithConfigure = &synonymDataSource{}
)

type synonymDataSource struct {
	client *search.APIClient
}

func NewDataSource() datasource.DataSource {
	return &synonymDataSource{}
}

func (d *synonymDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_synonym"
}

func (d *synonymDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = synonymDataSourceSchema()
}

func (d *synonymDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
}

func (d *synonymDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model SynonymDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := model.IndexName.ValueString()
	objectID := model.ObjectID.ValueString()
	tflog.Debug(ctx, "Reading synonym data source", map[string]any{"index_name": indexName, "object_id": objectID})

	apiResp, err := d.client.GetSynonym(d.client.NewApiGetSynonymRequest(indexName, objectID), search.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error reading synonym", "Could not read synonym "+objectID+" on index "+indexName+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(hydrateSynonymModel(indexName, apiResp, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
