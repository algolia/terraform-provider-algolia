package ingestion

import (
	"encoding/json"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// flattenSource copies a GetSource response into the Terraform resource
// model.
//
// Unlike flattenAuthentication, this does refresh Input: GetSource returns
// a source's configuration in full - nothing is redacted. But naively
// overwriting model.Input with the API's JSON encoding on every
// Create/Read/Update would cause a perpetual diff whenever the API echoes
// back semantically identical JSON in a different form (key order, array
// order). So flattenSourceInput keeps the model's existing Input string
// as-is when it is semantically equal to what the API returned, and only
// adopts the API's encoding when it actually differs.
func flattenSource(source *ingestionapi.Source, model *SourceResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(source.SourceID)
	model.SourceID = types.StringValue(source.SourceID)
	model.Type = types.StringValue(string(source.Type))
	model.Name = types.StringValue(source.Name)
	model.AuthenticationID = types.StringPointerValue(source.AuthenticationID)
	model.CreatedAt = types.StringValue(source.CreatedAt)
	model.UpdatedAt = types.StringValue(source.UpdatedAt)

	inputValue, inputDiags := flattenSourceInput(source.Input, model.Input)
	diags.Append(inputDiags...)
	model.Input = inputValue

	return diags
}

// flattenSourceInput JSON-encodes the API's *SourceInput and decides
// whether to adopt it into state or keep the value already configured.
// previous is model.Input's value before this Create/Read/Update call -
// i.e. the plan's configured value (Create/Update) or the prior state
// (Read).
func flattenSourceInput(input *ingestionapi.SourceInput, previous types.String) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if input == nil {
		// The API returned no input at all (e.g. a "push" source, which has
		// no input shape). Only surface that as null if nothing was
		// configured either; otherwise keep whatever's already in
		// state/plan rather than clobbering a configured value with null.
		if previous.IsNull() || previous.IsUnknown() {
			return types.StringNull(), diags
		}

		return previous, diags
	}

	encoded, err := json.Marshal(input)
	if err != nil {
		diags.AddError("Error encoding source input", "Could not JSON-encode the source's input: "+err.Error())
		return previous, diags
	}
	apiValue := string(encoded)

	if !previous.IsNull() && !previous.IsUnknown() && jsonSemanticallyEqual(previous.ValueString(), apiValue) {
		return previous, diags
	}

	return types.StringValue(apiValue), diags
}

// flattenSourceDataSource is the data source counterpart of flattenSource.
// The data source has no prior configuration to preserve, so it always
// surfaces the API's JSON encoding of input verbatim.
func flattenSourceDataSource(source *ingestionapi.Source, model *SourceDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(source.SourceID)
	model.SourceID = types.StringValue(source.SourceID)
	model.Type = types.StringValue(string(source.Type))
	model.Name = types.StringValue(source.Name)
	model.AuthenticationID = types.StringPointerValue(source.AuthenticationID)
	model.CreatedAt = types.StringValue(source.CreatedAt)
	model.UpdatedAt = types.StringValue(source.UpdatedAt)

	if source.Input == nil {
		model.Input = types.StringNull()
		return diags
	}

	encoded, err := json.Marshal(source.Input)
	if err != nil {
		diags.AddError("Error encoding source input", "Could not JSON-encode the source's input: "+err.Error())
		return diags
	}
	model.Input = types.StringValue(string(encoded))

	return diags
}
