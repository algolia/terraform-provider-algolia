package ingestion

import (
	"testing"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExpandTransformationCreate_Code(t *testing.T) {
	model := &TransformationResourceModel{
		Name: types.StringValue("my-code-transformation"),
		Code: types.StringValue("function transform(record) { return record; }"),
		Type: types.StringValue(string(ingestionapi.TRANSFORMATION_TYPE_CODE)),
	}

	create, diags := expandTransformationCreate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if create.Name != "my-code-transformation" {
		t.Fatalf("name = %v, want my-code-transformation", create.Name)
	}
	if create.Code == nil || *create.Code != "function transform(record) { return record; }" {
		t.Fatalf("code = %#v, want the configured source", create.Code)
	}
	if create.Type == nil || *create.Type != ingestionapi.TRANSFORMATION_TYPE_CODE {
		t.Fatalf("type = %#v, want code", create.Type)
	}
	if create.Input != nil {
		t.Fatalf("expected input to be nil, got %#v", create.Input)
	}
	if create.AuthenticationIDs != nil {
		t.Fatalf("expected authenticationIDs to be nil, got %#v", create.AuthenticationIDs)
	}
}

func TestExpandTransformationCreate_NoCodeInput(t *testing.T) {
	model := &TransformationResourceModel{
		Name:  types.StringValue("my-no-code-transformation"),
		Type:  types.StringValue(string(ingestionapi.TRANSFORMATION_TYPE_NO_CODE)),
		Input: types.StringValue(`{"steps": [{"action": "addField"}]}`),
	}

	create, diags := expandTransformationCreate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if create.Input == nil || create.Input.TransformationNoCode == nil {
		t.Fatalf("expected input to decode into TransformationNoCode, got %#v", create.Input)
	}
	if len(create.Input.TransformationNoCode.Steps) != 1 {
		t.Fatalf("steps = %#v, want 1 step", create.Input.TransformationNoCode.Steps)
	}
}

func TestExpandTransformationCreate_WithDescriptionAndAuthenticationIDs(t *testing.T) {
	model := &TransformationResourceModel{
		Name:        types.StringValue("my-transformation"),
		Description: types.StringValue("does a thing"),
		AuthenticationIDs: mustList(t,
			types.StringValue("auth-1"),
			types.StringValue("auth-2"),
		),
	}

	create, diags := expandTransformationCreate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if create.Description == nil || *create.Description != "does a thing" {
		t.Fatalf("description = %#v, want 'does a thing'", create.Description)
	}
	if len(create.AuthenticationIDs) != 2 || create.AuthenticationIDs[0] != "auth-1" || create.AuthenticationIDs[1] != "auth-2" {
		t.Fatalf("authenticationIDs = %#v, want [auth-1 auth-2]", create.AuthenticationIDs)
	}
}

func TestExpandTransformationCreate_NoTypeIsNil(t *testing.T) {
	model := &TransformationResourceModel{
		Name: types.StringValue("my-transformation"),
		Type: types.StringNull(),
	}

	create, diags := expandTransformationCreate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if create.Type != nil {
		t.Fatalf("expected type to be nil, got %#v", create.Type)
	}
}

func TestExpandTransformationCreate_InvalidInputJSON(t *testing.T) {
	model := &TransformationResourceModel{
		Name:  types.StringValue("broken"),
		Input: types.StringValue(`{not valid json`),
	}

	_, diags := expandTransformationCreate(model)
	if !diags.HasError() {
		t.Fatal("expected a diagnostic error for invalid input JSON")
	}
}

func TestExpandTransformationInput_NullAndEmptyReturnNil(t *testing.T) {
	t.Run("null", func(t *testing.T) {
		input, diags := expandTransformationInput(types.StringNull())
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if input != nil {
			t.Fatalf("expected nil input, got %#v", input)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		input, diags := expandTransformationInput(types.StringValue(""))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if input != nil {
			t.Fatalf("expected nil input, got %#v", input)
		}
	})
}

func TestExpandAuthenticationIDs_NullAndUnknownReturnNil(t *testing.T) {
	t.Run("null", func(t *testing.T) {
		if got := expandAuthenticationIDs(types.ListNull(types.StringType)); got != nil {
			t.Fatalf("expected nil, got %#v", got)
		}
	})

	t.Run("unknown", func(t *testing.T) {
		if got := expandAuthenticationIDs(types.ListUnknown(types.StringType)); got != nil {
			t.Fatalf("expected nil, got %#v", got)
		}
	})

	t.Run("unknown element", func(t *testing.T) {
		list, diags := types.ListValue(types.StringType, []attr.Value{types.StringUnknown()})
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics building list: %v", diags)
		}
		if got := expandAuthenticationIDs(list); got != nil {
			t.Fatalf("expected nil when an element is unknown, got %#v", got)
		}
	})
}

func TestExpandAuthenticationIDs_EmptyListReturnsEmptySlice(t *testing.T) {
	list := mustList(t)

	got := expandAuthenticationIDs(list)
	if got == nil {
		t.Fatal("expected a non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Fatalf("expected an empty slice, got %#v", got)
	}
}
