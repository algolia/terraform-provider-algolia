package ingestion

import (
	"testing"

	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestTransformationResourceSchema_NameIsRequired(t *testing.T) {
	s := transformationResourceSchema()

	nameAttr, ok := s.Attributes["name"].(resourceschema.StringAttribute)
	if !ok || !nameAttr.Required {
		t.Fatal("expected name to be a required string attribute")
	}
}

func TestTransformationResourceSchema_CodeIsOptionalAndNotSensitive(t *testing.T) {
	s := transformationResourceSchema()

	codeAttr, ok := s.Attributes["code"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatal("expected code to be a string attribute")
	}
	if !codeAttr.Optional {
		t.Fatal("expected code to be optional")
	}
	if codeAttr.Required {
		t.Fatal("expected code to not be required")
	}
	// Computed on purpose: the API derives `code` from `input.code` and returns
	// it, so a null plan value would abort the apply. `code` and `input` are
	// mutually exclusive in the API, which the schema enforces with
	// ConflictsWith.
	if !codeAttr.Computed {
		t.Fatal("expected code to be computed: the API derives it from input.code")
	}
	if codeAttr.Sensitive {
		t.Fatal("expected code to not be sensitive: it is configuration, not a secret")
	}
}

func TestTransformationResourceSchema_TypeIsOptionalWithoutReplace(t *testing.T) {
	s := transformationResourceSchema()

	typeAttr, ok := s.Attributes["type"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatal("expected type to be a string attribute")
	}
	if !typeAttr.Optional {
		t.Fatal("expected type to be optional")
	}
	if len(typeAttr.PlanModifiers) != 0 {
		t.Fatal("expected type to have no RequiresReplace plan modifier: UpdateTransformation accepts a `type` field")
	}
	// Deliberately not Computed. Computed requires UseStateForUnknown to avoid
	// planning as "known after apply" forever, and that combination replays the
	// prior type on an update - so switching an input-based transformation to
	// `code` while omitting `type` sent the old type alongside the new code, which
	// the API rejects. The derived value is dropped on read instead.
	if typeAttr.Computed {
		t.Fatal("expected type to not be computed: a computed type is replayed on update and contradicts a code-only transformation")
	}
}

func TestTransformationResourceSchema_InputIsOptionalAndNotSensitive(t *testing.T) {
	s := transformationResourceSchema()

	inputAttr, ok := s.Attributes["input"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatal("expected input to be a string attribute")
	}
	if !inputAttr.Optional {
		t.Fatal("expected input to be optional: a transformation's logic can be supplied via `code` instead")
	}
	if inputAttr.Required || inputAttr.Computed {
		t.Fatal("expected input to be neither required nor computed")
	}
	if inputAttr.Sensitive {
		t.Fatal("expected input to not be sensitive: it is configuration, not a secret")
	}
}

func TestTransformationResourceSchema_DescriptionIsOptional(t *testing.T) {
	s := transformationResourceSchema()

	descAttr, ok := s.Attributes["description"].(resourceschema.StringAttribute)
	if !ok || !descAttr.Optional {
		t.Fatal("expected description to be an optional string attribute")
	}
}

func TestTransformationResourceSchema_AuthenticationIDsIsOptionalList(t *testing.T) {
	s := transformationResourceSchema()

	authIDsAttr, ok := s.Attributes["authentication_ids"].(resourceschema.ListAttribute)
	if !ok {
		t.Fatal("expected authentication_ids to be a list attribute")
	}
	if !authIDsAttr.Optional {
		t.Fatal("expected authentication_ids to be optional")
	}
	if authIDsAttr.Required || authIDsAttr.Computed {
		t.Fatal("expected authentication_ids to be neither required nor computed")
	}
}

func TestTransformationResourceSchema_IDAndTransformationIDAreComputed(t *testing.T) {
	s := transformationResourceSchema()

	idAttr, ok := s.Attributes["id"].(resourceschema.StringAttribute)
	if !ok || !idAttr.Computed {
		t.Fatal("expected id to be a computed string attribute")
	}

	transformationIDAttr, ok := s.Attributes["transformation_id"].(resourceschema.StringAttribute)
	if !ok || !transformationIDAttr.Computed {
		t.Fatal("expected transformation_id to be a computed string attribute")
	}
}

func TestTransformationDataSourceSchema_TransformationIDIsRequired(t *testing.T) {
	s := transformationDataSourceSchema()

	transformationIDAttr, ok := s.Attributes["transformation_id"].(datasourceschema.StringAttribute)
	if !ok || !transformationIDAttr.Required {
		t.Fatal("expected transformation_id to be a required string attribute")
	}

	codeAttr, ok := s.Attributes["code"].(datasourceschema.StringAttribute)
	if !ok || !codeAttr.Computed {
		t.Fatal("expected code to be a computed string attribute")
	}

	inputAttr, ok := s.Attributes["input"].(datasourceschema.StringAttribute)
	if !ok || !inputAttr.Computed {
		t.Fatal("expected input to be a computed string attribute")
	}

	authIDsAttr, ok := s.Attributes["authentication_ids"].(datasourceschema.ListAttribute)
	if !ok || !authIDsAttr.Computed {
		t.Fatal("expected authentication_ids to be a computed list attribute")
	}
}

func TestAllowedTransformationTypeStrings_MatchesEnum(t *testing.T) {
	// Assert the known baseline values are present rather than an exact
	// count, so adding a new transformation type upstream doesn't break
	// this test.
	assertContains(t, "transformation types", allowedTransformationTypeStrings(), "code", "noCode")
}
