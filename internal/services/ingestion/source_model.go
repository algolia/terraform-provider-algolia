package ingestion

import "github.com/hashicorp/terraform-plugin-framework/types"

// SourceResourceModel describes the algolia_ingestion_source resource.
//
// Input holds JSON-encoded configuration matching Type (e.g. a "csv"
// source expects {"url": "..."}). Unlike AuthenticationResourceModel's
// Input, which GetAuthentication redacts and Read therefore never
// refreshes, GetSource returns a source's Input in full - nothing is
// redacted - so Read does refresh this field. To avoid a perpetual diff
// from harmless JSON differences (key order, array order), flattenSource
// only adopts the API's encoding when it is not semantically equal to the
// value already in state; otherwise it keeps the existing string as-is.
// See flattenSource/flattenSourceInput.
type SourceResourceModel struct {
	ID               types.String `tfsdk:"id"`
	SourceID         types.String `tfsdk:"source_id"`
	Type             types.String `tfsdk:"type"`
	Name             types.String `tfsdk:"name"`
	Input            types.String `tfsdk:"input"`
	AuthenticationID types.String `tfsdk:"authentication_id"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

// SourceDataSourceModel describes the algolia_ingestion_source data source.
// Unlike AuthenticationDataSourceModel, this does include Input: the
// Ingestion API does not redact a source's configuration.
type SourceDataSourceModel struct {
	SourceID         types.String `tfsdk:"source_id"`
	ID               types.String `tfsdk:"id"`
	Type             types.String `tfsdk:"type"`
	Name             types.String `tfsdk:"name"`
	Input            types.String `tfsdk:"input"`
	AuthenticationID types.String `tfsdk:"authentication_id"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}
