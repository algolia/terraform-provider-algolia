package abtest

import (
	"encoding/json"
	"testing"

	abtestingapi "github.com/algolia/algoliasearch-client-go/v4/algolia/abtesting-v3"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func newTestABTest() *abtestingapi.ABTest {
	description := "control"
	abTest := abtestingapi.NewABTest(
		42,
		"2026-07-01T00:00:00Z",
		"2026-06-01T00:00:00Z",
		"2026-08-01T00:00:00Z",
		"homepage-ranking",
		abtestingapi.STATUS_ACTIVE,
		[]abtestingapi.Variant{
			{Index: "prod", TrafficPercentage: 50, Description: description, Metrics: []abtestingapi.MetricResult{
				{Name: "addToCartRate", Value: 0.12, PValue: 0.03},
			}},
			{Index: "prod_variant", TrafficPercentage: 50, Metrics: []abtestingapi.MetricResult{
				{Name: "addToCartRate", Value: 0.14, PValue: 0.03},
			}},
		},
	)
	abTest.Configuration = abtestingapi.NewABTestConfiguration(
		abtestingapi.WithABTestConfigurationErrorCorrection(abtestingapi.ERROR_CORRECTION_TYPE_BONFERRONI),
	)

	return abTest
}

// TestFlattenABTestComputed_PreservesConfig is the "Read preserves config"
// regression test: flattenABTestComputed must refresh id/ab_test_id/status
// from the enriched GetABTest response, but must never touch
// name/end_at/variants/metrics/configuration - those stay exactly as they
// were in state before Read ran, even though the API response contains
// values for name/end_at/variants/configuration that differ in shape (and,
// in this test, in value) from what's already in state.
func TestFlattenABTestComputed_PreservesConfig(t *testing.T) {
	abTest := newTestABTest()

	model := &ABTestResourceModel{
		ID:            types.StringValue("42"),
		ABTestID:      types.Int64Value(42),
		Name:          types.StringValue("configured-name"),
		EndAt:         types.StringValue("2099-01-01T00:00:00Z"),
		Variants:      types.StringValue(`[{"index":"prod","trafficPercentage":50}]`),
		Metrics:       types.StringValue(`[{"name":"addToCartRate"}]`),
		Configuration: types.StringValue(`{"errorCorrection":"benjamini-hochberg"}`),
		Status:        types.StringValue("stale-status"),
	}

	diags := flattenABTestComputed(abTest, model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.ID.ValueString() != "42" {
		t.Fatalf("id = %v, want 42", model.ID.ValueString())
	}
	if model.ABTestID.ValueInt64() != 42 {
		t.Fatalf("ab_test_id = %v, want 42", model.ABTestID.ValueInt64())
	}
	if model.Status.ValueString() != "active" {
		t.Fatalf("status = %v, want active", model.Status.ValueString())
	}

	// Everything else must be untouched, even though it disagrees with the
	// API response (abTest.Name == "homepage-ranking", abTest.EndAt ==
	// "2026-08-01T00:00:00Z", etc.) - that's the whole point of this test.
	if model.Name.ValueString() != "configured-name" {
		t.Fatalf("name = %v, want configured-name (preserved)", model.Name.ValueString())
	}
	if model.EndAt.ValueString() != "2099-01-01T00:00:00Z" {
		t.Fatalf("end_at = %v, want 2099-01-01T00:00:00Z (preserved)", model.EndAt.ValueString())
	}
	if model.Variants.ValueString() != `[{"index":"prod","trafficPercentage":50}]` {
		t.Fatalf("variants = %v, want the original configured JSON (preserved)", model.Variants.ValueString())
	}
	if model.Metrics.ValueString() != `[{"name":"addToCartRate"}]` {
		t.Fatalf("metrics = %v, want the original configured JSON (preserved)", model.Metrics.ValueString())
	}
	if model.Configuration.ValueString() != `{"errorCorrection":"benjamini-hochberg"}` {
		t.Fatalf("configuration = %v, want the original configured JSON (preserved)", model.Configuration.ValueString())
	}
}

func TestFlattenABTestImport(t *testing.T) {
	abTest := newTestABTest()

	var model ABTestResourceModel
	diags := flattenABTestImport(abTest, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.ID.ValueString() != "42" {
		t.Fatalf("id = %v, want 42", model.ID.ValueString())
	}
	if model.Name.ValueString() != "homepage-ranking" {
		t.Fatalf("name = %v, want homepage-ranking", model.Name.ValueString())
	}
	if model.EndAt.ValueString() != "2026-08-01T00:00:00Z" {
		t.Fatalf("end_at = %v, want 2026-08-01T00:00:00Z", model.EndAt.ValueString())
	}

	// metrics cannot be reconstructed from GetABTest - always null on
	// import.
	if !model.Metrics.IsNull() {
		t.Fatalf("metrics = %v, want null (unrecoverable on import)", model.Metrics.ValueString())
	}

	// variants round-trips through JSON; assert on decoded fields rather
	// than the exact string, since the enriched shape includes extra keys
	// (metrics) beyond what AddABTestsVariant reads.
	var variants []map[string]any
	if err := json.Unmarshal([]byte(model.Variants.ValueString()), &variants); err != nil {
		t.Fatalf("variants is not valid JSON: %v", err)
	}
	if len(variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(variants))
	}
	if variants[0]["index"] != "prod" {
		t.Fatalf("variants[0].index = %v, want prod", variants[0]["index"])
	}
	if variants[0]["trafficPercentage"].(float64) != 50 {
		t.Fatalf("variants[0].trafficPercentage = %v, want 50", variants[0]["trafficPercentage"])
	}

	// configuration round-trips cleanly.
	var configuration map[string]any
	if err := json.Unmarshal([]byte(model.Configuration.ValueString()), &configuration); err != nil {
		t.Fatalf("configuration is not valid JSON: %v", err)
	}
	if configuration["errorCorrection"] != "bonferroni" {
		t.Fatalf("configuration.errorCorrection = %v, want bonferroni", configuration["errorCorrection"])
	}
}

func TestFlattenABTestImport_NoConfiguration(t *testing.T) {
	abTest := newTestABTest()
	abTest.Configuration = nil

	var model ABTestResourceModel
	diags := flattenABTestImport(abTest, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.Configuration.IsNull() {
		t.Fatalf("configuration = %v, want null", model.Configuration.ValueString())
	}
}

func TestFlattenABTestDataSource(t *testing.T) {
	abTest := newTestABTest()
	stoppedAt := "2026-07-15T00:00:00Z"
	abTest.SetStoppedAt(stoppedAt)

	var model ABTestDataSourceModel
	diags := flattenABTestDataSource(abTest, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.ID.ValueString() != "42" {
		t.Fatalf("id = %v, want 42", model.ID.ValueString())
	}
	if model.ABTestID.ValueInt64() != 42 {
		t.Fatalf("ab_test_id = %v, want 42", model.ABTestID.ValueInt64())
	}
	if model.Name.ValueString() != "homepage-ranking" {
		t.Fatalf("name = %v, want homepage-ranking", model.Name.ValueString())
	}
	if model.Status.ValueString() != "active" {
		t.Fatalf("status = %v, want active", model.Status.ValueString())
	}
	if model.StoppedAt.ValueString() != stoppedAt {
		t.Fatalf("stopped_at = %v, want %v", model.StoppedAt.ValueString(), stoppedAt)
	}

	// The enriched per-variant metrics results must round-trip into the
	// JSON blob - this is the data source's whole reason to exist.
	var variants []map[string]any
	if err := json.Unmarshal([]byte(model.Variants.ValueString()), &variants); err != nil {
		t.Fatalf("variants is not valid JSON: %v", err)
	}
	if len(variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(variants))
	}
	metrics, ok := variants[0]["metrics"].([]any)
	if !ok || len(metrics) != 1 {
		t.Fatalf("variants[0].metrics = %#v, want a 1-element array", variants[0]["metrics"])
	}
	metric := metrics[0].(map[string]any)
	if metric["name"] != "addToCartRate" {
		t.Fatalf("variants[0].metrics[0].name = %v, want addToCartRate", metric["name"])
	}
}

func TestFlattenABTestDataSource_NoStoppedAt(t *testing.T) {
	abTest := newTestABTest()

	var model ABTestDataSourceModel
	diags := flattenABTestDataSource(abTest, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.StoppedAt.IsNull() {
		t.Fatalf("stopped_at = %v, want null", model.StoppedAt.ValueString())
	}
}
