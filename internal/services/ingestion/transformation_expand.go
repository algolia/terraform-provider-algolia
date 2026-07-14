package ingestion

import (
	"encoding/json"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// expandTransformationCreate converts the Terraform plan into a
// TransformationCreate request body.
//
// Unlike Source/Destination, whose update endpoints take a distinct
// *SourceUpdate/*DestinationUpdate type from their create bodies,
// UpdateTransformation's request body (NewApiUpdateTransformationRequest's
// transformationCreate parameter) is the very same *TransformationCreate
// type used by CreateTransformation - so this single function is reused for
// both Create and Update, rather than having a separate
// expandTransformationUpdate. This also means `type` is not RequiresReplace
// in the schema: the update endpoint accepts a `type` field just like
// create does.
func expandTransformationCreate(model *TransformationResourceModel) (*ingestionapi.TransformationCreate, diag.Diagnostics) {
	var diags diag.Diagnostics

	input, inputDiags := expandTransformationInput(model.Input)
	diags.Append(inputDiags...)
	if diags.HasError() {
		return nil, diags
	}

	create := ingestionapi.NewTransformationCreate(model.Name.ValueString())
	create.Code = model.Code.ValueStringPointer()
	create.Description = model.Description.ValueStringPointer()
	create.Input = input
	create.AuthenticationIDs = expandAuthenticationIDs(model.AuthenticationIDs)

	if !model.Type.IsNull() && !model.Type.IsUnknown() && model.Type.ValueString() != "" {
		transformationType := ingestionapi.TransformationType(model.Type.ValueString())
		create.Type = &transformationType
	}

	return create, diags
}

// expandTransformationInput JSON-decodes the `input` attribute into the
// TransformationInput union type expected by TransformationCreate. `input`
// is Optional: a transformation's logic can instead be supplied via the
// legacy `code` attribute, or omitted entirely if `code` is set - so a
// null/unknown/empty value decodes to a nil *TransformationInput, which
// TransformationCreate's MarshalJSON simply omits from the request body.
func expandTransformationInput(input types.String) (*ingestionapi.TransformationInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	if input.IsNull() || input.IsUnknown() || input.ValueString() == "" {
		return nil, diags
	}

	var transformationInput ingestionapi.TransformationInput
	if err := json.Unmarshal([]byte(input.ValueString()), &transformationInput); err != nil {
		diags.AddError(
			"Invalid input JSON",
			"The `input` attribute must be JSON-encoded configuration matching the transformation `type` "+
				"(e.g. jsonencode({ steps = [...] }) for a no-code transformation). Failed to parse: "+err.Error(),
		)
		return nil, diags
	}

	return &transformationInput, diags
}

// expandAuthenticationIDs converts the `authentication_ids` list attribute
// into a []string for the Ingestion API, mirroring destination_expand.go's
// expandTransformationIDs (same null/unknown-element handling, applied
// here to a transformation's associated authentications instead of a
// destination's transformations).
//
// A null or unknown list yields a nil slice, which the API client's
// MarshalJSON omits from the request body entirely (as opposed to an
// explicit empty list, which clears authentication_ids).
func expandAuthenticationIDs(list types.List) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	values := make([]string, 0, len(list.Elements()))
	for _, element := range list.Elements() {
		value, ok := element.(types.String)
		if !ok || value.IsNull() || value.IsUnknown() {
			// A null/unknown element (e.g. authentication_ids built from a
			// not-yet-known computed value) must not be silently dropped:
			// sending a partial list could unintentionally clear or alter the
			// transformation's authentications. Treat the whole list as
			// not-yet-resolved and omit the field instead.
			return nil
		}
		values = append(values, value.ValueString())
	}

	return values
}
