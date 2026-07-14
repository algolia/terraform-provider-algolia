package allowedsources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func sourceSetValue(t *testing.T, models []SourceModel) types.Set {
	t.Helper()

	values := make([]attr.Value, 0, len(models))
	for _, m := range models {
		objVal, diags := types.ObjectValueFrom(context.Background(), sourceAttrTypes, &m)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics building source object: %v", diags)
		}
		values = append(values, objVal)
	}

	setVal, diags := types.SetValue(types.ObjectType{AttrTypes: sourceAttrTypes}, values)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics building source set: %v", diags)
	}

	return setVal
}

func TestExpandSources_Null(t *testing.T) {
	// A null/unknown set must surface an explicit diagnostic rather than
	// silently expanding to an empty ReplaceSources payload.
	_, diags := expandSources(context.Background(), types.SetNull(types.ObjectType{AttrTypes: sourceAttrTypes}))
	if !diags.HasError() {
		t.Fatalf("expected an error diagnostic for a null set, got none")
	}
}

func TestExpandSources_Empty(t *testing.T) {
	// An empty set must surface an explicit diagnostic (the API rejects an
	// empty allowlist), not silently produce zero sources.
	_, diags := expandSources(context.Background(), sourceSetValue(t, nil))
	if !diags.HasError() {
		t.Fatalf("expected an error diagnostic for an empty set, got none")
	}
}

func TestExpandSources_WithoutDescription(t *testing.T) {
	sources, diags := expandSources(context.Background(), sourceSetValue(t, []SourceModel{
		{Source: types.StringValue("1.2.3.4"), Description: types.StringNull()},
	}))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %#v", sources)
	}
	if sources[0].GetSource() != "1.2.3.4" {
		t.Fatalf("source = %q, want 1.2.3.4", sources[0].GetSource())
	}
	if sources[0].HasDescription() {
		t.Fatalf("expected no description, got %#v", sources[0].Description)
	}
}

func TestExpandSources_WithDescription(t *testing.T) {
	sources, diags := expandSources(context.Background(), sourceSetValue(t, []SourceModel{
		{Source: types.StringValue("10.0.0.0/24"), Description: types.StringValue("office network")},
	}))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %#v", sources)
	}
	if !sources[0].HasDescription() || sources[0].GetDescription() != "office network" {
		t.Fatalf("description = %q, want %q", sources[0].GetDescription(), "office network")
	}
}

func TestExpandSources_EmptyDescriptionOmitted(t *testing.T) {
	sources, diags := expandSources(context.Background(), sourceSetValue(t, []SourceModel{
		{Source: types.StringValue("1.2.3.4"), Description: types.StringValue("")},
	}))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %#v", sources)
	}
	if sources[0].HasDescription() {
		t.Fatalf("expected an empty description to be omitted, got %#v", sources[0].Description)
	}
}

func TestExpandSources_MultipleSortedBySource(t *testing.T) {
	sources, diags := expandSources(context.Background(), sourceSetValue(t, []SourceModel{
		{Source: types.StringValue("10.0.0.0/24"), Description: types.StringValue("b")},
		{Source: types.StringValue("1.2.3.4"), Description: types.StringValue("a")},
	}))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %#v", sources)
	}
	if sources[0].GetSource() != "1.2.3.4" || sources[1].GetSource() != "10.0.0.0/24" {
		t.Fatalf("expected sources sorted by source value, got %#v", sources)
	}
}
