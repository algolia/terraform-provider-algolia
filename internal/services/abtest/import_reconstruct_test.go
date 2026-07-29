package abtest

import (
	"context"
	"encoding/json"
	"testing"

	abtestingapi "github.com/algolia/algoliasearch-client-go/v4/algolia/abtesting-v3"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func strPtr(s string) *string { return &s }

// enrichedABTest is a GetABTest response of the shape the API actually answers
// with: variants carrying runtime results, and no top-level metrics list.
func enrichedABTest() *abtestingapi.ABTest {
	return &abtestingapi.ABTest{
		AbTestID:  42,
		Name:      "checkout-copy",
		Status:    abtestingapi.Status("active"),
		EndAt:     "2030-01-01T00:00:00Z",
		CreatedAt: "2026-07-01T10:00:00Z",
		UpdatedAt: "2026-07-02T11:00:00Z",
		Variants: []abtestingapi.Variant{
			{
				Index:               "products",
				TrafficPercentage:   60,
				Description:         "control",
				EstimatedSampleSize: func() *int32 { v := int32(1000); return &v }(),
				Metrics: []abtestingapi.MetricResult{
					{Name: "addToCartRate", Value: 0.12, PValue: 0.4},
					{Name: "revenue", Dimension: strPtr("USD"), Value: 91.5, PValue: 0.2},
				},
			},
			{
				Index:             "products_variant",
				TrafficPercentage: 40,
				// No description on this one, so the reconstruction must omit the key
				// rather than emit an empty string the create endpoint would store.
				CustomSearchParameters: map[string]any{"typoTolerance": "strict"},
				Metrics: []abtestingapi.MetricResult{
					// Repeated on purpose: metrics are a property of the test and are
					// reported once per variant, so the rebuild must not duplicate them.
					{Name: "addToCartRate", Value: 0.15, PValue: 0.3},
					{Name: "revenue", Dimension: strPtr("USD"), Value: 104.0, PValue: 0.1},
				},
			},
		},
	}
}

// The audit for this resource recorded that `metrics` could not be recovered on
// import, which left it null on a Required RequiresReplace attribute and made the
// first plan after importing propose destroying a running experiment. The metric
// name and dimension are in fact both present in the per-variant results, so this
// pins that they are recovered.
func TestFlattenABTestImport_RecoversMetricsFromVariantResults(t *testing.T) {
	var model ABTestResourceModel
	if diags := flattenABTestImport(enrichedABTest(), &model); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Metrics.IsNull() {
		t.Fatal("metrics is null: it must be rebuilt from the per-variant results")
	}

	var got []abtestingapi.CreateMetric
	if err := json.Unmarshal([]byte(model.Metrics.ValueString()), &got); err != nil {
		t.Fatalf("metrics is not a valid CreateMetric array: %v (%s)", err, model.Metrics.ValueString())
	}

	if len(got) != 2 {
		t.Fatalf("got %d metrics, want 2 deduplicated across variants: %s", len(got), model.Metrics.ValueString())
	}
	if got[0].Name != "addToCartRate" || got[0].Dimension != nil {
		t.Errorf("metrics[0] = %+v, want addToCartRate with no dimension", got[0])
	}
	if got[1].Name != "revenue" || got[1].Dimension == nil || *got[1].Dimension != "USD" {
		t.Errorf("metrics[1] = %+v, want revenue with dimension USD", got[1])
	}

	// The runtime result fields must not leak into a create-shaped document.
	for _, unwanted := range []string{"value", "pValue", "updatedAt", "significant"} {
		if containsKey(model.Metrics.ValueString(), unwanted) {
			t.Errorf("metrics carries runtime field %q: %s", unwanted, model.Metrics.ValueString())
		}
	}
}

func TestFlattenABTestImport_ReducesVariantsToTheCreateShape(t *testing.T) {
	var model ABTestResourceModel
	if diags := flattenABTestImport(enrichedABTest(), &model); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	var got []map[string]any
	if err := json.Unmarshal([]byte(model.Variants.ValueString()), &got); err != nil {
		t.Fatalf("variants is not a valid array: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d variants, want 2", len(got))
	}

	// Runtime enrichment must be gone: leaving it in is what made the imported
	// value differ from any reasonable configuration.
	for _, unwanted := range []string{"metrics", "metadata", "estimatedSampleSize"} {
		if containsKey(model.Variants.ValueString(), unwanted) {
			t.Errorf("variants carries enrichment field %q: %s", unwanted, model.Variants.ValueString())
		}
	}

	if got[0]["index"] != "products" || got[0]["description"] != "control" {
		t.Errorf("variants[0] = %v, want the control index and description preserved", got[0])
	}
	// An absent description must be omitted, not emitted as "".
	if _, present := got[1]["description"]; present {
		t.Errorf("variants[1] has a description key for a variant that had none: %v", got[1])
	}
	if _, present := got[1]["customSearchParameters"]; !present {
		t.Errorf("variants[1] lost customSearchParameters, which the create endpoint accepts: %v", got[1])
	}
	// ...and must not appear on a variant that never had any.
	if _, present := got[0]["customSearchParameters"]; present {
		t.Errorf("variants[0] gained an empty customSearchParameters: %v", got[0])
	}
}

// A variant created with an explicitly empty customSearchParameters is still a
// search-parameter variant: AddABTestsVariant is a union whose UnmarshalJSON picks
// the arm by whether the key is present at all. Dropping an empty map would
// reconstruct it as a plain variant and stop matching a configuration that still
// declares `customSearchParameters = {}`.
func TestFlattenABTestImport_KeepsAnExplicitlyEmptyCustomSearchParameters(t *testing.T) {
	abTest := enrichedABTest()
	abTest.Variants[0].CustomSearchParameters = map[string]any{}

	var model ABTestResourceModel
	if diags := flattenABTestImport(abTest, &model); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	var got []map[string]any
	if err := json.Unmarshal([]byte(model.Variants.ValueString()), &got); err != nil {
		t.Fatalf("variants is not valid JSON: %v", err)
	}

	params, present := got[0]["customSearchParameters"]
	if !present {
		t.Fatalf("an explicitly empty customSearchParameters was dropped: %s", model.Variants.ValueString())
	}
	if decoded, ok := params.(map[string]any); !ok || len(decoded) != 0 {
		t.Errorf("customSearchParameters = %v, want an empty object", params)
	}
}

func TestFlattenABTestImport_PopulatesTimestamps(t *testing.T) {
	var model ABTestResourceModel
	if diags := flattenABTestImport(enrichedABTest(), &model); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.CreatedAt.ValueString(); got != "2026-07-01T10:00:00Z" {
		t.Errorf("created_at = %q, want the API value", got)
	}
	if got := model.UpdatedAt.ValueString(); got != "2026-07-02T11:00:00Z" {
		t.Errorf("updated_at = %q, want the API value", got)
	}
	// A running test has not stopped, so this is the one timestamp that is null.
	if !model.StoppedAt.IsNull() {
		t.Errorf("stopped_at = %q, want null for a running test", model.StoppedAt.ValueString())
	}
}

func TestCreateShapedMetrics_EmptyWhenNoVariantReportsAny(t *testing.T) {
	abTest := enrichedABTest()
	for i := range abTest.Variants {
		abTest.Variants[i].Metrics = nil
	}

	var model ABTestResourceModel
	if diags := flattenABTestImport(abTest, &model); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	// metrics is Required, so an empty rebuild has to stay null and let the next
	// plan ask for it rather than inventing `[]`, which the API would reject.
	if !model.Metrics.IsNull() {
		t.Errorf("metrics = %q, want null when there is nothing to rebuild from", model.Metrics.ValueString())
	}
}

func TestSuppressEquivalentJSON(t *testing.T) {
	tests := []struct {
		name      string
		state     types.String
		plan      types.String
		wantPlan  types.String
		reasoning string
	}{
		{
			name:      "reformatted document keeps the prior value",
			state:     types.StringValue(`[{"name":"revenue","dimension":"USD"}]`),
			plan:      types.StringValue("[\n  {\n    \"dimension\": \"USD\",\n    \"name\": \"revenue\"\n  }\n]"),
			wantPlan:  types.StringValue(`[{"name":"revenue","dimension":"USD"}]`),
			reasoning: "whitespace and key order must not plan the destruction of a running test",
		},
		{
			name:     "a genuine change is left alone so RequiresReplace still fires",
			state:    types.StringValue(`[{"name":"revenue"}]`),
			plan:     types.StringValue(`[{"name":"addToCartRate"}]`),
			wantPlan: types.StringValue(`[{"name":"addToCartRate"}]`),
		},
		{
			name:     "array order is a real difference",
			state:    types.StringValue(`[{"name":"a"},{"name":"b"}]`),
			plan:     types.StringValue(`[{"name":"b"},{"name":"a"}]`),
			wantPlan: types.StringValue(`[{"name":"b"},{"name":"a"}]`),
		},
		{
			name:     "creation is untouched",
			state:    types.StringNull(),
			plan:     types.StringValue(`[{"name":"revenue"}]`),
			wantPlan: types.StringValue(`[{"name":"revenue"}]`),
		},
		{
			name:     "malformed JSON is never treated as equivalent",
			state:    types.StringValue(`[{"name":"revenue"}]`),
			plan:     types.StringValue(`not json`),
			wantPlan: types.StringValue(`not json`),
		},
		{
			name:     "an unknown plan value belongs to whichever modifier produced it",
			state:    types.StringValue(`[{"name":"revenue"}]`),
			plan:     types.StringUnknown(),
			wantPlan: types.StringUnknown(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := &planmodifier.StringResponse{PlanValue: tc.plan}
			suppressEquivalentJSON().PlanModifyString(context.Background(), planmodifier.StringRequest{
				StateValue: tc.state,
				PlanValue:  tc.plan,
			}, resp)

			if !resp.PlanValue.Equal(tc.wantPlan) {
				t.Errorf("PlanValue = %v, want %v (%s)", resp.PlanValue, tc.wantPlan, tc.reasoning)
			}
		})
	}
}

func containsKey(document, key string) bool {
	var decoded any
	if err := json.Unmarshal([]byte(document), &decoded); err != nil {
		return false
	}
	return hasKey(decoded, key)
}

func hasKey(value any, key string) bool {
	switch typed := value.(type) {
	case map[string]any:
		if _, present := typed[key]; present {
			return true
		}
		for _, nested := range typed {
			if hasKey(nested, key) {
				return true
			}
		}
	case []any:
		for _, nested := range typed {
			if hasKey(nested, key) {
				return true
			}
		}
	}
	return false
}
