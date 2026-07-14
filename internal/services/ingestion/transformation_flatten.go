package ingestion

import (
	"encoding/json"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// flattenTransformation copies a GetTransformation response into the
// Terraform resource model.
//
// Code is refreshed directly: GetTransformation always returns it as a
// plain string (never redacted), with no key/array ordering ambiguity like
// JSON-encoded fields have, so there's no need for the semantic-equality
// dance flattenTransformationInput uses for `input`. An empty string (a
// no-code transformation has no legacy `code`) becomes null rather than "",
// so an unconfigured `code` attribute stays unconfigured.
func flattenTransformation(transformation *ingestionapi.Transformation, model *TransformationResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(transformation.TransformationID)
	model.TransformationID = types.StringValue(transformation.TransformationID)
	model.Name = types.StringValue(transformation.Name)
	model.Description = types.StringPointerValue(transformation.Description)
	model.CreatedAt = types.StringValue(transformation.CreatedAt)
	model.UpdatedAt = types.StringValue(transformation.UpdatedAt)

	if transformation.Code == "" {
		model.Code = types.StringNull()
	} else {
		model.Code = types.StringValue(transformation.Code)
	}

	if transformation.Type != nil {
		model.Type = types.StringValue(string(*transformation.Type))
	} else {
		model.Type = types.StringNull()
	}

	authIDs, authIDsDiags := flattenAuthenticationIDs(transformation.AuthenticationIDs)
	diags.Append(authIDsDiags...)
	model.AuthenticationIDs = authIDs

	inputValue, inputDiags := flattenTransformationInput(transformation.Input, model.Input)
	diags.Append(inputDiags...)
	model.Input = inputValue

	return diags
}

// flattenTransformationInput JSON-encodes the API's *TransformationInput and
// decides whether to adopt it into state or keep the value already
// configured. previous is model.Input's value before this
// Create/Read/Update call - i.e. the plan's configured value
// (Create/Update) or the prior state (Read). Mirrors flattenSourceInput:
// `input` is Optional, so a nil *TransformationInput (e.g. a transformation
// defined via the legacy `code` attribute instead) is only surfaced as null
// if nothing was configured either.
func flattenTransformationInput(input *ingestionapi.TransformationInput, previous types.String) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if input == nil {
		if previous.IsNull() || previous.IsUnknown() {
			return types.StringNull(), diags
		}

		return previous, diags
	}

	encoded, err := json.Marshal(input)
	if err != nil {
		diags.AddError("Error encoding transformation input", "Could not JSON-encode the transformation's input: "+err.Error())
		return previous, diags
	}
	apiValue := string(encoded)

	if !previous.IsNull() && !previous.IsUnknown() && jsonSemanticallyEqual(previous.ValueString(), apiValue) {
		return previous, diags
	}

	return types.StringValue(apiValue), diags
}

// flattenTransformationDataSource is the data source counterpart of
// flattenTransformation. The data source has no prior configuration to
// preserve, so it always surfaces the API's JSON encoding of input
// verbatim.
func flattenTransformationDataSource(transformation *ingestionapi.Transformation, model *TransformationDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(transformation.TransformationID)
	model.TransformationID = types.StringValue(transformation.TransformationID)
	model.Name = types.StringValue(transformation.Name)
	model.Description = types.StringPointerValue(transformation.Description)
	model.CreatedAt = types.StringValue(transformation.CreatedAt)
	model.UpdatedAt = types.StringValue(transformation.UpdatedAt)

	if transformation.Code == "" {
		model.Code = types.StringNull()
	} else {
		model.Code = types.StringValue(transformation.Code)
	}

	if transformation.Type != nil {
		model.Type = types.StringValue(string(*transformation.Type))
	} else {
		model.Type = types.StringNull()
	}

	authIDs, authIDsDiags := flattenAuthenticationIDs(transformation.AuthenticationIDs)
	diags.Append(authIDsDiags...)
	model.AuthenticationIDs = authIDs

	if transformation.Input == nil {
		model.Input = types.StringNull()
		return diags
	}

	encoded, err := json.Marshal(transformation.Input)
	if err != nil {
		diags.AddError("Error encoding transformation input", "Could not JSON-encode the transformation's input: "+err.Error())
		return diags
	}
	model.Input = types.StringValue(string(encoded))

	return diags
}

// flattenAuthenticationIDs converts the API's []string into a Terraform
// List, mirroring destination_flatten.go's flattenTransformationIDs (same
// nil/empty-slice -> null-list handling, applied here to a transformation's
// associated authentications instead of a destination's transformations).
func flattenAuthenticationIDs(ids []string) (types.List, diag.Diagnostics) {
	if len(ids) == 0 {
		return types.ListNull(types.StringType), nil
	}

	values := make([]attr.Value, 0, len(ids))
	for _, id := range ids {
		values = append(values, types.StringValue(id))
	}

	return types.ListValue(types.StringType, values)
}
