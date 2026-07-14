package allowedsources

import (
	"testing"

	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestAllowedSourcesSchemas_RegisterExpectedAttributes(t *testing.T) {
	resourceSchema := allowedSourcesResourceSchema()

	idAttr, ok := resourceSchema.Attributes["id"].(resourceschema.StringAttribute)
	if !ok || !idAttr.Computed {
		t.Fatal("expected id to be a computed string attribute")
	}

	sourceAttr, ok := resourceSchema.Attributes["source"].(resourceschema.SetNestedAttribute)
	if !ok || !sourceAttr.Required {
		t.Fatal("expected source to be a required set nested attribute")
	}

	sourceStringAttr, ok := sourceAttr.NestedObject.Attributes["source"].(resourceschema.StringAttribute)
	if !ok || !sourceStringAttr.Required {
		t.Fatal("expected nested source to be a required string attribute")
	}

	descriptionAttr, ok := sourceAttr.NestedObject.Attributes["description"].(resourceschema.StringAttribute)
	if !ok || !descriptionAttr.Optional {
		t.Fatal("expected nested description to be an optional string attribute")
	}

	dataSourceSchema := allowedSourcesDataSourceSchema()

	dsIDAttr, ok := dataSourceSchema.Attributes["id"].(datasourceschema.StringAttribute)
	if !ok || !dsIDAttr.Computed {
		t.Fatal("expected data source id to be computed")
	}

	dsSourceAttr, ok := dataSourceSchema.Attributes["source"].(datasourceschema.SetNestedAttribute)
	if !ok || !dsSourceAttr.Computed {
		t.Fatal("expected data source source to be a computed set nested attribute")
	}

	dsSourceStringAttr, ok := dsSourceAttr.NestedObject.Attributes["source"].(datasourceschema.StringAttribute)
	if !ok || !dsSourceStringAttr.Computed {
		t.Fatal("expected data source nested source to be a computed string attribute")
	}
}
