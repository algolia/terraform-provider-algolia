package index

import (
	"context"
	"fmt"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

var (
	_ datasource.DataSource              = &virtualIndexDataSource{}
	_ datasource.DataSourceWithConfigure = &virtualIndexDataSource{}
)

type virtualIndexDataSource struct {
	client *search.APIClient
}

func NewVirtualDataSource() datasource.DataSource {
	return &virtualIndexDataSource{}
}

func (d *virtualIndexDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_index"
}

func (d *virtualIndexDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = virtualIndexDataSourceSchema()
}

func (d *virtualIndexDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *virtualIndexDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config VirtualIndexDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resource := &virtualIndexResource{client: d.client}
	indexModel := IndexResourceModel{Name: config.Name}
	found, diags := resource.readIndexModel(ctx, &indexModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		// A data source names an index the configuration expects to exist, so its
		// absence is a configuration error rather than drift to be reconciled.
		resp.Diagnostics.AddError("Error reading index", "Could not read index "+config.Name.ValueString()+": the index does not exist.")
		return
	}

	virtualDataSourceFromIndexModel(indexModel, &config)
	if config.PrimaryIndexName.IsNull() || config.PrimaryIndexName.ValueString() == "" {
		resp.Diagnostics.AddError("Index is not a virtual replica", "The requested index does not report a primary index and cannot be read as algolia_virtual_index.")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
