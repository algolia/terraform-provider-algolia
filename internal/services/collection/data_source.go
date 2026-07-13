package collection

import (
	"context"
	"fmt"

	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &collectionDataSource{}

type collectionDataSource struct {
	client *Client
}

func NewDataSource() datasource.DataSource {
	return &collectionDataSource{}
}

func (d *collectionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_collection"
}

func (d *collectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = collectionDataSourceSchema()
}

func (d *collectionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*providertypes.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *providertypes.ProviderData, got: %T", req.ProviderData),
		)
		return
	}

	client, ok := data.CollectionsClient.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Collections Client Type",
			fmt.Sprintf("Expected *collection.Client, got: %T", data.CollectionsClient),
		)
		return
	}

	d.client = client
}

func (d *collectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model CollectionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := model.ID.ValueString()
	tflog.Debug(ctx, "Reading collection data source", map[string]interface{}{"id": id})

	apiResp, err := d.client.GetCollection(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading collection", "Could not read collection "+id+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(flattenCollectionResponseForDataSource(ctx, apiResp, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
