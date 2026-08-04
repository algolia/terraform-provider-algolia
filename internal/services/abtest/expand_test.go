package abtest

import (
	"encoding/json"
	"strings"
	"testing"

	abtestingapi "github.com/algolia/algoliasearch-client-go/v4/algolia/abtesting-v3"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExpandAddABTestsRequest(t *testing.T) {
	model := &ABTestResourceModel{
		Name:  types.StringValue("homepage-ranking"),
		EndAt: types.StringValue("2026-08-01T00:00:00Z"),
		Variants: types.StringValue(`[
			{"index": "prod", "trafficPercentage": 50, "description": "control"},
			{"index": "prod_variant", "trafficPercentage": 50}
		]`),
		Metrics:       types.StringValue(`[{"name": "addToCartRate"}, {"name": "revenue", "dimension": "USD"}]`),
		Configuration: types.StringValue(`{"minimumDetectableEffect": {"size": 0.1, "metric": "conversionRate"}, "errorCorrection": "bonferroni"}`),
	}

	request, diags := expandAddABTestsRequest(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if request.Name != "homepage-ranking" {
		t.Fatalf("name = %v, want homepage-ranking", request.Name)
	}
	if request.EndAt != "2026-08-01T00:00:00Z" {
		t.Fatalf("endAt = %v, want 2026-08-01T00:00:00Z", request.EndAt)
	}

	if len(request.Variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(request.Variants))
	}
	first := request.Variants[0].AbTestsVariant
	if first == nil {
		t.Fatalf("expected variant[0] to decode into AbTestsVariant, got %#v", request.Variants[0])
	}
	if first.Index != "prod" || first.TrafficPercentage != 50 {
		t.Fatalf("variant[0] = %#v, want index=prod trafficPercentage=50", first)
	}
	if first.Description == nil || *first.Description != "control" {
		t.Fatalf("variant[0].Description = %#v, want control", first.Description)
	}

	if len(request.Metrics) != 2 {
		t.Fatalf("expected 2 metrics, got %d", len(request.Metrics))
	}
	if request.Metrics[0].Name != "addToCartRate" {
		t.Fatalf("metrics[0].Name = %v, want addToCartRate", request.Metrics[0].Name)
	}
	if request.Metrics[1].Dimension == nil || *request.Metrics[1].Dimension != "USD" {
		t.Fatalf("metrics[1].Dimension = %#v, want USD", request.Metrics[1].Dimension)
	}

	if request.Configuration == nil {
		t.Fatal("expected configuration to be set")
	}
	if request.Configuration.MinimumDetectableEffect == nil || request.Configuration.MinimumDetectableEffect.Size != 0.1 {
		t.Fatalf("configuration.MinimumDetectableEffect = %#v, want size=0.1", request.Configuration.MinimumDetectableEffect)
	}
	if request.Configuration.ErrorCorrection == nil || *request.Configuration.ErrorCorrection != abtestingapi.ERROR_CORRECTION_TYPE_BONFERRONI {
		t.Fatalf("configuration.ErrorCorrection = %#v, want bonferroni", request.Configuration.ErrorCorrection)
	}
}

func TestExpandAddABTestsRequest_NoConfiguration(t *testing.T) {
	model := &ABTestResourceModel{
		Name:          types.StringValue("no-config-test"),
		EndAt:         types.StringValue("2026-08-01T00:00:00Z"),
		Variants:      types.StringValue(`[{"index": "prod", "trafficPercentage": 100}]`),
		Metrics:       types.StringValue(`[{"name": "clickThroughRate"}]`),
		Configuration: types.StringNull(),
	}

	request, diags := expandAddABTestsRequest(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if request.Configuration != nil {
		t.Fatalf("expected configuration to be nil, got %#v", request.Configuration)
	}
}

func TestExpandVariants(t *testing.T) {
	t.Run("customSearchParameters decodes into AbTestsVariantSearchParams", func(t *testing.T) {
		variants, diags := expandVariants(types.StringValue(`[
			{"index": "prod", "trafficPercentage": 50, "customSearchParameters": {"typoTolerance": false}}
		]`))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if len(variants) != 1 {
			t.Fatalf("expected 1 variant, got %d", len(variants))
		}
		searchParams := variants[0].AbTestsVariantSearchParams
		if searchParams == nil {
			t.Fatalf("expected variant to decode into AbTestsVariantSearchParams, got %#v", variants[0])
		}
		if searchParams.Index != "prod" || searchParams.TrafficPercentage != 50 {
			t.Fatalf("searchParams = %#v, want index=prod trafficPercentage=50", searchParams)
		}
	})

	// Asserting the arm is populated is not enough, and that is how this went
	// unnoticed: the generated UnmarshalJSON populates the SearchParams arm and then
	// the plain arm as well, and the generated MarshalJSON serialises the plain one
	// first, so customSearchParameters never reached Algolia. Where the variants also
	// differed by index the API accepted the request and ran a test that exercised
	// none of the requested parameters. Pin the serialised form, which is the thing
	// that was actually wrong.
	t.Run("customSearchParameters survives serialisation", func(t *testing.T) {
		variants, diags := expandVariants(types.StringValue(`[
			{"index": "prod", "trafficPercentage": 50},
			{"index": "prod", "trafficPercentage": 50, "customSearchParameters": {"typoTolerance": false}}
		]`))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}

		if variants[1].AbTestsVariant != nil {
			t.Errorf("a variant carrying customSearchParameters must not also populate the plain arm, "+
				"or MarshalJSON serialises that one and drops the parameters: %#v", variants[1].AbTestsVariant)
		}

		encoded, err := json.Marshal(variants)
		if err != nil {
			t.Fatalf("marshalling variants: %v", err)
		}
		if !strings.Contains(string(encoded), "customSearchParameters") {
			t.Fatalf("customSearchParameters was dropped on the way to Algolia: %s", encoded)
		}
		if !strings.Contains(string(encoded), `"typoTolerance":false`) {
			t.Fatalf("the parameter value was not serialised: %s", encoded)
		}
	})

	// Selecting the arm by hand made this regress once: `null` decodes into a nil map
	// without error and then into the variant struct as a no-op, so a null entry became
	// a silent {"index":"","trafficPercentage":0} where it used to be rejected.
	t.Run("a null variant is rejected rather than becoming an empty one", func(t *testing.T) {
		variants, diags := expandVariants(types.StringValue(`[null]`))
		if !diags.HasError() {
			encoded, _ := json.Marshal(variants)
			t.Fatalf("expected a diagnostic for a null variant, got %s", encoded)
		}
	})

	t.Run("invalid JSON returns a diagnostic error", func(t *testing.T) {
		_, diags := expandVariants(types.StringValue(`not valid json`))
		if !diags.HasError() {
			t.Fatal("expected a diagnostic error for invalid variants JSON")
		}
	})

	t.Run("JSON object instead of array returns a diagnostic error", func(t *testing.T) {
		_, diags := expandVariants(types.StringValue(`{"index": "prod"}`))
		if !diags.HasError() {
			t.Fatal("expected a diagnostic error for a non-array variants value")
		}
	})
}

func TestExpandMetrics(t *testing.T) {
	t.Run("invalid JSON returns a diagnostic error", func(t *testing.T) {
		_, diags := expandMetrics(types.StringValue(`not valid json`))
		if !diags.HasError() {
			t.Fatal("expected a diagnostic error for invalid metrics JSON")
		}
	})
}

func TestExpandConfiguration(t *testing.T) {
	t.Run("invalid JSON returns a diagnostic error", func(t *testing.T) {
		_, diags := expandConfiguration(types.StringValue(`not valid json`))
		if !diags.HasError() {
			t.Fatal("expected a diagnostic error for invalid configuration JSON")
		}
	})

	t.Run("invalid errorCorrection enum value returns a diagnostic error", func(t *testing.T) {
		_, diags := expandConfiguration(types.StringValue(`{"errorCorrection": "not-a-real-method"}`))
		if !diags.HasError() {
			t.Fatal("expected a diagnostic error for an invalid errorCorrection value")
		}
	})
}
