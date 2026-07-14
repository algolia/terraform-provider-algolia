package abtest

import "github.com/hashicorp/terraform-plugin-framework/types"

// ABTestResourceModel describes the algolia_ab_test resource.
//
// Name, EndAt, Variants, Metrics, and Configuration are write-once: the
// A/B Testing API has no update endpoint (see resource.go), so every
// attribute that shapes test creation is RequiresReplace in the schema, and
// Read never overwrites them from GetABTest's response. That response is
// enriched with runtime results (per-variant conversion/click counts,
// significance, etc.) and its shape diverges from what was submitted to
// AddABTests, so treating it as the source of truth for these fields would
// corrupt state with runtime data and cause a perpetual diff. See
// flattenABTestComputed.
type ABTestResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ABTestID      types.Int64  `tfsdk:"ab_test_id"`
	Name          types.String `tfsdk:"name"`
	EndAt         types.String `tfsdk:"end_at"`
	Variants      types.String `tfsdk:"variants"`
	Metrics       types.String `tfsdk:"metrics"`
	Configuration types.String `tfsdk:"configuration"`
	Status        types.String `tfsdk:"status"`
}

// ABTestDataSourceModel describes the algolia_ab_test data source. Unlike
// the resource, it surfaces GetABTest's enriched response directly
// (including runtime results such as per-variant metrics), since a data
// source has no prior configuration that a naive refresh could corrupt.
type ABTestDataSourceModel struct {
	ABTestID  types.Int64  `tfsdk:"ab_test_id"`
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Status    types.String `tfsdk:"status"`
	EndAt     types.String `tfsdk:"end_at"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	StoppedAt types.String `tfsdk:"stopped_at"`
	Variants  types.String `tfsdk:"variants"`
}
