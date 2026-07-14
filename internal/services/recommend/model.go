package recommend

import "github.com/hashicorp/terraform-plugin-framework/types"

// RecommendRuleResourceModel describes the algolia_recommend_rule resource.
//
// Condition/Consequence/Validity are JSON-encoded strings rather than nested
// blocks (unlike algolia_rule, which models the equivalent Search Rule
// concepts as nested blocks). They are refreshed on Read using the same
// semantic-equality preserve-prior pattern as the Ingestion package's
// JSON-encoded attributes (see json.go and flatten.go), to avoid a perpetual
// diff when the API echoes back a semantically identical encoding (e.g. a
// different key or array order).
type RecommendRuleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	IndexName   types.String `tfsdk:"index_name"`
	Model       types.String `tfsdk:"model"`
	ObjectID    types.String `tfsdk:"object_id"`
	Condition   types.String `tfsdk:"condition"`
	Consequence types.String `tfsdk:"consequence"`
	Description types.String `tfsdk:"description"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Validity    types.String `tfsdk:"validity"`
}

// RecommendRuleDataSourceModel describes the algolia_recommend_rule data
// source. Same shape as the resource model; the data source has no prior
// configuration to preserve, so flattenRecommendRule always adopts the
// API's JSON encoding directly for it.
type RecommendRuleDataSourceModel = RecommendRuleResourceModel
