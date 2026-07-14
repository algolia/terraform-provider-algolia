package ingestion

import "github.com/hashicorp/terraform-plugin-framework/types"

// DestinationResourceModel describes the algolia_ingestion_destination
// resource.
//
// Input holds JSON-encoded configuration matching Type (e.g.
// {"indexName": "..."} for both "search" and "insights" destinations).
// Unlike SourceResourceModel's Input, which is Optional (some source types
// need no configuration at all), Destination's Input is Required: every
// destination writes to a specific indexName. Like Source, GetDestination
// returns Input in full - nothing is redacted - so Read does refresh this
// field, using the same semantic-equality preservation as
// flattenSourceInput to avoid a perpetual diff from harmless JSON
// differences (key order, array order). See
// flattenDestination/flattenDestinationInput.
type DestinationResourceModel struct {
	ID                types.String `tfsdk:"id"`
	DestinationID     types.String `tfsdk:"destination_id"`
	Type              types.String `tfsdk:"type"`
	Name              types.String `tfsdk:"name"`
	Input             types.String `tfsdk:"input"`
	AuthenticationID  types.String `tfsdk:"authentication_id"`
	TransformationIDs types.List   `tfsdk:"transformation_ids"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

// DestinationDataSourceModel describes the algolia_ingestion_destination
// data source. Like SourceDataSourceModel, this includes Input in full:
// the Ingestion API does not redact a destination's configuration.
type DestinationDataSourceModel struct {
	DestinationID     types.String `tfsdk:"destination_id"`
	ID                types.String `tfsdk:"id"`
	Type              types.String `tfsdk:"type"`
	Name              types.String `tfsdk:"name"`
	Input             types.String `tfsdk:"input"`
	AuthenticationID  types.String `tfsdk:"authentication_id"`
	TransformationIDs types.List   `tfsdk:"transformation_ids"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}
