package ingestion

import (
	"testing"

	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestSourceResourceSchema_TypeIsRequiredWithReplace(t *testing.T) {
	s := sourceResourceSchema()

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

func TestSourceResourceSchema_InputIsOptionalAndNotSensitive(t *testing.T) {
	s := sourceResourceSchema()

	inputAttr, ok := s.Attributes["input"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatal("expected input to be a string attribute")
	}
	if !inputAttr.Optional {
		t.Fatal("expected input to be optional")
	}
	if inputAttr.Required {
		t.Fatal("expected input to not be required")
	}
	if inputAttr.Sensitive {
		t.Fatal("expected input to not be sensitive: it is configuration, not a secret")
	}
}

func TestSourceResourceSchema_AuthenticationIDIsOptional(t *testing.T) {
	s := sourceResourceSchema()

	authIDAttr, ok := s.Attributes["authentication_id"].(resourceschema.StringAttribute)
	if !ok || !authIDAttr.Optional {
		t.Fatal("expected authentication_id to be an optional string attribute")
	}
	if authIDAttr.Required || authIDAttr.Computed {
		t.Fatal("expected authentication_id to be neither required nor computed")
	}
}

func TestSourceResourceSchema_IDAndSourceIDAreComputed(t *testing.T) {
	s := sourceResourceSchema()

	idAttr, ok := s.Attributes["id"].(resourceschema.StringAttribute)
	if !ok || !idAttr.Computed {
		t.Fatal("expected id to be a computed string attribute")
	}

	sourceIDAttr, ok := s.Attributes["source_id"].(resourceschema.StringAttribute)
	if !ok || !sourceIDAttr.Computed {
		t.Fatal("expected source_id to be a computed string attribute")
	}
}

func TestSourceDataSourceSchema_SourceIDIsRequired(t *testing.T) {
	s := sourceDataSourceSchema()

	sourceIDAttr, ok := s.Attributes["source_id"].(datasourceschema.StringAttribute)
	if !ok || !sourceIDAttr.Required {
		t.Fatal("expected source_id to be a required string attribute")
	}

	inputAttr, ok := s.Attributes["input"].(datasourceschema.StringAttribute)
	if !ok || !inputAttr.Computed {
		t.Fatal("expected input to be a computed string attribute")
	}

	nameAttr, ok := s.Attributes["name"].(datasourceschema.StringAttribute)
	if !ok || !nameAttr.Computed {
		t.Fatal("expected name to be a computed string attribute")
	}
}

func TestAllowedSourceTypeStrings_MatchesEnum(t *testing.T) {
	// Assert the known baseline values are present rather than an exact
	// count, so adding a new source type upstream doesn't break this test.
	assertContains(t, "source types", allowedSourceTypeStrings(),
		"algoliaIndex", "bigcommerce", "bigquery", "commercetools", "csv",
		"docker", "ga4BigqueryExport", "json", "shopify", "push")
}
