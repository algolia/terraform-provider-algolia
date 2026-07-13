package dictionary

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stringListValue(values ...string) types.List {
	attrValues := make([]attr.Value, 0, len(values))
	for _, value := range values {
		attrValues = append(attrValues, types.StringValue(value))
	}
	return types.ListValueMust(types.StringType, attrValues)
}

func TestExpandDictionaryEntryStopwords(t *testing.T) {
	model := DictionaryEntryResourceModel{
		Dictionary:    types.StringValue("stopwords"),
		Language:      types.StringValue("en"),
		Word:          types.StringValue("the"),
		Words:         types.ListNull(types.StringType),
		Decomposition: types.ListNull(types.StringType),
		State:         types.StringNull(),
	}

	entry, diags := expandDictionaryEntry("obj-1", &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := entry.GetObjectID(); got != "obj-1" {
		t.Fatalf("object_id = %q, want %q", got, "obj-1")
	}
	if got := entry.GetWord(); got != "the" {
		t.Fatalf("word = %q, want %q", got, "the")
	}
	if entry.Language == nil || string(*entry.Language) != "en" {
		t.Fatalf("language = %v, want en", entry.Language)
	}
	if entry.State == nil || string(*entry.State) != "enabled" {
		t.Fatalf("state = %v, want default enabled", entry.State)
	}
}

func TestExpandDictionaryEntryStopwordsExplicitState(t *testing.T) {
	model := DictionaryEntryResourceModel{
		Dictionary:    types.StringValue("stopwords"),
		Language:      types.StringValue("en"),
		Word:          types.StringValue("the"),
		Words:         types.ListNull(types.StringType),
		Decomposition: types.ListNull(types.StringType),
		State:         types.StringValue("disabled"),
	}

	entry, diags := expandDictionaryEntry("obj-1", &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if entry.State == nil || string(*entry.State) != "disabled" {
		t.Fatalf("state = %v, want disabled", entry.State)
	}
}

func TestExpandDictionaryEntryStopwordsMissingWord(t *testing.T) {
	model := DictionaryEntryResourceModel{
		Dictionary:    types.StringValue("stopwords"),
		Language:      types.StringValue("en"),
		Word:          types.StringNull(),
		Words:         types.ListNull(types.StringType),
		Decomposition: types.ListNull(types.StringType),
		State:         types.StringNull(),
	}

	_, diags := expandDictionaryEntry("obj-1", &model)
	if !diags.HasError() {
		t.Fatalf("expected an error for missing word")
	}
}

func TestExpandDictionaryEntryPlurals(t *testing.T) {
	model := DictionaryEntryResourceModel{
		Dictionary:    types.StringValue("plurals"),
		Language:      types.StringValue("fr"),
		Word:          types.StringNull(),
		Words:         stringListValue("cheval", "chevaux"),
		Decomposition: types.ListNull(types.StringType),
		State:         types.StringNull(),
	}

	entry, diags := expandDictionaryEntry("obj-2", &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := entry.GetWords(); len(got) != 2 || got[0] != "cheval" || got[1] != "chevaux" {
		t.Fatalf("words = %#v, want [cheval chevaux]", got)
	}
	if entry.State != nil {
		t.Fatalf("state = %v, want nil for plurals", entry.State)
	}
}

func TestExpandDictionaryEntryPluralsMissingWords(t *testing.T) {
	model := DictionaryEntryResourceModel{
		Dictionary:    types.StringValue("plurals"),
		Language:      types.StringValue("fr"),
		Word:          types.StringNull(),
		Words:         types.ListNull(types.StringType),
		Decomposition: types.ListNull(types.StringType),
		State:         types.StringNull(),
	}

	_, diags := expandDictionaryEntry("obj-2", &model)
	if !diags.HasError() {
		t.Fatalf("expected an error for missing words")
	}
}

func TestExpandDictionaryEntryCompounds(t *testing.T) {
	model := DictionaryEntryResourceModel{
		Dictionary:    types.StringValue("compounds"),
		Language:      types.StringValue("de"),
		Word:          types.StringValue("kopfschmerzen"),
		Words:         types.ListNull(types.StringType),
		Decomposition: stringListValue("kopf", "schmerzen"),
		State:         types.StringNull(),
	}

	entry, diags := expandDictionaryEntry("obj-3", &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := entry.GetWord(); got != "kopfschmerzen" {
		t.Fatalf("word = %q, want kopfschmerzen", got)
	}
	if got := entry.GetDecomposition(); len(got) != 2 || got[0] != "kopf" || got[1] != "schmerzen" {
		t.Fatalf("decomposition = %#v, want [kopf schmerzen]", got)
	}
}

func TestExpandDictionaryEntryCompoundsMissingDecomposition(t *testing.T) {
	model := DictionaryEntryResourceModel{
		Dictionary:    types.StringValue("compounds"),
		Language:      types.StringValue("de"),
		Word:          types.StringValue("kopfschmerzen"),
		Words:         types.ListNull(types.StringType),
		Decomposition: types.ListNull(types.StringType),
		State:         types.StringNull(),
	}

	_, diags := expandDictionaryEntry("obj-3", &model)
	if !diags.HasError() {
		t.Fatalf("expected an error for missing decomposition")
	}
}

func TestExpandDictionaryEntryMissingLanguage(t *testing.T) {
	model := DictionaryEntryResourceModel{
		Dictionary:    types.StringValue("stopwords"),
		Language:      types.StringNull(),
		Word:          types.StringValue("the"),
		Words:         types.ListNull(types.StringType),
		Decomposition: types.ListNull(types.StringType),
		State:         types.StringNull(),
	}

	_, diags := expandDictionaryEntry("obj-1", &model)
	if !diags.HasError() {
		t.Fatalf("expected an error for missing language")
	}
}

func TestParseDictionaryEntryImportID(t *testing.T) {
	dictionary, objectID, err := parseDictionaryEntryImportID("stopwords/obj-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dictionary != "stopwords" || objectID != "obj-1" {
		t.Fatalf("got (%q, %q), want (stopwords, obj-1)", dictionary, objectID)
	}
}

func TestParseDictionaryEntryImportIDInvalid(t *testing.T) {
	for _, id := range []string{"", "stopwords", "stopwords/", "/obj-1"} {
		if _, _, err := parseDictionaryEntryImportID(id); err == nil {
			t.Fatalf("expected an error for import ID %q", id)
		}
	}
}

func TestGenerateObjectIDIsUniqueUUIDv4(t *testing.T) {
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

	first, err := generateObjectID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := generateObjectID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first == second {
		t.Fatalf("expected generateObjectID to produce unique values, got %q twice", first)
	}
	if !uuidPattern.MatchString(first) {
		t.Fatalf("generated object_id %q does not look like a UUIDv4", first)
	}
	if !uuidPattern.MatchString(second) {
		t.Fatalf("generated object_id %q does not look like a UUIDv4", second)
	}
}
