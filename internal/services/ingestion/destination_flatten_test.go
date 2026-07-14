package ingestion

import (
	"testing"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenDestination_PopulatesFields(t *testing.T) {
	authID := "auth-1"
	destination := &ingestionapi.Destination{
		DestinationID:     "destination-1",
		Type:              ingestionapi.DESTINATION_TYPE_SEARCH,
		Name:              "my-destination",
		AuthenticationID:  &authID,
		TransformationIDs: []string{"transformation-1", "transformation-2"},
		Input:             *ingestionapi.NewDestinationInput("products"),
		CreatedAt:         "2024-01-01T00:00:00Z",
		UpdatedAt:         "2024-01-02T00:00:00Z",
	}

	var model DestinationResourceModel
	diags := flattenDestination(destination, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.ID.ValueString() != "destination-1" {
		t.Fatalf("id = %v, want destination-1", model.ID.ValueString())
	}
	if model.DestinationID.ValueString() != "destination-1" {
		t.Fatalf("destination_id = %v, want destination-1", model.DestinationID.ValueString())
	}
	if model.Type.ValueString() != "search" {
		t.Fatalf("type = %v, want search", model.Type.ValueString())
	}
	if model.Name.ValueString() != "my-destination" {
		t.Fatalf("name = %v, want my-destination", model.Name.ValueString())
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

	elements := model.TransformationIDs.Elements()
	if len(elements) != 2 {
		t.Fatalf("transformation_ids has %d elements, want 2", len(elements))
	}
	if elements[0].(types.String).ValueString() != "transformation-1" || elements[1].(types.String).ValueString() != "transformation-2" {
		t.Fatalf("transformation_ids = %#v, want [transformation-1 transformation-2]", elements)
	}

	if model.Input.ValueString() != `{"indexName":"products"}` {
		t.Fatalf("input = %v, want {\"indexName\":\"products\"}", model.Input.ValueString())
	}
}

func TestFlattenDestination_NoAuthenticationIDIsNull(t *testing.T) {
	destination := &ingestionapi.Destination{
		DestinationID: "destination-2",
		Type:          ingestionapi.DESTINATION_TYPE_SEARCH,
		Name:          "my-destination",
		Input:         *ingestionapi.NewDestinationInput("products"),
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
	}

	var model DestinationResourceModel
	diags := flattenDestination(destination, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.AuthenticationID.IsNull() {
		t.Fatalf("authentication_id = %#v, want null", model.AuthenticationID)
	}
}

func TestFlattenDestination_NoTransformationIDsIsNull(t *testing.T) {
	destination := &ingestionapi.Destination{
		DestinationID: "destination-3",
		Type:          ingestionapi.DESTINATION_TYPE_SEARCH,
		Name:          "my-destination",
		Input:         *ingestionapi.NewDestinationInput("products"),
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
	}

	var model DestinationResourceModel
	diags := flattenDestination(destination, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.TransformationIDs.IsNull() {
		t.Fatalf("transformation_ids = %#v, want null", model.TransformationIDs)
	}
}

// TestFlattenDestination_InputPreservedWhenSemanticallyEqual is the core
// regression test for the semantic-equality refresh behavior: like
// Source's input, GetDestination returns a destination's input in full, so
// naively overwriting the configured JSON string with the API's encoding
// on every Read would create a perpetual diff whenever the API echoes back
// the same configuration with different key ordering.
func TestFlattenDestination_InputPreservedWhenSemanticallyEqual(t *testing.T) {
	const configuredInput = `{"indexName": "products", "attributesToExclude": ["secret"]}`

	model := DestinationResourceModel{
		Input: types.StringValue(configuredInput),
	}

	destination := &ingestionapi.Destination{
		DestinationID: "destination-4",
		Type:          ingestionapi.DESTINATION_TYPE_SEARCH,
		Name:          "my-destination",
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
		// Same data as configuredInput, but produced independently by the
		// Go client - simulates the API echoing back a semantically
		// identical encoding.
		Input: *ingestionapi.NewDestinationInput("products",
			ingestionapi.WithDestinationInputAttributesToExclude([]string{"secret"}),
		),
	}

	diags := flattenDestination(destination, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Input.ValueString() != configuredInput {
		t.Fatalf("input = %v, want unchanged configured value %v", model.Input.ValueString(), configuredInput)
	}
}

// TestFlattenDestination_InputAdoptsAPIValueWhenDifferent covers the
// opposite case: when the API's input is not semantically equal to what's
// configured (e.g. drift introduced out-of-band), flattenDestination must
// adopt the API's value so Terraform actually reports the difference.
func TestFlattenDestination_InputAdoptsAPIValueWhenDifferent(t *testing.T) {
	model := DestinationResourceModel{
		Input: types.StringValue(`{"indexName": "old-products"}`),
	}

	destination := &ingestionapi.Destination{
		DestinationID: "destination-5",
		Type:          ingestionapi.DESTINATION_TYPE_SEARCH,
		Name:          "my-destination",
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
		Input:         *ingestionapi.NewDestinationInput("new-products"),
	}

	diags := flattenDestination(destination, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Input.ValueString() != `{"indexName":"new-products"}` {
		t.Fatalf("input = %v, want the API's new value", model.Input.ValueString())
	}
}

func TestFlattenDestinationDataSource_PopulatesInputAndTransformationIDs(t *testing.T) {
	destination := &ingestionapi.Destination{
		DestinationID:     "destination-6",
		Type:              ingestionapi.DESTINATION_TYPE_SEARCH,
		Name:              "my-destination",
		TransformationIDs: []string{"transformation-1"},
		Input:             *ingestionapi.NewDestinationInput("products"),
		CreatedAt:         "2024-01-01T00:00:00Z",
		UpdatedAt:         "2024-01-01T00:00:00Z",
	}

	var model DestinationDataSourceModel
	diags := flattenDestinationDataSource(destination, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Input.ValueString() != `{"indexName":"products"}` {
		t.Fatalf("input = %v, want the encoded API value", model.Input.ValueString())
	}

	elements := model.TransformationIDs.Elements()
	if len(elements) != 1 || elements[0].(types.String).ValueString() != "transformation-1" {
		t.Fatalf("transformation_ids = %#v, want [transformation-1]", elements)
	}
}

func TestFlattenDestinationDataSource_NoTransformationIDsIsNull(t *testing.T) {
	destination := &ingestionapi.Destination{
		DestinationID: "destination-7",
		Type:          ingestionapi.DESTINATION_TYPE_SEARCH,
		Name:          "my-destination",
		Input:         *ingestionapi.NewDestinationInput("products"),
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
	}

	var model DestinationDataSourceModel
	diags := flattenDestinationDataSource(destination, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.TransformationIDs.IsNull() {
		t.Fatalf("transformation_ids = %#v, want null", model.TransformationIDs)
	}
}

// TestExpandFlattenDestination_RoundTrip exercises the JSON round trip
// end-to-end: JSON string -> DestinationInput (Create) -> ... ->
// Destination response -> flatten, mirroring how the real resource
// lifecycle uses these two halves of the JSON-encoded-field pattern
// together.
func TestExpandFlattenDestination_RoundTrip(t *testing.T) {
	const configuredInput = `{"indexName": "products"}`

	model := &DestinationResourceModel{
		Type:  types.StringValue(string(ingestionapi.DESTINATION_TYPE_SEARCH)),
		Name:  types.StringValue("round-trip-destination"),
		Input: types.StringValue(configuredInput),
	}

	create, diags := expandDestinationCreate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics expanding: %v", diags)
	}

	// The API would echo back the destinationID/createdAt/updatedAt and the
	// input in full; simulate that response using the same value produced
	// by expandDestinationCreate.
	destination := &ingestionapi.Destination{
		DestinationID: "destination-8",
		Type:          create.Type,
		Name:          create.Name,
		Input:         create.Input,
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
	}

	diags = flattenDestination(destination, model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics flattening: %v", diags)
	}

	if model.DestinationID.ValueString() != "destination-8" {
		t.Fatalf("destination_id = %v, want destination-8", model.DestinationID.ValueString())
	}
	// Input survives the round trip unchanged, exactly as configured, since
	// the API's encoding is semantically equal to it.
	if model.Input.ValueString() != configuredInput {
		t.Fatalf("input = %v, want unchanged %v", model.Input.ValueString(), configuredInput)
	}
}
