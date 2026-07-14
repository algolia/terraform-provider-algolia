package ingestion

import (
	"testing"

	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestAuthenticationResourceSchema_InputIsRequiredAndSensitive(t *testing.T) {
	s := authenticationResourceSchema()

	inputAttr, ok := s.Attributes["input"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatal("expected input to be a string attribute")
	}
	if !inputAttr.Required {
		t.Fatal("expected input to be required")
	}
	if !inputAttr.Sensitive {
		t.Fatal("expected input to be sensitive")
	}

	typeAttr, ok := s.Attributes["type"].(resourceschema.StringAttribute)
	if !ok || !typeAttr.Required {
		t.Fatal("expected type to be a required string attribute")
	}
	if len(typeAttr.PlanModifiers) == 0 {
		t.Fatal("expected type to have a RequiresReplace plan modifier")
	}

	idAttr, ok := s.Attributes["id"].(resourceschema.StringAttribute)
	if !ok || !idAttr.Computed {
		t.Fatal("expected id to be a computed string attribute")
	}

	authIDAttr, ok := s.Attributes["authentication_id"].(resourceschema.StringAttribute)
	if !ok || !authIDAttr.Computed {
		t.Fatal("expected authentication_id to be a computed string attribute")
	}

	platformAttr, ok := s.Attributes["platform"].(resourceschema.StringAttribute)
	if !ok || !platformAttr.Optional {
		t.Fatal("expected platform to be an optional string attribute")
	}
	if len(platformAttr.PlanModifiers) == 0 {
		t.Fatal("expected platform to have a RequiresReplace plan modifier (update endpoint cannot change it)")
	}
}

func TestAuthenticationDataSourceSchema_OmitsInput(t *testing.T) {
	s := authenticationDataSourceSchema()

	if _, ok := s.Attributes["input"]; ok {
		t.Fatal("expected the data source schema to omit input entirely")
	}

	authIDAttr, ok := s.Attributes["authentication_id"].(datasourceschema.StringAttribute)
	if !ok || !authIDAttr.Required {
		t.Fatal("expected authentication_id to be a required string attribute")
	}

	nameAttr, ok := s.Attributes["name"].(datasourceschema.StringAttribute)
	if !ok || !nameAttr.Computed {
		t.Fatal("expected name to be a computed string attribute")
	}
}

func TestAllowedAuthenticationTypeStrings_MatchesEnum(t *testing.T) {
	values := allowedAuthenticationTypeStrings()
	if len(values) != 7 {
		t.Fatalf("got %d allowed authentication types, want 7: %v", len(values), values)
	}
}

func TestAllowedPlatformStrings_MatchesEnum(t *testing.T) {
	values := allowedPlatformStrings()
	if len(values) != 3 {
		t.Fatalf("got %d allowed platforms, want 3: %v", len(values), values)
	}
}
