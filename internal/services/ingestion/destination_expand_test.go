package ingestion

import (
	"testing"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExpandDestinationCreate_Search(t *testing.T) {
	model := &DestinationResourceModel{
		Type:  types.StringValue(string(ingestionapi.DESTINATION_TYPE_SEARCH)),
		Name:  types.StringValue("my-search-destination"),
		Input: types.StringValue(`{"indexName": "products"}`),
	}

	create, diags := expandDestinationCreate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if create.Type != ingestionapi.DESTINATION_TYPE_SEARCH {
		t.Fatalf("type = %v, want search", create.Type)
	}
	if create.Name != "my-search-destination" {
		t.Fatalf("name = %v, want my-search-destination", create.Name)
	}
	if create.AuthenticationID != nil {
		t.Fatalf("expected authenticationID to be nil, got %#v", create.AuthenticationID)
	}
	if create.Input.IndexName != "products" {
		t.Fatalf("indexName = %v, want products", create.Input.IndexName)
	}
	if create.TransformationIDs != nil {
		t.Fatalf("expected transformationIDs to be nil, got %#v", create.TransformationIDs)
	}
}

func TestExpandDestinationCreate_WithAuthenticationIDAndTransformationIDs(t *testing.T) {
	model := &DestinationResourceModel{
		Type:             types.StringValue(string(ingestionapi.DESTINATION_TYPE_INSIGHTS)),
		Name:             types.StringValue("my-insights-destination"),
		Input:            types.StringValue(`{"indexName": "events"}`),
		AuthenticationID: types.StringValue("auth-123"),
		TransformationIDs: mustList(t,
			types.StringValue("transformation-1"),
			types.StringValue("transformation-2"),
		),
	}

	create, diags := expandDestinationCreate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if create.AuthenticationID == nil || *create.AuthenticationID != "auth-123" {
		t.Fatalf("authenticationID = %#v, want auth-123", create.AuthenticationID)
	}
	if len(create.TransformationIDs) != 2 || create.TransformationIDs[0] != "transformation-1" || create.TransformationIDs[1] != "transformation-2" {
		t.Fatalf("transformationIDs = %#v, want [transformation-1 transformation-2]", create.TransformationIDs)
	}
}

func TestExpandDestinationCreate_MissingInputIsError(t *testing.T) {
	model := &DestinationResourceModel{
		Type:  types.StringValue(string(ingestionapi.DESTINATION_TYPE_SEARCH)),
		Name:  types.StringValue("my-destination"),
		Input: types.StringNull(),
	}

	_, diags := expandDestinationCreate(model)
	if !diags.HasError() {
		t.Fatal("expected a diagnostic error: input is required for algolia_ingestion_destination")
	}
}

func TestExpandDestinationCreate_InvalidInputJSON(t *testing.T) {
	model := &DestinationResourceModel{
		Type:  types.StringValue(string(ingestionapi.DESTINATION_TYPE_SEARCH)),
		Name:  types.StringValue("broken"),
		Input: types.StringValue(`{not valid json`),
	}

	_, diags := expandDestinationCreate(model)
	if !diags.HasError() {
		t.Fatal("expected a diagnostic error for invalid input JSON")
	}
}

func TestExpandDestinationUpdate(t *testing.T) {
	model := &DestinationResourceModel{
		Name:             types.StringValue("renamed-destination"),
		Input:            types.StringValue(`{"indexName": "renamed-products"}`),
		AuthenticationID: types.StringValue("auth-456"),
		TransformationIDs: mustList(t,
			types.StringValue("transformation-1"),
		),
	}

	update, diags := expandDestinationUpdate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if update.Name == nil || *update.Name != "renamed-destination" {
		t.Fatalf("name = %#v, want renamed-destination", update.Name)
	}
	if update.AuthenticationID == nil || *update.AuthenticationID != "auth-456" {
		t.Fatalf("authenticationID = %#v, want auth-456", update.AuthenticationID)
	}
	if update.Input == nil || update.Input.IndexName == nil || *update.Input.IndexName != "renamed-products" {
		t.Fatalf("input = %#v, want indexName renamed-products", update.Input)
	}
	if len(update.TransformationIDs) != 1 || update.TransformationIDs[0] != "transformation-1" {
		t.Fatalf("transformationIDs = %#v, want [transformation-1]", update.TransformationIDs)
	}
}

func TestExpandDestinationUpdate_MissingInputIsError(t *testing.T) {
	model := &DestinationResourceModel{
		Name:  types.StringValue("renamed"),
		Input: types.StringNull(),
	}

	_, diags := expandDestinationUpdate(model)
	if !diags.HasError() {
		t.Fatal("expected a diagnostic error: input is required for algolia_ingestion_destination")
	}
}

func TestExpandDestinationUpdate_InvalidInputJSON(t *testing.T) {
	model := &DestinationResourceModel{
		Name:  types.StringValue("renamed"),
		Input: types.StringValue(`not json at all`),
	}

	_, diags := expandDestinationUpdate(model)
	if !diags.HasError() {
		t.Fatal("expected a diagnostic error for invalid input JSON")
	}
}

func TestExpandTransformationIDs_NullAndUnknownReturnNil(t *testing.T) {
	t.Run("null", func(t *testing.T) {
		if got := expandTransformationIDs(types.ListNull(types.StringType)); got != nil {
			t.Fatalf("expected nil, got %#v", got)
		}
	})

	t.Run("unknown", func(t *testing.T) {
		if got := expandTransformationIDs(types.ListUnknown(types.StringType)); got != nil {
			t.Fatalf("expected nil, got %#v", got)
		}
	})
}

func TestExpandTransformationIDs_EmptyListReturnsEmptySlice(t *testing.T) {
	list := mustList(t)

	got := expandTransformationIDs(list)
	if got == nil {
		t.Fatal("expected a non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Fatalf("expected an empty slice, got %#v", got)
	}
}

// mustList builds a types.List of strings for tests, failing the test on
// error rather than returning diagnostics through every call site.
func mustList(t *testing.T, values ...attr.Value) types.List {
	t.Helper()

	list, diags := types.ListValue(types.StringType, values)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics building list: %v", diags)
	}

	return list
}
