package composition

import (
	compositionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/composition"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// flattenComposition copies a Composition (from GetComposition, or read back
// after PutComposition) into the Terraform model. behavior/sorting_strategy
// are refreshed using the semantic-equality preserve-prior pattern in
// json.go: model's existing value is kept as-is when it is semantically
// equal to the API's encoding, and only replaced when it actually differs.
// For the data source, and on import, model has no prior configuration for
// those fields (they start out null), so the API's encoding is always
// adopted.
func flattenComposition(apiResp *compositionapi.Composition, model *CompositionResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(apiResp.GetObjectID())
	model.ObjectID = types.StringValue(apiResp.GetObjectID())
	model.Name = types.StringValue(apiResp.GetName())

	if apiResp.Description != nil {
		model.Description = types.StringValue(apiResp.GetDescription())
	} else {
		model.Description = types.StringNull()
	}

	behaviorValue, behaviorDiags := flattenRequiredJSONField(apiResp.Behavior, model.Behavior, "behavior")
	diags.Append(behaviorDiags...)
	model.Behavior = behaviorValue

	sortingStrategyValue, sortingStrategyDiags := flattenOptionalJSONField(apiResp.SortingStrategy, model.SortingStrategy, "sorting_strategy")
	diags.Append(sortingStrategyDiags...)
	model.SortingStrategy = sortingStrategyValue

	return diags
}
