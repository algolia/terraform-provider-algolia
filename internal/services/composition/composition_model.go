package composition

import "github.com/hashicorp/terraform-plugin-framework/types"

// CompositionResourceModel describes the algolia_composition resource.
//
// Behavior/SortingStrategy are JSON-encoded strings rather than nested
// blocks, following the same convention as algolia_recommend_rule for
// complex nested API types. They are refreshed on Read using the
// semantic-equality preserve-prior pattern in json.go, to avoid a perpetual
// diff when the API echoes back a semantically identical encoding (e.g. a
// different key or array order).
type CompositionResourceModel struct {
	ID              types.String `tfsdk:"id"`
	ObjectID        types.String `tfsdk:"object_id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	Behavior        types.String `tfsdk:"behavior"`
	SortingStrategy types.String `tfsdk:"sorting_strategy"`
}

// CompositionDataSourceModel describes the algolia_composition data source.
// Same shape as the resource model; the data source has no prior
// configuration to preserve, so flattenComposition always adopts the API's
// JSON encoding directly for it.
type CompositionDataSourceModel = CompositionResourceModel
