package ingestion

import "github.com/hashicorp/terraform-plugin-framework/types"

// AuthenticationResourceModel describes the algolia_ingestion_authentication
// resource.
//
// Input holds JSON-encoded credentials matching Type (e.g. an "algolia"
// authentication expects {"appID": "...", "apiKey": "..."}). It is
// effectively write-only: Read/flattenAuthentication never overwrites it,
// because GetAuthentication redacts secret values in its response, and
// storing that redacted value would create a permanent diff against the
// real configured credentials. See the input attribute's schema
// description for the full rationale.
type AuthenticationResourceModel struct {
	ID               types.String `tfsdk:"id"`
	AuthenticationID types.String `tfsdk:"authentication_id"`
	Type             types.String `tfsdk:"type"`
	Name             types.String `tfsdk:"name"`
	Platform         types.String `tfsdk:"platform"`
	Input            types.String `tfsdk:"input"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

// AuthenticationDataSourceModel describes the algolia_ingestion_authentication
// data source. It has no Input field at all: the Ingestion API redacts
// secret values, so there is nothing meaningful to expose here. Use the
// algolia_ingestion_authentication resource to manage credentials.
type AuthenticationDataSourceModel struct {
	AuthenticationID types.String `tfsdk:"authentication_id"`
	ID               types.String `tfsdk:"id"`
	Type             types.String `tfsdk:"type"`
	Name             types.String `tfsdk:"name"`
	Platform         types.String `tfsdk:"platform"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}
