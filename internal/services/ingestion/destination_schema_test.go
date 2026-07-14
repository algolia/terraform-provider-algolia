package ingestion

import (
	"testing"

	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestDestinationResourceSchema_TypeIsRequiredWithReplace(t *testing.T) {
	s := destinationResourceSchema()

	typeAttr, ok := s.Attributes["type"].(resourceschema.StringAttribute)
	if !ok || !typeAttr.Required {
		t.Fatal("expected type to be a required string attribute")
	}
	if len(typeAttr.PlanModifiers) == 0 {
		t.Fatal("expected type to have a RequiresReplace plan modifier")
	}

	nameAttr, ok := s.Attributes["name"].(resourceschema.StringAttribute)
	if !ok || !nameAttr.Required {
		t.Fatal("expected name to be a required string attribute")
	}
}

func TestDestinationResourceSchema_InputIsRequiredAndNotSensitive(t *testing.T) {
	s := destinationResourceSchema()

	inputAttr, ok := s.Attributes["input"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatal("expected input to be a string attribute")
	}
	if !inputAttr.Required {
		t.Fatal("expected input to be required: unlike algolia_ingestion_source, a destination always needs an indexName")
	}
	if inputAttr.Optional {
		t.Fatal("expected input to not be optional")
	}
	if inputAttr.Sensitive {
		t.Fatal("expected input to not be sensitive: it is configuration, not a secret")
	}
}

func TestDestinationResourceSchema_AuthenticationIDIsOptional(t *testing.T) {
	s := destinationResourceSchema()

	authIDAttr, ok := s.Attributes["authentication_id"].(resourceschema.StringAttribute)
	if !ok || !authIDAttr.Optional {
		t.Fatal("expected authentication_id to be an optional string attribute")
	}
	if authIDAttr.Required || authIDAttr.Computed {
		t.Fatal("expected authentication_id to be neither required nor computed")
	}
}

func TestDestinationResourceSchema_TransformationIDsIsOptionalList(t *testing.T) {
	s := destinationResourceSchema()

	tIDsAttr, ok := s.Attributes["transformation_ids"].(resourceschema.ListAttribute)
	if !ok {
		t.Fatal("expected transformation_ids to be a list attribute")
	}
	if !tIDsAttr.Optional {
		t.Fatal("expected transformation_ids to be optional")
	}
	if tIDsAttr.Required || tIDsAttr.Computed {
		t.Fatal("expected transformation_ids to be neither required nor computed")
	}
}

func TestDestinationResourceSchema_IDAndDestinationIDAreComputed(t *testing.T) {
	s := destinationResourceSchema()

	idAttr, ok := s.Attributes["id"].(resourceschema.StringAttribute)
	if !ok || !idAttr.Computed {
		t.Fatal("expected id to be a computed string attribute")
	}

	destinationIDAttr, ok := s.Attributes["destination_id"].(resourceschema.StringAttribute)
	if !ok || !destinationIDAttr.Computed {
		t.Fatal("expected destination_id to be a computed string attribute")
	}
}

func TestDestinationDataSourceSchema_DestinationIDIsRequired(t *testing.T) {
	s := destinationDataSourceSchema()

	destinationIDAttr, ok := s.Attributes["destination_id"].(datasourceschema.StringAttribute)
	if !ok || !destinationIDAttr.Required {
		t.Fatal("expected destination_id to be a required string attribute")
	}

	inputAttr, ok := s.Attributes["input"].(datasourceschema.StringAttribute)
	if !ok || !inputAttr.Computed {
		t.Fatal("expected input to be a computed string attribute")
	}

	nameAttr, ok := s.Attributes["name"].(datasourceschema.StringAttribute)
	if !ok || !nameAttr.Computed {
		t.Fatal("expected name to be a computed string attribute")
	}

	tIDsAttr, ok := s.Attributes["transformation_ids"].(datasourceschema.ListAttribute)
	if !ok || !tIDsAttr.Computed {
		t.Fatal("expected transformation_ids to be a computed list attribute")
	}
}

func TestAllowedDestinationTypeStrings_MatchesEnum(t *testing.T) {
	// Assert the known baseline values are present rather than an exact
	// count, so adding a new destination type upstream doesn't break this
	// test.
	assertContains(t, "destination types", allowedDestinationTypeStrings(), "search", "insights")
}
