package ingestion

import (
	"encoding/json"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// expandDestinationCreate converts the Terraform plan into a
// DestinationCreate request body for CreateDestination.
func expandDestinationCreate(model *DestinationResourceModel) (*ingestionapi.DestinationCreate, diag.Diagnostics) {
	var diags diag.Diagnostics

	input, inputDiags := expandDestinationInput(model.Input)
	diags.Append(inputDiags...)
	if diags.HasError() {
		return nil, diags
	}

	create := ingestionapi.NewDestinationCreate(
		ingestionapi.DestinationType(model.Type.ValueString()),
		model.Name.ValueString(),
		input,
	)
	create.AuthenticationID = model.AuthenticationID.ValueStringPointer()
	create.TransformationIDs = expandTransformationIDs(model.TransformationIDs)

	return create, diags
}

// expandDestinationUpdate converts the Terraform plan into a
// DestinationUpdate request body for UpdateDestination.
//
// DestinationUpdate takes a *DestinationUpdateInput - a distinct type from
// DestinationCreate's DestinationInput (compare model_destination_input.go
// and model_destination_update_input.go in the Go client) - so `input` is
// decoded into a different Go type depending on whether we're creating or
// updating: expandDestinationInput for Create, expandDestinationUpdateInput
// for Update. There is no `type` field on DestinationUpdate at all: the
// Ingestion API gives no way to change a destination's type after
// creation, which is why `type` is RequiresReplace in the resource schema.
func expandDestinationUpdate(model *DestinationResourceModel) (*ingestionapi.DestinationUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics

	input, inputDiags := expandDestinationUpdateInput(model.Input)
	diags.Append(inputDiags...)
	if diags.HasError() {
		return nil, diags
	}

	update := ingestionapi.NewDestinationUpdate(ingestionapi.WithDestinationUpdateName(model.Name.ValueString()))
	update.Input = input
	update.AuthenticationID = model.AuthenticationID.ValueStringPointer()
	update.TransformationIDs = expandTransformationIDs(model.TransformationIDs)

	return update, diags
}

// expandDestinationInput JSON-decodes the `input` attribute into the
// DestinationInput type expected by DestinationCreate. Unlike
// expandSourceInput, `input` is Required on this resource - every
// destination writes to a specific indexName - so a null/unknown/empty
// value is a validation error rather than a legitimate "no configuration"
// case.
func expandDestinationInput(input types.String) (ingestionapi.DestinationInput, diag.Diagnostics) {
	var diags diag.Diagnostics
	var destinationInput ingestionapi.DestinationInput

	if input.IsNull() || input.IsUnknown() || input.ValueString() == "" {
		diags.AddError(
			"Missing input",
			"The `input` attribute is required for algolia_ingestion_destination (e.g. "+
				"jsonencode({ indexName = \"...\" })).",
		)
		return destinationInput, diags
	}

	if err := json.Unmarshal([]byte(input.ValueString()), &destinationInput); err != nil {
		diags.AddError(
			"Invalid input JSON",
			"The `input` attribute must be JSON-encoded configuration matching the destination `type` "+
				"(e.g. jsonencode({ indexName = \"...\" })). Failed to parse: "+err.Error(),
		)
		return destinationInput, diags
	}

	return destinationInput, diags
}

// expandDestinationUpdateInput is the DestinationUpdate counterpart of
// expandDestinationInput: the update endpoint accepts
// *DestinationUpdateInput rather than DestinationInput.
func expandDestinationUpdateInput(input types.String) (*ingestionapi.DestinationUpdateInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	if input.IsNull() || input.IsUnknown() || input.ValueString() == "" {
		diags.AddError(
			"Missing input",
			"The `input` attribute is required for algolia_ingestion_destination (e.g. "+
				"jsonencode({ indexName = \"...\" })).",
		)
		return nil, diags
	}

	var destinationInput ingestionapi.DestinationUpdateInput
	if err := json.Unmarshal([]byte(input.ValueString()), &destinationInput); err != nil {
		diags.AddError(
			"Invalid input JSON",
			"The `input` attribute must be JSON-encoded configuration matching the destination `type`. "+
				"Failed to parse: "+err.Error(),
		)
		return nil, diags
	}

	return &destinationInput, diags
}

// expandTransformationIDs converts the `transformation_ids` list attribute
// into a []string for the Ingestion API, mirroring
// internal/services/dictionary's expandStringList (not imported across
// service packages, per that package's own note on jsonSemanticallyEqual).
// A null or unknown list yields a nil slice, which the API client's
// MarshalJSON omits from the request body entirely (as opposed to an
// explicit empty list, which clears transformation_ids).
func expandTransformationIDs(list types.List) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	values := make([]string, 0, len(list.Elements()))
	for _, element := range list.Elements() {
		if value, ok := element.(types.String); ok && !value.IsNull() && !value.IsUnknown() {
			values = append(values, value.ValueString())
		}
	}

	return values
}
