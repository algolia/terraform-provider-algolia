package ingestion

import (
	"testing"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenTransformation_PopulatesFields(t *testing.T) {
	transformationType := ingestionapi.TRANSFORMATION_TYPE_CODE
	description := "does a thing"

	transformation := &ingestionapi.Transformation{
		TransformationID:  "transformation-1",
		Name:              "my-transformation",
		Code:              "function transform(record) { return record; }",
		Type:              &transformationType,
		Description:       &description,
		AuthenticationIDs: []string{"auth-1", "auth-2"},
		CreatedAt:         "2024-01-01T00:00:00Z",
		UpdatedAt:         "2024-01-02T00:00:00Z",
	}

	var model TransformationResourceModel
	diags := flattenTransformation(transformation, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.ID.ValueString() != "transformation-1" {
		t.Fatalf("id = %v, want transformation-1", model.ID.ValueString())
	}
	if model.TransformationID.ValueString() != "transformation-1" {
		t.Fatalf("transformation_id = %v, want transformation-1", model.TransformationID.ValueString())
	}
	if model.Name.ValueString() != "my-transformation" {
		t.Fatalf("name = %v, want my-transformation", model.Name.ValueString())
	}
	if model.Code.ValueString() != "function transform(record) { return record; }" {
		t.Fatalf("code = %v, want the transformation source", model.Code.ValueString())
	}
	if model.Type.ValueString() != "code" {
		t.Fatalf("type = %v, want code", model.Type.ValueString())
	}
	if model.Description.ValueString() != "does a thing" {
		t.Fatalf("description = %v, want 'does a thing'", model.Description.ValueString())
	}
	if model.CreatedAt.ValueString() != "2024-01-01T00:00:00Z" {
		t.Fatalf("created_at = %v, want 2024-01-01T00:00:00Z", model.CreatedAt.ValueString())
	}
	if model.UpdatedAt.ValueString() != "2024-01-02T00:00:00Z" {
		t.Fatalf("updated_at = %v, want 2024-01-02T00:00:00Z", model.UpdatedAt.ValueString())
	}

	elements := model.AuthenticationIDs.Elements()
	if len(elements) != 2 {
		t.Fatalf("authentication_ids has %d elements, want 2", len(elements))
	}
	if elements[0].(types.String).ValueString() != "auth-1" || elements[1].(types.String).ValueString() != "auth-2" {
		t.Fatalf("authentication_ids = %#v, want [auth-1 auth-2]", elements)
	}
}

// TestFlattenTransformation_EmptyCodeIsNull covers a no-code transformation,
// whose deprecated Code field the API returns as an empty string: the
// `code` attribute should surface as null rather than "", so an
// unconfigured `code` attribute doesn't drift.
func TestFlattenTransformation_EmptyCodeIsNull(t *testing.T) {
	transformationType := ingestionapi.TRANSFORMATION_TYPE_NO_CODE

	transformation := &ingestionapi.Transformation{
		TransformationID: "transformation-2",
		Name:             "my-no-code-transformation",
		Code:             "",
		Type:             &transformationType,
		Input:            ingestionapi.TransformationNoCodeAsTransformationInput(ingestionapi.NewTransformationNoCode([]map[string]any{{"action": "addField"}})),
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-01T00:00:00Z",
	}

	var model TransformationResourceModel
	diags := flattenTransformation(transformation, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.Code.IsNull() {
		t.Fatalf("code = %#v, want null", model.Code)
	}
}

func TestFlattenTransformation_NoTypeIsNull(t *testing.T) {
	transformation := &ingestionapi.Transformation{
		TransformationID: "transformation-3",
		Name:             "my-transformation",
		Code:             "function transform(record) { return record; }",
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-01T00:00:00Z",
	}

	var model TransformationResourceModel
	diags := flattenTransformation(transformation, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.Type.IsNull() {
		t.Fatalf("type = %#v, want null", model.Type)
	}
}

// TestFlattenTransformation_CodeSuppliedDoesNotAdoptDerivedInput covers a
// transformation whose logic is given through the legacy `code` attribute. The
// API derives an `input` from it and returns that, and adopting it put a value
// into state for an attribute the configuration left unset - which Terraform
// rejects with "Provider produced inconsistent result after apply: .input: was
// null, but now ...".
func TestFlattenTransformation_CodeSuppliedDoesNotAdoptDerivedInput(t *testing.T) {
	code := "function transform({ record }) { return record; }"
	transformation := &ingestionapi.Transformation{
		TransformationID: "transformation-code-only",
		Name:             "my-transformation",
		Code:             code,
		Input:            ingestionapi.TransformationCodeAsTransformationInput(ingestionapi.NewTransformationCode(code)),
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-01T00:00:00Z",
	}

	// What the operator configured: code, no input.
	model := TransformationResourceModel{Code: types.StringValue(code)}

	diags := flattenTransformation(transformation, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.Input.IsNull() {
		t.Errorf("input = %#v, want null: the operator configured code instead", model.Input)
	}
	// The same rule applies to type, and for a sharper reason: a type adopted here
	// is replayed on the next update, and sending it alongside `code` is rejected
	// by the API with "'input' is required if 'Type' is present".
	if !model.Type.IsNull() {
		t.Errorf("type = %#v, want null: adopting the derived type breaks the next update", model.Type)
	}
}

// TestFlattenTransformation_ExplicitTypeIsKeptAlongsideCode covers an operator
// who sets `type` themselves while using `code`. Their value is theirs to keep -
// the rule above only drops a type the API derived.
func TestFlattenTransformation_ExplicitTypeIsKeptAlongsideCode(t *testing.T) {
	code := "function transform({ record }) { return record; }"
	transformationType := ingestionapi.TRANSFORMATION_TYPE_CODE
	transformation := &ingestionapi.Transformation{
		TransformationID: "transformation-explicit-type",
		Name:             "my-transformation",
		Code:             code,
		Type:             &transformationType,
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-01T00:00:00Z",
	}

	model := TransformationResourceModel{
		Code: types.StringValue(code),
		Type: types.StringValue("code"),
	}

	diags := flattenTransformation(transformation, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Type.ValueString() != "code" {
		t.Errorf("type = %#v, want the configured value to survive", model.Type)
	}
}

// TestFlattenTransformation_ImportAdoptsDerivedInput is the other side of the
// same rule: an import has no configured code to defer to, so the API's input is
// the only thing that can populate state.
func TestFlattenTransformation_ImportAdoptsDerivedInput(t *testing.T) {
	code := "function transform({ record }) { return record; }"
	transformation := &ingestionapi.Transformation{
		TransformationID: "transformation-import",
		Name:             "my-transformation",
		Code:             code,
		Input:            ingestionapi.TransformationCodeAsTransformationInput(ingestionapi.NewTransformationCode(code)),
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-01T00:00:00Z",
	}

	// An import starts from an empty model: nothing configured at all.
	var model TransformationResourceModel

	diags := flattenTransformation(transformation, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Input.IsNull() {
		t.Error("input = null after import, want the value the API returned")
	}
}

func TestFlattenTransformation_NoAuthenticationIDsIsNull(t *testing.T) {
	transformation := &ingestionapi.Transformation{
		TransformationID: "transformation-4",
		Name:             "my-transformation",
		Code:             "function transform(record) { return record; }",
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-01T00:00:00Z",
	}

	var model TransformationResourceModel
	diags := flattenTransformation(transformation, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.AuthenticationIDs.IsNull() {
		t.Fatalf("authentication_ids = %#v, want null", model.AuthenticationIDs)
	}
}

func TestFlattenTransformation_NoDescriptionIsNull(t *testing.T) {
	transformation := &ingestionapi.Transformation{
		TransformationID: "transformation-5",
		Name:             "my-transformation",
		Code:             "function transform(record) { return record; }",
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-01T00:00:00Z",
	}

	var model TransformationResourceModel
	diags := flattenTransformation(transformation, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.Description.IsNull() {
		t.Fatalf("description = %#v, want null", model.Description)
	}
}

// TestFlattenTransformation_InputPreservedWhenSemanticallyEqual is the core
// regression test for the semantic-equality refresh behavior: like
// Source/Destination's input, GetTransformation returns a transformation's
// input in full, so naively overwriting the configured JSON string with the
// API's encoding on every Read would create a perpetual diff whenever the
// API echoes back the same configuration with different key/array
// ordering.
func TestFlattenTransformation_InputPreservedWhenSemanticallyEqual(t *testing.T) {
	const configuredInput = `{"steps": [{"action": "addField", "extra": "value"}]}`

	model := TransformationResourceModel{
		Input: types.StringValue(configuredInput),
	}

	transformation := &ingestionapi.Transformation{
		TransformationID: "transformation-6",
		Name:             "my-transformation",
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-01T00:00:00Z",
		// Same data as configuredInput, but produced independently by the Go
		// client - simulates the API echoing back a semantically identical
		// encoding.
		Input: ingestionapi.TransformationNoCodeAsTransformationInput(ingestionapi.NewTransformationNoCode(
			[]map[string]any{{"action": "addField", "extra": "value"}},
		)),
	}

	diags := flattenTransformation(transformation, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Input.ValueString() != configuredInput {
		t.Fatalf("input = %v, want unchanged configured value %v", model.Input.ValueString(), configuredInput)
	}
}

// TestFlattenTransformation_InputAdoptsAPIValueWhenDifferent covers the
// opposite case: when the API's input is not semantically equal to what's
// configured (e.g. drift introduced out-of-band), flattenTransformation
// must adopt the API's value so Terraform actually reports the difference.
func TestFlattenTransformation_InputAdoptsAPIValueWhenDifferent(t *testing.T) {
	model := TransformationResourceModel{
		Input: types.StringValue(`{"steps": [{"action": "old"}]}`),
	}

	transformation := &ingestionapi.Transformation{
		TransformationID: "transformation-7",
		Name:             "my-transformation",
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-01T00:00:00Z",
		Input: ingestionapi.TransformationNoCodeAsTransformationInput(ingestionapi.NewTransformationNoCode(
			[]map[string]any{{"action": "new"}},
		)),
	}

	diags := flattenTransformation(transformation, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Input.ValueString() != `{"steps":[{"action":"new"}]}` {
		t.Fatalf("input = %v, want the API's new value", model.Input.ValueString())
	}
}

// TestFlattenTransformation_NilAPIInputKeepsConfiguredValue covers a
// transformation whose API response has no `input` at all (e.g. one
// defined via the legacy `code` attribute): flattenTransformation must not
// clobber a configured `input` value with null just because the API
// omitted it from its response.
func TestFlattenTransformation_NilAPIInputKeepsConfiguredValue(t *testing.T) {
	model := TransformationResourceModel{
		Input: types.StringValue(`{"some": "config"}`),
	}

	transformation := &ingestionapi.Transformation{
		TransformationID: "transformation-8",
		Name:             "my-transformation",
		Code:             "function transform(record) { return record; }",
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-01T00:00:00Z",
		Input:            nil,
	}

	diags := flattenTransformation(transformation, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Input.ValueString() != `{"some": "config"}` {
		t.Fatalf("input = %v, want the previously configured value preserved", model.Input.ValueString())
	}
}

func TestFlattenTransformation_NilAPIInputAndUnconfiguredIsNull(t *testing.T) {
	model := TransformationResourceModel{
		Input: types.StringNull(),
	}

	transformation := &ingestionapi.Transformation{
		TransformationID: "transformation-9",
		Name:             "my-transformation",
		Code:             "function transform(record) { return record; }",
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-01T00:00:00Z",
	}

	diags := flattenTransformation(transformation, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.Input.IsNull() {
		t.Fatalf("input = %#v, want null", model.Input)
	}
}

func TestFlattenTransformationDataSource_PopulatesFields(t *testing.T) {
	transformation := &ingestionapi.Transformation{
		TransformationID:  "transformation-10",
		Name:              "my-transformation",
		Code:              "function transform(record) { return record; }",
		AuthenticationIDs: []string{"auth-1"},
		CreatedAt:         "2024-01-01T00:00:00Z",
		UpdatedAt:         "2024-01-01T00:00:00Z",
	}

	var model TransformationDataSourceModel
	diags := flattenTransformationDataSource(transformation, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Code.ValueString() != "function transform(record) { return record; }" {
		t.Fatalf("code = %v, want the transformation source", model.Code.ValueString())
	}

	elements := model.AuthenticationIDs.Elements()
	if len(elements) != 1 || elements[0].(types.String).ValueString() != "auth-1" {
		t.Fatalf("authentication_ids = %#v, want [auth-1]", elements)
	}

	if !model.Input.IsNull() {
		t.Fatalf("input = %#v, want null", model.Input)
	}
}

func TestFlattenTransformationDataSource_PopulatesInput(t *testing.T) {
	transformation := &ingestionapi.Transformation{
		TransformationID: "transformation-11",
		Name:             "my-transformation",
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-01T00:00:00Z",
		Input: ingestionapi.TransformationNoCodeAsTransformationInput(ingestionapi.NewTransformationNoCode(
			[]map[string]any{{"action": "addField"}},
		)),
	}

	var model TransformationDataSourceModel
	diags := flattenTransformationDataSource(transformation, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Input.ValueString() != `{"steps":[{"action":"addField"}]}` {
		t.Fatalf("input = %v, want the encoded API value", model.Input.ValueString())
	}
}

// TestExpandFlattenTransformation_CodeRoundTrip exercises the plain-string
// `code` attribute end-to-end: configured code -> TransformationCreate ->
// ... -> Transformation response -> flatten, mirroring how the real
// resource lifecycle uses expand/flatten together for a code-type
// transformation.
func TestExpandFlattenTransformation_CodeRoundTrip(t *testing.T) {
	const configuredCode = `function transform(record) { record.extra = true; return record; }`

	model := &TransformationResourceModel{
		Name: types.StringValue("round-trip-transformation"),
		Code: types.StringValue(configuredCode),
		Type: types.StringValue(string(ingestionapi.TRANSFORMATION_TYPE_CODE)),
	}

	create, diags := expandTransformationCreate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics expanding: %v", diags)
	}

	// The API would echo back the transformationID/createdAt/updatedAt and
	// the code in full; simulate that response using the same value
	// produced by expandTransformationCreate.
	transformation := &ingestionapi.Transformation{
		TransformationID: "transformation-12",
		Name:             create.Name,
		Code:             *create.Code,
		Type:             create.Type,
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-01T00:00:00Z",
	}

	diags = flattenTransformation(transformation, model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics flattening: %v", diags)
	}

	if model.TransformationID.ValueString() != "transformation-12" {
		t.Fatalf("transformation_id = %v, want transformation-12", model.TransformationID.ValueString())
	}
	if model.Code.ValueString() != configuredCode {
		t.Fatalf("code = %v, want unchanged %v", model.Code.ValueString(), configuredCode)
	}
}

// TestExpandFlattenTransformation_InputRoundTrip is the `input` analogue of
// TestExpandFlattenTransformation_CodeRoundTrip, for a no-code
// transformation.
func TestExpandFlattenTransformation_InputRoundTrip(t *testing.T) {
	const configuredInput = `{"steps": [{"action": "addField"}]}`

	model := &TransformationResourceModel{
		Name:  types.StringValue("round-trip-no-code-transformation"),
		Type:  types.StringValue(string(ingestionapi.TRANSFORMATION_TYPE_NO_CODE)),
		Input: types.StringValue(configuredInput),
	}

	create, diags := expandTransformationCreate(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics expanding: %v", diags)
	}

	transformation := &ingestionapi.Transformation{
		TransformationID: "transformation-13",
		Name:             create.Name,
		Type:             create.Type,
		Input:            create.Input,
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-01T00:00:00Z",
	}

	diags = flattenTransformation(transformation, model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics flattening: %v", diags)
	}

	// Input survives the round trip unchanged, exactly as configured, since
	// the API's encoding is semantically equal to it.
	if model.Input.ValueString() != configuredInput {
		t.Fatalf("input = %v, want unchanged %v", model.Input.ValueString(), configuredInput)
	}
}

func TestFlattenAuthenticationIDs(t *testing.T) {
	nullList := types.ListNull(types.StringType)
	emptyList, _ := types.ListValue(types.StringType, []attr.Value{})
	populatedPrev, _ := types.ListValue(types.StringType, []attr.Value{types.StringValue("auth-x")})

	// API empty + unset prior -> null.
	if list, _ := flattenAuthenticationIDs(nil, nullList); !list.IsNull() {
		t.Fatalf("empty API + null prior = %#v, want null", list)
	}
	// API empty + explicit empty prior -> preserved empty list (no diff).
	if list, _ := flattenAuthenticationIDs([]string{}, emptyList); list.IsNull() || len(list.Elements()) != 0 {
		t.Fatalf("empty API + [] prior = %#v, want empty list", list)
	}
	// API empty + non-empty prior -> null (real drift, cleared externally).
	if list, _ := flattenAuthenticationIDs(nil, populatedPrev); !list.IsNull() {
		t.Fatalf("empty API + populated prior = %#v, want null (drift)", list)
	}
	// API non-empty -> adopt API value.
	list, _ := flattenAuthenticationIDs([]string{"auth-y"}, nullList)
	if list.IsNull() || len(list.Elements()) != 1 {
		t.Fatalf("non-empty API = %#v, want 1 element", list)
	}
}

func TestFlattenCode(t *testing.T) {
	// API empty + unset prior -> null.
	if got := flattenCode("", types.StringNull()); !got.IsNull() {
		t.Fatalf("empty API + null prior = %#v, want null", got)
	}
	// API empty + explicit "" prior -> preserved "".
	if got := flattenCode("", types.StringValue("")); got.IsNull() || got.ValueString() != "" {
		t.Fatalf("empty API + \"\" prior = %#v, want empty string", got)
	}
	// API empty + non-empty prior -> null (drift, cleared externally).
	if got := flattenCode("", types.StringValue("old")); !got.IsNull() {
		t.Fatalf("empty API + non-empty prior = %#v, want null (drift)", got)
	}
	// API non-empty -> adopt API value.
	if got := flattenCode("new", types.StringValue("old")); got.ValueString() != "new" {
		t.Fatalf("non-empty API = %q, want new", got.ValueString())
	}
}
