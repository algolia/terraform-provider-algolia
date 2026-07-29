package abtest

import (
	"encoding/json"
	"strconv"

	abtestingapi "github.com/algolia/algoliasearch-client-go/v4/algolia/abtesting-v3"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// flattenABTestComputed refreshes only the server-owned computed
// attributes (id, ab_test_id, status) from a GetABTest response. It is what
// Create and Update use: for a Required attribute the value Terraform applied
// must equal the value it planned, so those two must not adopt anything the
// API echoes back. Read uses flattenABTestRead instead, which also refreshes
// the two attributes GetABTest returns in the shape they were submitted.
//
// It deliberately does NOT touch Variants or Metrics: GetABTest's response is
// enriched with runtime results (per-variant conversion/click counts,
// significance, etc.) and its shape diverges from what was submitted to
// AddABTests, so overwriting those fields would corrupt state with runtime data
// and cause a perpetual diff against the user's configuration - see the resource
// schema's description.
//
// Configuration is the exception, and only when the model has none: Algolia
// substitutes its own when a test is created without one, so an absent value is
// filled from the response while a configured one is left untouched.
func flattenABTestComputed(abTest *abtestingapi.ABTest, model *ABTestResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	id := strconv.FormatInt(int64(abTest.AbTestID), 10)
	model.ID = types.StringValue(id)
	model.ABTestID = types.Int64Value(int64(abTest.AbTestID))
	model.Status = types.StringValue(string(abTest.Status))
	model.CreatedAt = types.StringValue(abTest.CreatedAt)
	model.UpdatedAt = types.StringValue(abTest.UpdatedAt)

	// stoppedAt is nullable and stays null while the test runs, so it is the one
	// timestamp that legitimately has no value most of the time.
	if stoppedAt := abTest.StoppedAt.Get(); stoppedAt != nil {
		model.StoppedAt = types.StringValue(*stoppedAt)
	} else {
		model.StoppedAt = types.StringNull()
	}

	// configuration is Optional+Computed because Algolia substitutes its own when
	// none is given. A configured document is kept verbatim, since the attribute is
	// still the user's to own and re-encoding it would fail the apply; only an
	// absent one is filled from the response. Doing that here rather than on the
	// import path alone is what makes an imported test comparable with one created
	// from a configuration that omitted the attribute: both then hold what the
	// server chose.
	if model.Configuration.IsNull() || model.Configuration.IsUnknown() {
		configuration, configDiags := encodeConfiguration(abTest.Configuration)
		diags.Append(configDiags...)
		if diags.HasError() {
			return diags
		}
		model.Configuration = configuration
	}

	return diags
}

// encodeConfiguration renders an A/B test configuration into the JSON-encoded
// string the schema holds, or null when the API reported none.
func encodeConfiguration(configuration *abtestingapi.ABTestConfiguration) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if configuration == nil {
		return types.StringNull(), diags
	}

	encoded, err := json.Marshal(configuration)
	if err != nil {
		diags.AddError("Error encoding configuration", "Could not JSON-encode A/B test configuration: "+err.Error())
		return types.StringNull(), diags
	}

	return types.StringValue(string(encoded)), diags
}

// flattenABTestRead is the Read-path refresh. On top of the computed
// attributes it refreshes `name` and `end_at`, so that a test renamed or
// rescheduled outside Terraform shows up as drift instead of staying
// invisible forever. Both are plain strings that GetABTest returns exactly as
// they were submitted to AddABTests (flattenABTestImport relies on the same
// property, and the import step of TestAccABTestResource_basic verifies it
// byte-for-byte), so adopting them cannot corrupt state.
//
// `variants` and `metrics` stay excluded: unlike name/end_at, GetABTest's shape
// for those diverges from the create shape (see flattenABTestComputed), so
// refreshing them would corrupt state rather than reveal drift. `configuration`
// is only ever filled when state holds none.
func flattenABTestRead(abTest *abtestingapi.ABTest, model *ABTestResourceModel) diag.Diagnostics {
	diags := flattenABTestComputed(abTest, model)
	if diags.HasError() {
		return diags
	}

	model.Name = types.StringValue(abTest.Name)
	model.EndAt = types.StringValue(abTest.EndAt)

	return diags
}

// flattenABTestImport populates state for `terraform import`. There is no prior
// configuration to preserve, so on top of what Read refreshes (name/end_at) it
// reconstructs variants, metrics and configuration in the shape the create
// endpoint accepts, rather than the enriched shape GetABTest answers with.
//
// Reconstructing rather than echoing matters because all three attributes are
// RequiresReplace. State holding the enriched response would differ from any
// reasonable configuration, and the first plan after importing would propose
// destroying a running experiment and discarding its statistics. Emitting the
// create shape means a configuration describing the same test matches, and
// `suppressEquivalentJSON` absorbs any remaining difference in formatting or key
// order.
//
//   - variants keeps only the keys AddABTestsVariant accepts (index,
//     trafficPercentage, description, customSearchParameters) and drops the
//     runtime enrichment (metrics, metadata, estimatedSampleSize).
//   - metrics is rebuilt from the per-variant results where there are any.
//     GetABTest does not return the submitted metrics list, but a variant's
//     results carry each metric's `name` and `dimension`, which is exactly what
//     CreateMetric holds. The catch, verified against the API: results only exist
//     once the test has gathered data. A test created moments ago reports
//     `"metrics": null` on every variant, so nothing can be rebuilt and the
//     attribute stays null. That is the uninteresting case in practice, since a
//     test worth importing has been running, but it does mean import cannot
//     promise metrics.
//   - configuration round-trips as-is: ABTestConfiguration is the same shape on
//     create and read.
func flattenABTestImport(abTest *abtestingapi.ABTest, model *ABTestResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.Append(flattenABTestRead(abTest, model)...)
	if diags.HasError() {
		return diags
	}

	variantsJSON, err := json.Marshal(createShapedVariants(abTest.Variants))
	if err != nil {
		diags.AddError("Error encoding variants", "Could not JSON-encode A/B test variants during import: "+err.Error())
		return diags
	}
	model.Variants = types.StringValue(string(variantsJSON))

	metrics := createShapedMetrics(abTest.Variants)
	if len(metrics) == 0 {
		// Nothing to rebuild from. metrics is Required, so leaving it null makes the
		// next plan ask for it rather than silently inventing an empty list.
		model.Metrics = types.StringNull()
	} else {
		metricsJSON, err := json.Marshal(metrics)
		if err != nil {
			diags.AddError("Error encoding metrics", "Could not JSON-encode A/B test metrics during import: "+err.Error())
			return diags
		}
		model.Metrics = types.StringValue(string(metricsJSON))
	}

	// configuration needs nothing here: the model starts empty on import, so
	// flattenABTestComputed has already filled it from the response.

	return diags
}

// createShapedVariants reduces the enriched variants GetABTest returns to the
// keys the create endpoint accepts. The JSON tags come from the client's own
// AbTestsVariantSearchParams, so the reconstruction stays correct if the create
// shape gains a field, and `customSearchParameters` is omitted entirely for a
// plain index-to-index test rather than emitted as null.
func createShapedVariants(variants []abtestingapi.Variant) []map[string]any {
	shaped := make([]map[string]any, 0, len(variants))

	for _, variant := range variants {
		entry := map[string]any{
			"index":             variant.Index,
			"trafficPercentage": variant.TrafficPercentage,
		}
		if variant.Description != "" {
			entry["description"] = variant.Description
		}
		if len(variant.CustomSearchParameters) > 0 {
			entry["customSearchParameters"] = variant.CustomSearchParameters
		}

		shaped = append(shaped, entry)
	}

	return shaped
}

// createShapedMetrics rebuilds the submitted metrics list from the per-variant
// results.
//
// The metrics are a property of the test, reported once per variant, so every
// variant carries the same set. Results are deduplicated by name and dimension so
// that walking all the variants cannot produce a list with repeats that the create
// endpoint would reject.
//
// Returns an empty slice when no variant reports any results, which is what a test
// that has not gathered data yet looks like: every variant has `"metrics": null`
// until traffic arrives.
func createShapedMetrics(variants []abtestingapi.Variant) []abtestingapi.CreateMetric {
	metrics := make([]abtestingapi.CreateMetric, 0)
	seen := make(map[string]bool)

	for _, variant := range variants {
		for _, result := range variant.Metrics {
			key := result.Name
			if result.Dimension != nil {
				key += "\x00" + *result.Dimension
			}
			if seen[key] {
				continue
			}
			seen[key] = true

			metrics = append(metrics, abtestingapi.CreateMetric{
				Name:      result.Name,
				Dimension: result.Dimension,
			})
		}
	}

	return metrics
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
