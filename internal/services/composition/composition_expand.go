package composition

import (
	"encoding/json"

	compositionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/composition"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// expandComposition converts the Terraform plan into a Composition for
// PutComposition.
func expandComposition(objectID string, model *CompositionResourceModel) (*compositionapi.Composition, diag.Diagnostics) {
	var diags diag.Diagnostics

	composition := compositionapi.NewEmptyComposition()
	composition.ObjectID = objectID
	composition.Name = model.Name.ValueString()

	if !model.Description.IsNull() && !model.Description.IsUnknown() {
		description := model.Description.ValueString()
		composition.Description = &description
	}

	if model.Behavior.IsNull() || model.Behavior.IsUnknown() || model.Behavior.ValueString() == "" {
		diags.AddError("Missing behavior", "The `behavior` attribute is required.")
		return nil, diags
	}
	var behavior compositionapi.CompositionBehavior
	if err := json.Unmarshal([]byte(model.Behavior.ValueString()), &behavior); err != nil {
		diags.AddError(
			"Invalid behavior JSON",
			"The `behavior` attribute must be a JSON-encoded object with either an `injection` or a `multifeed` "+
				"key (e.g. jsonencode({ injection = { main = { source = { search = { index = \"products\" } } } "+
				"} })). Failed to parse: "+err.Error(),
		)
		return nil, diags
	}
	composition.Behavior = behavior

	if !model.SortingStrategy.IsNull() && !model.SortingStrategy.IsUnknown() && model.SortingStrategy.ValueString() != "" {
		var sortingStrategy map[string]string
		if err := json.Unmarshal([]byte(model.SortingStrategy.ValueString()), &sortingStrategy); err != nil {
			diags.AddError(
				"Invalid sorting_strategy JSON",
				"The `sorting_strategy` attribute must be a JSON-encoded map of strings to strings (e.g. "+
					"jsonencode({ \"Price (asc)\" = \"products_price_asc\" })). Failed to parse: "+err.Error(),
			)
			return nil, diags
		}
		composition.SortingStrategy = &sortingStrategy
	}

	return composition, diags
}
