package dictionary

import (
	"context"
	"fmt"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &dictionaryEntryDataSource{}
	_ datasource.DataSourceWithConfigure = &dictionaryEntryDataSource{}
)

type dictionaryEntryDataSource struct {
	client *search.APIClient
}

func NewDataSource() datasource.DataSource {
	return &dictionaryEntryDataSource{}
}

func (d *dictionaryEntryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dictionary_entry"
}

func (d *dictionaryEntryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dictionaryEntryDataSourceSchema()
}

func (d *dictionaryEntryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *dictionaryEntryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model DictionaryEntryDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dictionaryType := search.DictionaryType(model.Dictionary.ValueString())
	objectID := model.ObjectID.ValueString()
	tflog.Debug(ctx, "Reading dictionary entry data source", map[string]any{"dictionary": string(dictionaryType), "object_id": objectID})

	entry, err := findDictionaryEntry(ctx, d.client, dictionaryType, objectID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading dictionary entry", "Could not read entry "+objectID+" in dictionary "+string(dictionaryType)+": "+err.Error())
		return
	}

	if entry == nil {
		resp.Diagnostics.AddError("Dictionary entry not found", "No entry with object_id "+objectID+" was found in dictionary "+string(dictionaryType)+".")
		return
	}

	resp.Diagnostics.Append(flattenDictionaryEntry(dictionaryType, entry, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
