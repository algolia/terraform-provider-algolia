package abtest

import (
	"encoding/json"
	"strconv"

	abtestingapi "github.com/algolia/algoliasearch-client-go/v4/algolia/abtesting-v3"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// flattenABTestComputed refreshes only the server-owned computed
// attributes (id, ab_test_id, status) from a GetABTest response.
//
// It deliberately does NOT touch Name/EndAt/Variants/Metrics/Configuration:
// GetABTest's response is enriched with runtime results (per-variant
// conversion/click counts, significance, etc.) and its shape diverges from
// what was submitted to AddABTests, so overwriting those fields here would
// corrupt state with runtime data and cause a perpetual diff against the
// user's configuration. Callers (Create/Read/Update in resource.go) must
// leave the model's existing values for those fields as-is - see the
// resource schema's description.
func flattenABTestComputed(abTest *abtestingapi.ABTest, model *ABTestResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	id := strconv.FormatInt(int64(abTest.AbTestID), 10)
	model.ID = types.StringValue(id)
	model.ABTestID = types.Int64Value(int64(abTest.AbTestID))
	model.Status = types.StringValue(string(abTest.Status))

	return diags
}

// flattenABTestImport populates state for `terraform import`. Unlike Read,
// there is no prior configuration to preserve, so it seeds
// name/end_at/variants/configuration from the enriched GetABTest response -
// but this is a best-effort reconstruction, not a faithful replay of what
// was originally submitted to AddABTests:
//
//   - variants is the enriched Variant shape (adds metrics/metadata/
//     estimatedSampleSize not present in AddABTestsVariant); it round-trips
//     through the "index"/"trafficPercentage"/"description"/
//     "customSearchParameters" keys AddABTestsVariant reads, so a
//     subsequent apply that matches those values won't force a replace,
//     but the extra keys mean a byte-for-byte config match is unlikely.
//   - metrics cannot be reconstructed at all: GetABTest does not return the
//     metrics list submitted at creation, only per-metric *results* nested
//     under each variant. Left null - the user must set `metrics`
//     explicitly in configuration after import, which - since it is
//     RequiresReplace - will plan a replace on the next apply.
//   - configuration round-trips cleanly: ABTestConfiguration is the same
//     shape on create and read.
func flattenABTestImport(abTest *abtestingapi.ABTest, model *ABTestResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.Append(flattenABTestComputed(abTest, model)...)
	if diags.HasError() {
		return diags
	}

	model.Name = types.StringValue(abTest.Name)
	model.EndAt = types.StringValue(abTest.EndAt)
	model.Metrics = types.StringNull()

	variantsJSON, err := json.Marshal(abTest.Variants)
	if err != nil {
		diags.AddError("Error encoding variants", "Could not JSON-encode A/B test variants during import: "+err.Error())
		return diags
	}
	model.Variants = types.StringValue(string(variantsJSON))

	if abTest.Configuration != nil {
		configJSON, err := json.Marshal(abTest.Configuration)
		if err != nil {
			diags.AddError("Error encoding configuration", "Could not JSON-encode A/B test configuration during import: "+err.Error())
			return diags
		}
		model.Configuration = types.StringValue(string(configJSON))
	} else {
		model.Configuration = types.StringNull()
	}

	return diags
}

// flattenABTestDataSource maps a GetABTest response onto the
// algolia_ab_test data source model. Unlike the resource, the data source
// has no prior configuration to preserve, so it surfaces the enriched
// response directly, including runtime results.
func flattenABTestDataSource(abTest *abtestingapi.ABTest, model *ABTestDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	id := strconv.FormatInt(int64(abTest.AbTestID), 10)
	model.ID = types.StringValue(id)
	model.ABTestID = types.Int64Value(int64(abTest.AbTestID))
	model.Name = types.StringValue(abTest.Name)
	model.Status = types.StringValue(string(abTest.Status))
	model.EndAt = types.StringValue(abTest.EndAt)
	model.CreatedAt = types.StringValue(abTest.CreatedAt)
	model.UpdatedAt = types.StringValue(abTest.UpdatedAt)

	if stoppedAt := abTest.StoppedAt.Get(); stoppedAt != nil {
		model.StoppedAt = types.StringValue(*stoppedAt)
	} else {
		model.StoppedAt = types.StringNull()
	}

	variantsJSON, err := json.Marshal(abTest.Variants)
	if err != nil {
		diags.AddError("Error encoding variants", "Could not JSON-encode A/B test variants: "+err.Error())
		return diags
	}
	model.Variants = types.StringValue(string(variantsJSON))

	return diags
}
