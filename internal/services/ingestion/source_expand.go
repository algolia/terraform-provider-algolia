package ingestion

import (
	"encoding/json"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// expandSourceCreate converts the Terraform plan into a SourceCreate
// request body for CreateSource.
func expandSourceCreate(model *SourceResourceModel) (*ingestionapi.SourceCreate, diag.Diagnostics) {
	var diags diag.Diagnostics

	input, inputDiags := expandSourceInput(model.Input)
	diags.Append(inputDiags...)
	if diags.HasError() {
		return nil, diags
	}

	create := ingestionapi.NewSourceCreate(
		ingestionapi.SourceType(model.Type.ValueString()),
		model.Name.ValueString(),
	)
	create.Input = input
	create.AuthenticationID = model.AuthenticationID.ValueStringPointer()

	return create, diags
}

// expandSourceUpdate converts the Terraform plan into a SourceUpdate
// request body for UpdateSource.
//
// SourceUpdate takes a *SourceUpdateInput - a distinct union type from
// SourceCreate's *SourceInput (compare model_source_input.go and
// model_source_update_input.go in the Go client) - so `input` is decoded
// into a different Go type depending on whether we're creating or
// updating: expandSourceInput for Create, expandSourceUpdateInput for
// Update. There is no `type` field on SourceUpdate at all: the Ingestion
// API gives no way to change a source's type after creation, which is why
// `type` is RequiresReplace in the resource schema.
func expandSourceUpdate(model *SourceResourceModel) (*ingestionapi.SourceUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics

	input, inputDiags := expandSourceUpdateInput(model.Input)
	diags.Append(inputDiags...)
	if diags.HasError() {
		return nil, diags
	}

	update := ingestionapi.NewSourceUpdate(ingestionapi.WithSourceUpdateName(model.Name.ValueString()))
	update.Input = input
	update.AuthenticationID = model.AuthenticationID.ValueStringPointer()

	return update, diags
}

// expandSourceInput JSON-decodes the `input` attribute into the SourceInput
// union type expected by SourceCreate. `input` is Optional: some source
// types (e.g. "push") need no configuration at all, so a null or empty
// value decodes to a nil *SourceInput, which SourceCreate's MarshalJSON
// simply omits from the request body.
func expandSourceInput(input types.String) (*ingestionapi.SourceInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	if input.IsNull() || input.IsUnknown() || input.ValueString() == "" {
		return nil, diags
	}

	var sourceInput ingestionapi.SourceInput
	if err := json.Unmarshal([]byte(input.ValueString()), &sourceInput); err != nil {
		diags.AddError(
			"Invalid input JSON",
			"The `input` attribute must be JSON-encoded configuration matching the source `type` "+
				"(e.g. jsonencode({ url = \"...\" }) for type \"csv\"). Failed to parse: "+err.Error(),
		)
		return nil, diags
	}

	return &sourceInput, diags
}

// expandSourceUpdateInput is the SourceUpdate counterpart of
// expandSourceInput: the update endpoint accepts SourceUpdateInput rather
// than SourceInput.
func expandSourceUpdateInput(input types.String) (*ingestionapi.SourceUpdateInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	if input.IsNull() || input.IsUnknown() || input.ValueString() == "" {
		return nil, diags
	}

	var sourceInput ingestionapi.SourceUpdateInput
	if err := json.Unmarshal([]byte(input.ValueString()), &sourceInput); err != nil {
		diags.AddError(
			"Invalid input JSON",
			"The `input` attribute must be JSON-encoded configuration matching the source `type`. "+
				"Failed to parse: "+err.Error(),
		)
		return nil, diags
	}

	return &sourceInput, diags
}
