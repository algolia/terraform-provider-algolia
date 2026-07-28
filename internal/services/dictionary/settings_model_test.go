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

	if entries.Stopwords != nil {
		t.Fatalf("expected stopwords nil for a null block, got %#v", entries)
	}
}

func TestExpandDictionarySettings_Empty(t *testing.T) {
	model := DictionarySettingsResourceModel{
		DisableStandardEntries: disableStandardEntriesObject(t, DisableStandardEntriesModel{
			Stopwords: types.MapNull(types.BoolType),
		}),
	}

	entries, diags := expandDictionarySettings(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if entries.Stopwords != nil {
		t.Fatalf("expected stopwords nil when the map is null, got %#v", entries)
	}
}

func TestExpandDictionarySettings_Partial(t *testing.T) {
	model := DictionarySettingsResourceModel{
		DisableStandardEntries: disableStandardEntriesObject(t, DisableStandardEntriesModel{
			Stopwords: boolMapValue(t, map[string]bool{"en": true}),
		}),
	}

	entries, diags := expandDictionarySettings(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if len(entries.Stopwords) != 1 || !entries.Stopwords["en"] {
		t.Fatalf("stopwords = %#v, want {en: true}", entries.Stopwords)
	}
}

func TestExpandDictionarySettings_MultipleLanguages(t *testing.T) {
	model := DictionarySettingsResourceModel{
		DisableStandardEntries: disableStandardEntriesObject(t, DisableStandardEntriesModel{
			Stopwords: boolMapValue(t, map[string]bool{"en": true, "fr": false}),
		}),
	}

	entries, diags := expandDictionarySettings(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if len(entries.Stopwords) != 2 || !entries.Stopwords["en"] || entries.Stopwords["fr"] {
		t.Fatalf("stopwords = %#v, want {en: true, fr: false}", entries.Stopwords)
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
	for _, key := range []string{"stopwords"} {
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
}

func TestFlattenDictionarySettings_MultipleLanguages(t *testing.T) {
	entries := search.StandardEntries{
		Stopwords: map[string]bool{"en": true, "fr": false},
	}

	var model DictionarySettingsResourceModel
	diags := flattenDictionarySettings(context.Background(), entries, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	attrs := model.DisableStandardEntries.Attributes()

	stopwords := attrs["stopwords"].(types.Map)
	if len(stopwords.Elements()) != 2 {
		t.Fatalf("stopwords = %#v, want 2 entries", stopwords.Elements())
	}
	if v, ok := stopwords.Elements()["en"].(types.Bool); !ok || !v.ValueBool() {
		t.Fatalf("stopwords[en] = %#v, want true", stopwords.Elements()["en"])
	}
	if v, ok := stopwords.Elements()["fr"].(types.Bool); !ok || v.ValueBool() {
		t.Fatalf("stopwords[fr] = %#v, want false", stopwords.Elements()["fr"])
	}
}

func TestExpandFlattenDictionarySettings_RoundTrip(t *testing.T) {
	entries := search.StandardEntries{
		Stopwords: map[string]bool{"en": true},
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

	for _, key := range []string{"stopwords"} {
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
