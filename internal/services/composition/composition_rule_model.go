package composition

import "github.com/hashicorp/terraform-plugin-framework/types"

// CompositionRuleResourceModel describes the algolia_composition_rule
// resource: a per-object rule scoped to a composition, analogous to
// algolia_recommend_rule for Recommend and algolia_rule for Search Rules.
//
// Unlike algolia_recommend_rule's Condition (a single object), composition's
// own CompositionRule models its conditions as a list
// (Conditions []Condition), so `conditions` here is a JSON-encoded array
// rather than a single JSON object. Consequence/Validity are JSON-encoded
// strings for the same reason algolia_recommend_rule uses them: they are
// refreshed on Read using the semantic-equality preserve-prior pattern in
// json.go, to avoid a perpetual diff when the API echoes back a
// semantically identical encoding. Tags is a plain list of strings (not
// JSON-encoded), matching how internal/services/dictionary models simple
// string-list API fields.
type CompositionRuleResourceModel struct {
	ID            types.String `tfsdk:"id"`
	CompositionID types.String `tfsdk:"composition_id"`
	ObjectID      types.String `tfsdk:"object_id"`
	Conditions    types.String `tfsdk:"conditions"`
	Consequence   types.String `tfsdk:"consequence"`
	Description   types.String `tfsdk:"description"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	Validity      types.String `tfsdk:"validity"`
	Tags          types.List   `tfsdk:"tags"`
}

// CompositionRuleDataSourceModel describes the algolia_composition_rule data
// source. Same shape as the resource model; the data source has no prior
// configuration to preserve, so flattenCompositionRule always adopts the
// API's JSON encoding directly for it.
type CompositionRuleDataSourceModel = CompositionRuleResourceModel
