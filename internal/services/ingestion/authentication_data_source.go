package ingestion

import (
	"context"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &authenticationDataSource{}
	_ datasource.DataSourceWithConfigure = &authenticationDataSource{}
)

// authenticationDataSource reads an algolia_ingestion_authentication
// resource's metadata. It never exposes `input`: GetAuthentication redacts
// secret values, so there is nothing meaningful to return.
type authenticationDataSource struct {
	base
}

// NewAuthenticationDataSource returns the algolia_ingestion_authentication data source.
func NewAuthenticationDataSource() datasource.DataSource {
	return &authenticationDataSource{}
}

func (d *authenticationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingestion_authentication"
}

func (d *authenticationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = authenticationDataSourceSchema()
}

func (d *authenticationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(d.configure(req.ProviderData)...)
}

func (d *authenticationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model AuthenticationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := d.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	authenticationID := model.AuthenticationID.ValueString()
	tflog.Debug(ctx, "Reading Ingestion authentication data source", map[string]any{"authentication_id": authenticationID})

	apiResp, err := client.GetAuthentication(client.NewApiGetAuthenticationRequest(authenticationID), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ingestion authentication", "Could not read authentication "+authenticationID+": "+algoliaerr.Explain(err))
		return
	}

	resp.Diagnostics.Append(flattenAuthenticationDataSource(apiResp, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
