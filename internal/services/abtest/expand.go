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
// oneOf(AbTestsVariant, AbTestsVariantSearchParams), and the union arm has to be
// selected here rather than left to the generated UnmarshalJSON, which cannot do
// it: given a variant carrying `customSearchParameters` it populates the
// SearchParams arm and *then* unconditionally populates the plain AbTestsVariant
// arm too, which also succeeds because the extra key is simply ignored. The
// generated MarshalJSON serialises whichever arm it finds first, and that is the
// plain one, so `customSearchParameters` is dropped on the way out.
//
// The consequence is worse than an error. When the variants also differ by index
// Algolia accepts the request and creates a test that silently exercises none of
// the parameters that were asked for; only variants sharing an index fail loudly,
// with "An A/B test variant must have a unique index or custom search
// parameters". So decode into a plain map first and construct the arm explicitly.
func expandVariants(value types.String) ([]abtestingapi.AddABTestsVariant, diag.Diagnostics) {
	var diags diag.Diagnostics

	invalid := func(detail string) ([]abtestingapi.AddABTestsVariant, diag.Diagnostics) {
		diags.AddError(
			"Invalid variants JSON",
			"The `variants` attribute must be a JSON-encoded array of A/B test variants (e.g. "+
				"jsonencode([{ index = \"prod\", trafficPercentage = 50 }, ...])). "+detail,
		)

		return nil, diags
	}

	var raw []json.RawMessage
	if err := json.Unmarshal([]byte(value.ValueString()), &raw); err != nil {
		return invalid("Failed to parse: " + err.Error())
	}

	variants := make([]abtestingapi.AddABTestsVariant, 0, len(raw))
	for _, entry := range raw {
		var keys map[string]json.RawMessage
		if err := json.Unmarshal(entry, &keys); err != nil {
			return invalid("Failed to parse a variant: " + err.Error())
		}

		if _, ok := keys["customSearchParameters"]; ok {
			var withParams abtestingapi.AbTestsVariantSearchParams
			if err := json.Unmarshal(entry, &withParams); err != nil {
				return invalid("Failed to parse a variant with customSearchParameters: " + err.Error())
			}
			variants = append(variants, *abtestingapi.AbTestsVariantSearchParamsAsAddABTestsVariant(&withParams))

			continue
		}

		var plain abtestingapi.AbTestsVariant
		if err := json.Unmarshal(entry, &plain); err != nil {
			return invalid("Failed to parse a variant: " + err.Error())
		}
		variants = append(variants, *abtestingapi.AbTestsVariantAsAddABTestsVariant(&plain))
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
