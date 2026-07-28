package ingestion

import (
	"testing"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExpandSourceCreate_CSV(t *testing.T) {
	model := &SourceResourceModel{
		Type:  types.StringValue(string(ingestionapi.SOURCE_TYPE_CSV)),
		Name:  types.StringValue("my-csv-source"),
		Input: types.StringValue(`{"url": "https://example.com/data.csv"}`),
	}

	create, diags := expandSourceCreate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if create.Type != ingestionapi.SOURCE_TYPE_CSV {
		t.Fatalf("type = %v, want csv", create.Type)
	}
	if create.Name != "my-csv-source" {
		t.Fatalf("name = %v, want my-csv-source", create.Name)
	}
	if create.AuthenticationID != nil {
		t.Fatalf("expected authenticationID to be nil, got %#v", create.AuthenticationID)
	}

	if create.Input == nil || create.Input.SourceCSV == nil {
		t.Fatalf("expected input to decode into SourceCSV, got %#v", create.Input)
	}
	if create.Input.SourceCSV.Url != "https://example.com/data.csv" {
		t.Fatalf("url = %v, want https://example.com/data.csv", create.Input.SourceCSV.Url)
	}
}

func TestExpandSourceCreate_WithAuthenticationID(t *testing.T) {
	model := &SourceResourceModel{
		Type:             types.StringValue(string(ingestionapi.SOURCE_TYPE_SHOPIFY)),
		Name:             types.StringValue("my-shopify-source"),
		AuthenticationID: types.StringValue("auth-123"),
	}

	create, diags := expandSourceCreate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if create.AuthenticationID == nil || *create.AuthenticationID != "auth-123" {
		t.Fatalf("authenticationID = %#v, want auth-123", create.AuthenticationID)
	}
}

func TestExpandSourceCreate_NoInput(t *testing.T) {
	// A "push" source needs no configuration at all.
	model := &SourceResourceModel{
		Type:  types.StringValue(string(ingestionapi.SOURCE_TYPE_PUSH)),
		Name:  types.StringValue("my-push-source"),
		Input: types.StringNull(),
	}

	create, diags := expandSourceCreate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if create.Input != nil {
		t.Fatalf("expected input to be nil, got %#v", create.Input)
	}
}

func TestExpandSourceCreate_InvalidInputJSON(t *testing.T) {
	model := &SourceResourceModel{
		Type:  types.StringValue(string(ingestionapi.SOURCE_TYPE_CSV)),
		Name:  types.StringValue("broken"),
		Input: types.StringValue(`{not valid json`),
	}

	_, diags := expandSourceCreate(model)
	if !diags.HasError() {
		t.Fatal("expected a diagnostic error for invalid input JSON")
	}
}

func TestExpandSourceUpdate(t *testing.T) {
	model := &SourceResourceModel{
		// `type` is RequiresReplace, so the plan always carries the source's
		// real type - which is what selects the SourceUpdateInput variant.
		Type:             types.StringValue(string(ingestionapi.SOURCE_TYPE_CSV)),
		Name:             types.StringValue("renamed-source"),
		Input:            types.StringValue(`{"url": "https://example.com/renamed.csv"}`),
		AuthenticationID: types.StringValue("auth-456"),
	}

	update, diags := expandSourceUpdate(model, types.StringNull())
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if update.Name == nil || *update.Name != "renamed-source" {
		t.Fatalf("name = %#v, want renamed-source", update.Name)
	}
	if update.AuthenticationID == nil || *update.AuthenticationID != "auth-456" {
		t.Fatalf("authenticationID = %#v, want auth-456", update.AuthenticationID)
	}
	if update.Input == nil || update.Input.SourceCSV == nil {
		t.Fatalf("expected input to decode into SourceCSV, got %#v", update.Input)
	}
	if update.Input.SourceCSV.Url != "https://example.com/renamed.csv" {
		t.Fatalf("url = %v, want https://example.com/renamed.csv", update.Input.SourceCSV.Url)
	}
}

func TestExpandSourceUpdate_NoInput(t *testing.T) {
	model := &SourceResourceModel{
		Type:  types.StringValue(string(ingestionapi.SOURCE_TYPE_PUSH)),
		Name:  types.StringValue("renamed-push-source"),
		Input: types.StringNull(),
	}

	update, diags := expandSourceUpdate(model, types.StringNull())
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if update.Input != nil {
		t.Fatalf("expected input to be nil, got %#v", update.Input)
	}
}

func TestExpandSourceUpdate_InvalidInputJSON(t *testing.T) {
	model := &SourceResourceModel{
		Type:  types.StringValue(string(ingestionapi.SOURCE_TYPE_CSV)),
		Name:  types.StringValue("renamed"),
		Input: types.StringValue(`not json at all`),
	}

	_, diags := expandSourceUpdate(model, types.StringNull())
	if !diags.HasError() {
		t.Fatal("expected a diagnostic error for invalid input JSON")
	}
}

func TestExpandSourceInput_NullAndEmptyReturnNil(t *testing.T) {
	t.Run("null", func(t *testing.T) {
		input, diags := expandSourceInput(string(ingestionapi.SOURCE_TYPE_CSV), types.StringNull())
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if input != nil {
			t.Fatalf("expected nil input, got %#v", input)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		input, diags := expandSourceInput(string(ingestionapi.SOURCE_TYPE_CSV), types.StringValue(""))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if input != nil {
			t.Fatalf("expected nil input, got %#v", input)
		}
	})
}

func TestExpandSourceUpdateInput_NullAndEmptyReturnNil(t *testing.T) {
	t.Run("null", func(t *testing.T) {
		input, diags := expandSourceUpdateInput(string(ingestionapi.SOURCE_TYPE_CSV), types.StringNull())
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if input != nil {
			t.Fatalf("expected nil input, got %#v", input)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		input, diags := expandSourceUpdateInput(string(ingestionapi.SOURCE_TYPE_CSV), types.StringValue(""))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if input != nil {
			t.Fatalf("expected nil input, got %#v", input)
		}
	})
}
