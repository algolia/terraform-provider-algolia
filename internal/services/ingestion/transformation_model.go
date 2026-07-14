package ingestion

import "github.com/hashicorp/terraform-plugin-framework/types"

// TransformationResourceModel describes the algolia_ingestion_transformation
// resource.
//
// A transformation's logic is expressed one of two ways:
//   - Code holds the transformation's source code directly, as a plain
//     string. This mirrors the API's deprecated top-level `code` field
//     (Transformation.Code/TransformationCreate.Code): the Ingestion API
//     always returns Code as a plain string (never redacted, never
//     JSON-wrapped), so it is refreshed on Read like any other plain string
//     attribute - no JSON encoding/decoding involved. It is empty for
//     no-code transformations.
//   - Input holds JSON-encoded configuration matching the TransformationInput
//     union (either {"code": "..."} or {"steps": [...]}), primarily used for
//     no-code transformations built from a series of steps. Like
//     Source/Destination's Input, GetTransformation returns it in full
//     (nothing is redacted), so Read refreshes it too - but only adopts the
//     API's encoding when it is not semantically equal to what's already
//     configured (ignoring object key order and the order of scalar arrays;
//     the order of object arrays like `steps` is significant and preserved),
//     to avoid a perpetual diff. See flattenTransformation/
//     flattenTransformationInput.
type TransformationResourceModel struct {
	ID                types.String `tfsdk:"id"`
	TransformationID  types.String `tfsdk:"transformation_id"`
	Name              types.String `tfsdk:"name"`
	Code              types.String `tfsdk:"code"`
	Type              types.String `tfsdk:"type"`
	Input             types.String `tfsdk:"input"`
	Description       types.String `tfsdk:"description"`
	AuthenticationIDs types.List   `tfsdk:"authentication_ids"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

// TransformationDataSourceModel describes the
// algolia_ingestion_transformation data source. Like the destination/source
// data sources, this includes Code and Input in full: the Ingestion API does
// not redact a transformation's configuration.
type TransformationDataSourceModel struct {
	TransformationID  types.String `tfsdk:"transformation_id"`
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Code              types.String `tfsdk:"code"`
	Type              types.String `tfsdk:"type"`
	Input             types.String `tfsdk:"input"`
	Description       types.String `tfsdk:"description"`
	AuthenticationIDs types.List   `tfsdk:"authentication_ids"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}
