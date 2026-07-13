package dictionary

import (
	"context"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func boolMapValue(t *testing.T, values map[string]bool) types.Map {
	t.Helper()

	attrValues := make(map[string]attr.Value, len(values))
	for k, v := range values {
		attrValues[k] = types.BoolValue(v)
	}

	return types.MapValueMust(types.BoolType, attrValues)
}

func disableStandardEntriesObject(t *testing.T, block DisableStandardEntriesModel) types.Object {
	t.Helper()

	objVal, diags := types.ObjectValueFrom(context.Background(), disableStandardEntriesAttrTypes, &block)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics building disable_standard_entries object: %v", diags)
	}

	return objVal
}

func TestExpandDictionarySettings_NullBlock(t *testing.T) {
	model := DictionarySettingsResourceModel{
		DisableStandardEntries: types.ObjectNull(disableStandardEntriesAttrTypes),
	}

	entries, diags := expandDictionarySettings(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if entries.Stopwords != nil || entries.Plurals != nil || entries.Compounds != nil {
		t.Fatalf("expected all maps nil for a null block, got %#v", entries)
	}
}

func TestExpandDictionarySettings_Empty(t *testing.T) {
	model := DictionarySettingsResourceModel{
		DisableStandardEntries: disableStandardEntriesObject(t, DisableStandardEntriesModel{
			Stopwords: types.MapNull(types.BoolType),
			Plurals:   types.MapNull(types.BoolType),
			Compounds: types.MapNull(types.BoolType),
		}),
	}

	entries, diags := expandDictionarySettings(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if entries.Stopwords != nil || entries.Plurals != nil || entries.Compounds != nil {
		t.Fatalf("expected all maps nil when every map is null, got %#v", entries)
	}
}

func TestExpandDictionarySettings_Partial(t *testing.T) {
	model := DictionarySettingsResourceModel{
		DisableStandardEntries: disableStandardEntriesObject(t, DisableStandardEntriesModel{
			Stopwords: boolMapValue(t, map[string]bool{"en": true}),
			Plurals:   types.MapNull(types.BoolType),
			Compounds: types.MapNull(types.BoolType),
		}),
	}

	entries, diags := expandDictionarySettings(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if len(entries.Stopwords) != 1 || !entries.Stopwords["en"] {
		t.Fatalf("stopwords = %#v, want {en: true}", entries.Stopwords)
	}
	if entries.Plurals != nil {
		t.Fatalf("plurals = %#v, want nil", entries.Plurals)
	}
	if entries.Compounds != nil {
		t.Fatalf("compounds = %#v, want nil", entries.Compounds)
	}
}

func TestExpandDictionarySettings_AllThreeTypes(t *testing.T) {
	model := DictionarySettingsResourceModel{
		DisableStandardEntries: disableStandardEntriesObject(t, DisableStandardEntriesModel{
			Stopwords: boolMapValue(t, map[string]bool{"en": true, "fr": false}),
			Plurals:   boolMapValue(t, map[string]bool{"de": true}),
			Compounds: boolMapValue(t, map[string]bool{"nl": true}),
		}),
	}

	entries, diags := expandDictionarySettings(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if len(entries.Stopwords) != 2 || !entries.Stopwords["en"] || entries.Stopwords["fr"] {
		t.Fatalf("stopwords = %#v, want {en: true, fr: false}", entries.Stopwords)
	}
	if len(entries.Plurals) != 1 || !entries.Plurals["de"] {
		t.Fatalf("plurals = %#v, want {de: true}", entries.Plurals)
	}
	if len(entries.Compounds) != 1 || !entries.Compounds["nl"] {
		t.Fatalf("compounds = %#v, want {nl: true}", entries.Compounds)
	}
}

func TestFlattenDictionarySettings_Empty(t *testing.T) {
	var model DictionarySettingsResourceModel

	diags := flattenDictionarySettings(context.Background(), search.StandardEntries{}, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.DisableStandardEntries.IsNull() {
		t.Fatal("expected a non-null disable_standard_entries object; dictionary settings always exist")
	}

	attrs := model.DisableStandardEntries.Attributes()
	for _, key := range []string{"stopwords", "plurals", "compounds"} {
		m, ok := attrs[key].(types.Map)
		if !ok || m.IsNull() {
			t.Fatalf("%s: want a non-null empty map, got %#v", key, attrs[key])
		}
		if len(m.Elements()) != 0 {
			t.Fatalf("%s: want an empty map, got %#v", key, m.Elements())
		}
	}
}

func TestFlattenDictionarySettings_Partial(t *testing.T) {
	entries := search.StandardEntries{
		Stopwords: map[string]bool{"en": true},
	}

	var model DictionarySettingsResourceModel
	diags := flattenDictionarySettings(context.Background(), entries, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	attrs := model.DisableStandardEntries.Attributes()

	stopwords := attrs["stopwords"].(types.Map)
	boolValue, ok := stopwords.Elements()["en"].(types.Bool)
	if !ok || !boolValue.ValueBool() {
		t.Fatalf("stopwords[en] = %#v, want true", stopwords.Elements()["en"])
	}

	plurals := attrs["plurals"].(types.Map)
	if len(plurals.Elements()) != 0 {
		t.Fatalf("plurals = %#v, want empty", plurals.Elements())
	}

	compounds := attrs["compounds"].(types.Map)
	if len(compounds.Elements()) != 0 {
		t.Fatalf("compounds = %#v, want empty", compounds.Elements())
	}
}

func TestFlattenDictionarySettings_AllThreeTypes(t *testing.T) {
	entries := search.StandardEntries{
		Stopwords: map[string]bool{"en": true},
		Plurals:   map[string]bool{"fr": true, "de": false},
		Compounds: map[string]bool{"nl": true},
	}

	var model DictionarySettingsResourceModel
	diags := flattenDictionarySettings(context.Background(), entries, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	attrs := model.DisableStandardEntries.Attributes()

	stopwords := attrs["stopwords"].(types.Map)
	if v, ok := stopwords.Elements()["en"].(types.Bool); !ok || !v.ValueBool() {
		t.Fatalf("stopwords[en] = %#v, want true", stopwords.Elements()["en"])
	}

	plurals := attrs["plurals"].(types.Map)
	if len(plurals.Elements()) != 2 {
		t.Fatalf("plurals = %#v, want 2 entries", plurals.Elements())
	}
	if v, ok := plurals.Elements()["de"].(types.Bool); !ok || v.ValueBool() {
		t.Fatalf("plurals[de] = %#v, want false", plurals.Elements()["de"])
	}

	compounds := attrs["compounds"].(types.Map)
	if v, ok := compounds.Elements()["nl"].(types.Bool); !ok || !v.ValueBool() {
		t.Fatalf("compounds[nl] = %#v, want true", compounds.Elements()["nl"])
	}
}

func TestExpandFlattenDictionarySettings_RoundTrip(t *testing.T) {
	entries := search.StandardEntries{
		Stopwords: map[string]bool{"en": true},
		Plurals:   map[string]bool{"fr": true},
		Compounds: map[string]bool{"de": true},
	}

	var model DictionarySettingsResourceModel
	if diags := flattenDictionarySettings(context.Background(), entries, &model); diags.HasError() {
		t.Fatalf("unexpected diagnostics flattening: %v", diags)
	}

	roundTripped, diags := expandDictionarySettings(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics expanding: %v", diags)
	}

	if len(roundTripped.Stopwords) != 1 || !roundTripped.Stopwords["en"] {
		t.Fatalf("stopwords = %#v, want {en: true}", roundTripped.Stopwords)
	}
	if len(roundTripped.Plurals) != 1 || !roundTripped.Plurals["fr"] {
		t.Fatalf("plurals = %#v, want {fr: true}", roundTripped.Plurals)
	}
	if len(roundTripped.Compounds) != 1 || !roundTripped.Compounds["de"] {
		t.Fatalf("compounds = %#v, want {de: true}", roundTripped.Compounds)
	}
}

func TestDictionarySettingsSchemas_RegisterExpectedAttributes(t *testing.T) {
	resourceSchema := dictionarySettingsResourceSchema()

	idAttr, ok := resourceSchema.Attributes["id"].(resourceschema.StringAttribute)
	if !ok || !idAttr.Computed {
		t.Fatal("expected id to be a computed string attribute")
	}

	disableAttr, ok := resourceSchema.Attributes["disable_standard_entries"].(resourceschema.SingleNestedAttribute)
	if !ok || !disableAttr.Optional || !disableAttr.Computed {
		t.Fatal("expected disable_standard_entries to be an optional+computed single nested attribute")
	}

	for _, key := range []string{"stopwords", "plurals", "compounds"} {
		mapAttr, ok := disableAttr.Attributes[key].(resourceschema.MapAttribute)
		if !ok || mapAttr.ElementType != types.BoolType {
			t.Fatalf("expected %s to be a map(bool) attribute", key)
		}
	}

	dataSourceSchema := dictionarySettingsDataSourceSchema()
	dsIDAttr, ok := dataSourceSchema.Attributes["id"].(datasourceschema.StringAttribute)
	if !ok || !dsIDAttr.Computed {
		t.Fatal("expected data source id to be computed")
	}

	dsDisableAttr, ok := dataSourceSchema.Attributes["disable_standard_entries"].(datasourceschema.SingleNestedAttribute)
	if !ok || !dsDisableAttr.Computed {
		t.Fatal("expected data source disable_standard_entries to be a computed single nested attribute")
	}
}
