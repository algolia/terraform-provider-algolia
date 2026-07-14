package ingestion

import (
	"encoding/json"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// flattenDestination copies a GetDestination response into the Terraform
// resource model.
//
// Like flattenSource, Input is refreshed on Read - GetDestination returns
// a destination's configuration in full, nothing is redacted - but naively
// overwriting model.Input with the API's JSON encoding on every
// Create/Read/Update would cause a perpetual diff whenever the API echoes
// back semantically identical JSON in a different form (key order, array
// order). So flattenDestinationInput keeps the model's existing Input
// string as-is when it is semantically equal to what the API returned, and
// only adopts the API's encoding when it actually differs. Unlike Source's
// Input (a pointer that can be nil for source types needing no
// configuration), Destination's Input is a plain value - the API always
// returns one - so there is no "missing input" branch to handle here.
func flattenDestination(destination *ingestionapi.Destination, model *DestinationResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(destination.DestinationID)
	model.DestinationID = types.StringValue(destination.DestinationID)
	model.Type = types.StringValue(string(destination.Type))
	model.Name = types.StringValue(destination.Name)
	model.AuthenticationID = types.StringPointerValue(destination.AuthenticationID)
	model.CreatedAt = types.StringValue(destination.CreatedAt)
	model.UpdatedAt = types.StringValue(destination.UpdatedAt)

	transformationIDs, tIDsDiags := flattenTransformationIDs(destination.TransformationIDs)
	diags.Append(tIDsDiags...)
	model.TransformationIDs = transformationIDs

	inputValue, inputDiags := flattenDestinationInput(destination.Input, model.Input)
	diags.Append(inputDiags...)
	model.Input = inputValue

	return diags
}

// flattenDestinationInput JSON-encodes the API's DestinationInput and
// decides whether to adopt it into state or keep the value already
// configured. previous is model.Input's value before this
// Create/Read/Update call - i.e. the plan's configured value
// (Create/Update) or the prior state (Read).
func flattenDestinationInput(input ingestionapi.DestinationInput, previous types.String) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	encoded, err := json.Marshal(input)
	if err != nil {
		diags.AddError("Error encoding destination input", "Could not JSON-encode the destination's input: "+err.Error())
		return previous, diags
	}
	apiValue := string(encoded)

	if !previous.IsNull() && !previous.IsUnknown() && jsonSemanticallyEqual(previous.ValueString(), apiValue) {
		return previous, diags
	}

	return types.StringValue(apiValue), diags
}

// flattenDestinationDataSource is the data source counterpart of
// flattenDestination. The data source has no prior configuration to
// preserve, so it always surfaces the API's JSON encoding of input
// verbatim.
func flattenDestinationDataSource(destination *ingestionapi.Destination, model *DestinationDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(destination.DestinationID)
	model.DestinationID = types.StringValue(destination.DestinationID)
	model.Type = types.StringValue(string(destination.Type))
	model.Name = types.StringValue(destination.Name)
	model.AuthenticationID = types.StringPointerValue(destination.AuthenticationID)
	model.CreatedAt = types.StringValue(destination.CreatedAt)
	model.UpdatedAt = types.StringValue(destination.UpdatedAt)

	transformationIDs, tIDsDiags := flattenTransformationIDs(destination.TransformationIDs)
	diags.Append(tIDsDiags...)
	model.TransformationIDs = transformationIDs

	encoded, err := json.Marshal(destination.Input)
	if err != nil {
		diags.AddError("Error encoding destination input", "Could not JSON-encode the destination's input: "+err.Error())
		return diags
	}
	model.Input = types.StringValue(string(encoded))

	return diags
}

// flattenTransformationIDs converts the API's []string into a Terraform
// List, mirroring internal/services/dictionary's flattenStringList: a nil
// or empty slice becomes a null list rather than an empty list value.
func flattenTransformationIDs(ids []string) (types.List, diag.Diagnostics) {
	if len(ids) == 0 {
		return types.ListNull(types.StringType), nil
	}

	values := make([]attr.Value, 0, len(ids))
	for _, id := range ids {
		values = append(values, types.StringValue(id))
	}

	return types.ListValue(types.StringType, values)
}
