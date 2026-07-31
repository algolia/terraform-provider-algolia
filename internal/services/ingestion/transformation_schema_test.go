package ingestion

import (
	"context"
	"testing"

	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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
	ctx := context.Background()
	s := transformationResourceSchema()

	typeAttr, ok := s.Attributes["type"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatal("expected type to be a string attribute")
	}
	if !typeAttr.Optional {
		t.Fatal("expected type to be optional")
	}
	// Computed as well: a transformation defined through the legacy `code`
	// attribute sets no type of its own, and the API derives one.
	if !typeAttr.Computed {
		t.Fatal("expected type to be computed: the API derives it for a code-only transformation")
	}

	// Asserted behaviourally rather than by counting modifiers, because `type`
	// carries UseStateForUnknown to stop an unconfigured value planning as "known
	// after apply" forever. What must stay true is that changing the type updates
	// in place: UpdateTransformation accepts a `type` field.
	changed := "noCode"
	minimal := resourceschema.Schema{
		Attributes: map[string]resourceschema.Attribute{
			"type": resourceschema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: typeAttr.PlanModifiers,
			},
		},
	}
	objectType := minimal.Type().TerraformType(ctx)
	raw := func(v string) tftypes.Value {
		return tftypes.NewValue(objectType, map[string]tftypes.Value{"type": tftypes.NewValue(tftypes.String, v)})
	}

	for _, modifier := range typeAttr.PlanModifiers {
		req := planmodifier.StringRequest{
			Path:        path.Root("type"),
			State:       tfsdk.State{Schema: minimal, Raw: raw("code")},
			Plan:        tfsdk.Plan{Schema: minimal, Raw: raw(changed)},
			Config:      tfsdk.Config{Schema: minimal, Raw: raw(changed)},
			StateValue:  types.StringValue("code"),
			PlanValue:   types.StringValue(changed),
			ConfigValue: types.StringValue(changed),
		}
		resp := &planmodifier.StringResponse{PlanValue: types.StringValue(changed)}

		modifier.PlanModifyString(ctx, req, resp)

		if resp.RequiresReplace {
			t.Fatal("changing type must not force replacement: UpdateTransformation accepts a `type` field")
		}
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
