package ingestion

import (
	"testing"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenSource_PopulatesFields(t *testing.T) {
	authID := "auth-1"
	source := &ingestionapi.Source{
		SourceID:         "source-1",
		Type:             ingestionapi.SOURCE_TYPE_CSV,
		Name:             "my-source",
		AuthenticationID: &authID,
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-02T00:00:00Z",
	}

	var model SourceResourceModel
	diags := flattenSource(source, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.ID.ValueString() != "source-1" {
		t.Fatalf("id = %v, want source-1", model.ID.ValueString())
	}
	if model.SourceID.ValueString() != "source-1" {
		t.Fatalf("source_id = %v, want source-1", model.SourceID.ValueString())
	}
	if model.Type.ValueString() != "csv" {
		t.Fatalf("type = %v, want csv", model.Type.ValueString())
	}
	if model.Name.ValueString() != "my-source" {
		t.Fatalf("name = %v, want my-source", model.Name.ValueString())
	}
	if model.AuthenticationID.ValueString() != "auth-1" {
		t.Fatalf("authentication_id = %v, want auth-1", model.AuthenticationID.ValueString())
	}
	if model.CreatedAt.ValueString() != "2024-01-01T00:00:00Z" {
		t.Fatalf("created_at = %v, want 2024-01-01T00:00:00Z", model.CreatedAt.ValueString())
	}
	if model.UpdatedAt.ValueString() != "2024-01-02T00:00:00Z" {
		t.Fatalf("updated_at = %v, want 2024-01-02T00:00:00Z", model.UpdatedAt.ValueString())
	}
}

func TestFlattenSource_NoAuthenticationIDIsNull(t *testing.T) {
	source := &ingestionapi.Source{
		SourceID:  "source-2",
		Type:      ingestionapi.SOURCE_TYPE_PUSH,
		Name:      "my-push-source",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-01T00:00:00Z",
	}

	var model SourceResourceModel
	diags := flattenSource(source, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.AuthenticationID.IsNull() {
		t.Fatalf("authentication_id = %#v, want null", model.AuthenticationID)
	}
}

// TestFlattenSource_InputPreservedWhenSemanticallyEqual is the core
// regression test for the semantic-equality refresh behavior: unlike
// authentication's write-only input, GetSource returns a source's input in
// full, so naively overwriting the configured JSON string with the API's
// encoding on every Read would create a perpetual diff whenever the API
// echoes back the same configuration with different key/array ordering.
func TestFlattenSource_InputPreservedWhenSemanticallyEqual(t *testing.T) {
	const configuredInput = `{"url": "https://example.com/data.csv", "uniqueIDColumn": "id"}`

	model := SourceResourceModel{
		Input: types.StringValue(configuredInput),
	}

	source := &ingestionapi.Source{
		SourceID:  "source-3",
		Type:      ingestionapi.SOURCE_TYPE_CSV,
		Name:      "my-source",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-01T00:00:00Z",
		// Same data as configuredInput, but with keys in a different order -
		// simulates the API echoing back a semantically identical encoding.
		Input: ingestionapi.SourceCSVAsSourceInput(ingestionapi.NewSourceCSV(
			"https://example.com/data.csv",
			ingestionapi.WithSourceCSVUniqueIDColumn("id"),
		)),
	}

	diags := flattenSource(source, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Input.ValueString() != configuredInput {
		t.Fatalf("input = %v, want unchanged configured value %v", model.Input.ValueString(), configuredInput)
	}
}

// TestFlattenSource_InputAdoptsAPIValueWhenDifferent covers the opposite
// case: when the API's input is not semantically equal to what's
// configured (e.g. drift introduced out-of-band), flattenSource must adopt
// the API's value so Terraform actually reports the difference.
func TestFlattenSource_InputAdoptsAPIValueWhenDifferent(t *testing.T) {
	model := SourceResourceModel{
		Input: types.StringValue(`{"url": "https://example.com/old.csv"}`),
	}

	source := &ingestionapi.Source{
		SourceID:  "source-4",
		Type:      ingestionapi.SOURCE_TYPE_CSV,
		Name:      "my-source",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-01T00:00:00Z",
		Input:     ingestionapi.SourceCSVAsSourceInput(ingestionapi.NewSourceCSV("https://example.com/new.csv")),
	}

	diags := flattenSource(source, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Input.ValueString() != `{"url":"https://example.com/new.csv"}` {
		t.Fatalf("input = %v, want the API's new value", model.Input.ValueString())
	}
}

// TestFlattenSource_NilAPIInputKeepsConfiguredValue covers a source type
// whose API response has no input at all (e.g. "push"): flattenSource must
// not clobber a configured value with null just because the API omitted
// input from its response.
func TestFlattenSource_NilAPIInputKeepsConfiguredValue(t *testing.T) {
	model := SourceResourceModel{
		Input: types.StringValue(`{"some": "config"}`),
	}

	source := &ingestionapi.Source{
		SourceID:  "source-5",
		Type:      ingestionapi.SOURCE_TYPE_PUSH,
		Name:      "my-source",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-01T00:00:00Z",
		Input:     nil,
	}

	diags := flattenSource(source, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Input.ValueString() != `{"some": "config"}` {
		t.Fatalf("input = %v, want the previously configured value preserved", model.Input.ValueString())
	}
}

func TestFlattenSource_NilAPIInputAndUnconfiguredIsNull(t *testing.T) {
	model := SourceResourceModel{
		Input: types.StringNull(),
	}

	source := &ingestionapi.Source{
		SourceID:  "source-6",
		Type:      ingestionapi.SOURCE_TYPE_PUSH,
		Name:      "my-push-source",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-01T00:00:00Z",
	}

	diags := flattenSource(source, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.Input.IsNull() {
		t.Fatalf("input = %#v, want null", model.Input)
	}
}

func TestFlattenSourceDataSource_PopulatesInput(t *testing.T) {
	source := &ingestionapi.Source{
		SourceID:  "source-7",
		Type:      ingestionapi.SOURCE_TYPE_CSV,
		Name:      "my-source",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-01T00:00:00Z",
		Input:     ingestionapi.SourceCSVAsSourceInput(ingestionapi.NewSourceCSV("https://example.com/data.csv")),
	}

	var model SourceDataSourceModel
	diags := flattenSourceDataSource(source, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Input.ValueString() != `{"url":"https://example.com/data.csv"}` {
		t.Fatalf("input = %v, want the encoded API value", model.Input.ValueString())
	}
}

func TestFlattenSourceDataSource_NilInputIsNull(t *testing.T) {
	source := &ingestionapi.Source{
		SourceID:  "source-8",
		Type:      ingestionapi.SOURCE_TYPE_PUSH,
		Name:      "my-push-source",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-01T00:00:00Z",
	}

	var model SourceDataSourceModel
	diags := flattenSourceDataSource(source, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.Input.IsNull() {
		t.Fatalf("input = %#v, want null", model.Input)
	}
}

// TestExpandFlattenSource_RoundTrip exercises the JSON round trip
// end-to-end: JSON string -> SourceInput (Create) -> ... -> Source
// response -> flatten, mirroring how the real resource lifecycle uses
// these two halves of the JSON-encoded-field pattern together.
func TestExpandFlattenSource_RoundTrip(t *testing.T) {
	const configuredInput = `{"url": "https://example.com/data.csv"}`

	model := &SourceResourceModel{
		Type:  types.StringValue(string(ingestionapi.SOURCE_TYPE_CSV)),
		Name:  types.StringValue("round-trip-source"),
		Input: types.StringValue(configuredInput),
	}

	create, diags := expandSourceCreate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics expanding: %v", diags)
	}

	// The API would echo back the sourceID/createdAt/updatedAt and the
	// input in full; simulate that response using the same union value
	// produced by expandSourceCreate.
	source := &ingestionapi.Source{
		SourceID:  "source-9",
		Type:      create.Type,
		Name:      create.Name,
		Input:     create.Input,
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-01T00:00:00Z",
	}

	diags = flattenSource(source, model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics flattening: %v", diags)
	}

	if model.SourceID.ValueString() != "source-9" {
		t.Fatalf("source_id = %v, want source-9", model.SourceID.ValueString())
	}
	// Input survives the round trip unchanged, exactly as configured, since
	// the API's encoding is semantically equal to it.
	if model.Input.ValueString() != configuredInput {
		t.Fatalf("input = %v, want unchanged %v", model.Input.ValueString(), configuredInput)
	}
}
