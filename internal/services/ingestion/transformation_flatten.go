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
// Code is a plain string (never redacted, no key/array ordering ambiguity),
// so it's refreshed directly. When the API returns an empty code (a no-code
// transformation, or none set), the previously configured value is preserved
// rather than forced to null, so an explicitly configured `code = ""` doesn't
// perpetually diff against a null state; an unset `code` stays null.
func flattenTransformation(transformation *ingestionapi.Transformation, model *TransformationResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(transformation.TransformationID)
	model.TransformationID = types.StringValue(transformation.TransformationID)
	model.Name = types.StringValue(transformation.Name)
	model.Description = types.StringPointerValue(transformation.Description)
	model.CreatedAt = types.StringValue(transformation.CreatedAt)
	model.UpdatedAt = types.StringValue(transformation.UpdatedAt)

	// Captured before model.Code is overwritten: this is the value the caller
	// brought in - the configured one on Create/Update, the prior state on Read -
	// and it is what says whether this transformation's logic is expressed
	// through `code` rather than `input`.
	logicSuppliedAsCode := !model.Code.IsNull() && !model.Code.IsUnknown()

	model.Code = flattenCode(transformation.Code, model.Code)

	if transformation.Type != nil {
		model.Type = types.StringValue(string(*transformation.Type))
	} else {
		model.Type = types.StringNull()
	}

	authIDs, authIDsDiags := flattenAuthenticationIDs(transformation.AuthenticationIDs, model.AuthenticationIDs)
	diags.Append(authIDsDiags...)
	model.AuthenticationIDs = authIDs

	inputValue, inputDiags := flattenTransformationInput(transformation.Input, model.Input, logicSuppliedAsCode)
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
//
// logicSuppliedAsCode says the transformation's logic comes from `code`. The API
// derives an `input` from it and returns that, but adopting it would put a value
// into state for an attribute the configuration deliberately left unset, which
// Terraform rejects as an inconsistent apply result. `code` and `input` are
// mutually exclusive, so in that case the derived value is redundant with `code`
// and is dropped. An import has neither, so it still adopts what the API returns.
func flattenTransformationInput(input *ingestionapi.TransformationInput, previous types.String, logicSuppliedAsCode bool) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if input == nil {
		if previous.IsNull() || previous.IsUnknown() {
			return types.StringNull(), diags
		}

		return previous, diags
	}

	if logicSuppliedAsCode && (previous.IsNull() || previous.IsUnknown()) {
		return types.StringNull(), diags
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

	// The data source has no prior configuration to preserve, so an empty
	// authentication list maps to null.
	authIDs, authIDsDiags := flattenAuthenticationIDs(transformation.AuthenticationIDs, types.ListNull(types.StringType))
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

// flattenCode refreshes the plain-string `code`. A non-empty API value is
// adopted directly. When the API returns an empty code, an equivalently
// "empty" prior value is preserved so it doesn't perpetually diff: null stays
// null (unset), and an explicit "" stays "". If the prior value was a
// non-empty string but the API now has none, that's real drift (the code was
// cleared externally) and is surfaced as null.
func flattenCode(apiCode string, previous types.String) types.String {
	if apiCode != "" {
		return types.StringValue(apiCode)
	}
	if previous.IsNull() || previous.IsUnknown() || previous.ValueString() != "" {
		return types.StringNull()
	}
	return previous // explicit ""
}

// flattenAuthenticationIDs converts the API's []string into a Terraform List.
// A non-empty slice is adopted directly. When the API returns no IDs, an
// equivalently "empty" prior value is preserved: null stays null (unset), and
// an explicit empty list stays []. If the prior value was a non-empty list
// but the API now has none, that's real drift (associations cleared
// externally) and is surfaced as null. Pass a null previous (e.g. from the
// data source) to always map empty to null.
func flattenAuthenticationIDs(ids []string, previous types.List) (types.List, diag.Diagnostics) {
	if len(ids) == 0 {
		if !previous.IsNull() && !previous.IsUnknown() && len(previous.Elements()) == 0 {
			return previous, nil // explicit []
		}
		return types.ListNull(types.StringType), nil
	}

	values := make([]attr.Value, 0, len(ids))
	for _, id := range ids {
		values = append(values, types.StringValue(id))
	}

	return types.ListValue(types.StringType, values)
}
