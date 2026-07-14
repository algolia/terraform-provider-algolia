package abtest

import (
	"encoding/json"

	abtestingapi "github.com/algolia/algoliasearch-client-go/v4/algolia/abtesting-v3"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// expandAddABTestsRequest converts the Terraform plan into an
// AddABTestsRequest for AddABTests. name/end_at/variants/metrics/
// configuration are all RequiresReplace in the schema, since the A/B
// Testing API has no update endpoint - see resource.go.
func expandAddABTestsRequest(model *ABTestResourceModel) (*abtestingapi.AddABTestsRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	variants, variantDiags := expandVariants(model.Variants)
	diags.Append(variantDiags...)

	metrics, metricDiags := expandMetrics(model.Metrics)
	diags.Append(metricDiags...)

	if diags.HasError() {
		return nil, diags
	}

	request := abtestingapi.NewAddABTestsRequest(
		model.Name.ValueString(),
		variants,
		metrics,
		model.EndAt.ValueString(),
	)

	if !model.Configuration.IsNull() && !model.Configuration.IsUnknown() && model.Configuration.ValueString() != "" {
		configuration, configDiags := expandConfiguration(model.Configuration)
		diags.Append(configDiags...)
		if diags.HasError() {
			return nil, diags
		}
		request.Configuration = configuration
	}

	return request, diags
}

// expandVariants JSON-decodes the `variants` attribute into the slice of
// AddABTestsVariant expected by AddABTestsRequest. AddABTestsVariant is a
// oneOf(AbTestsVariant, AbTestsVariantSearchParams); its generated
// UnmarshalJSON picks the right variant based on whether
// `customSearchParameters` is present, so a plain json.Unmarshal into the
// slice is enough.
func expandVariants(value types.String) ([]abtestingapi.AddABTestsVariant, diag.Diagnostics) {
	var diags diag.Diagnostics
	var variants []abtestingapi.AddABTestsVariant

	if err := json.Unmarshal([]byte(value.ValueString()), &variants); err != nil {
		diags.AddError(
			"Invalid variants JSON",
			"The `variants` attribute must be a JSON-encoded array of A/B test variants (e.g. "+
				"jsonencode([{ index = \"prod\", trafficPercentage = 50 }, ...])). Failed to parse: "+err.Error(),
		)
	}

	return variants, diags
}

// expandMetrics JSON-decodes the `metrics` attribute into the slice of
// CreateMetric expected by AddABTestsRequest.
func expandMetrics(value types.String) ([]abtestingapi.CreateMetric, diag.Diagnostics) {
	var diags diag.Diagnostics
	var metrics []abtestingapi.CreateMetric

	if err := json.Unmarshal([]byte(value.ValueString()), &metrics); err != nil {
		diags.AddError(
			"Invalid metrics JSON",
			"The `metrics` attribute must be a JSON-encoded array of A/B test metrics (e.g. "+
				"jsonencode([{ name = \"addToCartRate\" }, ...])). Failed to parse: "+err.Error(),
		)
	}

	return metrics, diags
}

// expandConfiguration JSON-decodes the `configuration` attribute into the
// ABTestConfiguration expected by AddABTestsRequest.
func expandConfiguration(value types.String) (*abtestingapi.ABTestConfiguration, diag.Diagnostics) {
	var diags diag.Diagnostics
	var configuration abtestingapi.ABTestConfiguration

	if err := json.Unmarshal([]byte(value.ValueString()), &configuration); err != nil {
		diags.AddError(
			"Invalid configuration JSON",
			"The `configuration` attribute must be JSON-encoded (e.g. jsonencode({ minimumDetectableEffect = "+
				"{ size = 0.1, metric = \"conversionRate\" } })). Failed to parse: "+err.Error(),
		)
		return nil, diags
	}

	return &configuration, diags
}
