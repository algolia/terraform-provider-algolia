package index

import (
	"context"
	"fmt"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &indexDataSource{}

// IndexDataSourceModel describes the algolia_index data source data model. It
// mirrors IndexResourceModel minus deletion_protection, which is a provider-side
// guard on destroy with no representation in the Algolia API and therefore
// nothing a read-only data source can report.
type IndexDataSourceModel struct {
	Name          types.String `tfsdk:"name"`
	Primary       types.String `tfsdk:"primary"`
	Entries       types.Int64  `tfsdk:"entries"`
	DataSize      types.Int64  `tfsdk:"data_size"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
	Attributes    types.Object `tfsdk:"attributes"`
	Ranking       types.Object `tfsdk:"ranking"`
	Faceting      types.Object `tfsdk:"faceting"`
	Highlighting  types.Object `tfsdk:"highlighting"`
	Pagination    types.Object `tfsdk:"pagination"`
	Typos         types.Object `tfsdk:"typos"`
	Languages     types.Object `tfsdk:"languages"`
	QueryStrategy types.Object `tfsdk:"query_strategy"`
	Performance   types.Object `tfsdk:"performance"`
	Advanced      types.Object `tfsdk:"advanced"`
}

type indexDataSource struct {
	client *search.APIClient
}

func NewDataSource() datasource.DataSource {
	return &indexDataSource{}
}

func (d *indexDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_index"
}

func (d *indexDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = indexDataSourceSchema()
}

func (d *indexDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = data.Client
}

func (d *indexDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config IndexDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := config.Name.ValueString()
	tflog.Debug(ctx, "Reading index data source", map[string]any{"name": indexName})

	// Read through the managed resource so the data source reports exactly what
	// the resource does, rather than only the settings: readIndex is what fills
	// entries, data_size, created_at and updated_at, which GetSettings does not
	// return.
	model := IndexResourceModel{Name: config.Name}
	found, diags := (&indexResource{client: d.client}).readIndex(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		// A data source names an index the configuration expects to exist, so its
		// absence is a configuration error rather than drift to be reconciled.
		resp.Diagnostics.AddError("Error reading index", "Could not read index "+indexName+": the index does not exist.")
		return
	}

	indexDataSourceFromIndexModel(model, &config)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func indexDataSourceFromIndexModel(indexModel IndexResourceModel, model *IndexDataSourceModel) {
	model.Name = indexModel.Name
	model.Primary = indexModel.Primary
	model.Entries = indexModel.Entries
	model.DataSize = indexModel.DataSize
	model.CreatedAt = indexModel.CreatedAt
	model.UpdatedAt = indexModel.UpdatedAt
	model.Attributes = indexModel.Attributes
	model.Ranking = indexModel.Ranking
	model.Faceting = indexModel.Faceting
	model.Highlighting = indexModel.Highlighting
	model.Pagination = indexModel.Pagination
	model.Typos = indexModel.Typos
	model.Languages = indexModel.Languages
	model.QueryStrategy = indexModel.QueryStrategy
	model.Performance = indexModel.Performance
	model.Advanced = indexModel.Advanced
}
