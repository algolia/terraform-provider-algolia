package dictionary

import (
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
)

func TestFlattenDictionaryEntryStopwords(t *testing.T) {
	language := search.SUPPORTED_LANGUAGE_EN
	state := search.DICTIONARY_ENTRY_STATE_DISABLED
	entry := search.NewDictionaryEntry(
		"obj-1",
		search.WithDictionaryEntryLanguage(language),
		search.WithDictionaryEntryWord("the"),
		search.WithDictionaryEntryState(state),
	)

	var model DictionaryEntryResourceModel
	diags := flattenDictionaryEntry(search.DICTIONARY_TYPE_STOPWORDS, entry, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.ID.ValueString(); got != "stopwords/obj-1" {
		t.Fatalf("id = %q, want stopwords/obj-1", got)
	}
	if got := model.Dictionary.ValueString(); got != "stopwords" {
		t.Fatalf("dictionary = %q, want stopwords", got)
	}
	if got := model.Language.ValueString(); got != "en" {
		t.Fatalf("language = %q, want en", got)
	}
	if got := model.Word.ValueString(); got != "the" {
		t.Fatalf("word = %q, want the", got)
	}
	if got := model.State.ValueString(); got != "disabled" {
		t.Fatalf("state = %q, want disabled", got)
	}
	if !model.Words.IsNull() {
		t.Fatalf("words = %v, want null", model.Words)
	}
	if !model.Decomposition.IsNull() {
		t.Fatalf("decomposition = %v, want null", model.Decomposition)
	}
}

func TestFlattenDictionaryEntryStopwordsDefaultState(t *testing.T) {
	language := search.SUPPORTED_LANGUAGE_EN
	entry := search.NewDictionaryEntry(
		"obj-1",
		search.WithDictionaryEntryLanguage(language),
		search.WithDictionaryEntryWord("the"),
	)

	var model DictionaryEntryResourceModel
	diags := flattenDictionaryEntry(search.DICTIONARY_TYPE_STOPWORDS, entry, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.State.ValueString(); got != "enabled" {
		t.Fatalf("state = %q, want default enabled", got)
	}
}

func TestFlattenDictionaryEntryPlurals(t *testing.T) {
	language := search.SUPPORTED_LANGUAGE_FR
	entry := search.NewDictionaryEntry(
		"obj-2",
		search.WithDictionaryEntryLanguage(language),
		search.WithDictionaryEntryWords([]string{"cheval", "chevaux"}),
	)

	var model DictionaryEntryResourceModel
	diags := flattenDictionaryEntry(search.DICTIONARY_TYPE_PLURALS, entry, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.Dictionary.ValueString(); got != "plurals" {
		t.Fatalf("dictionary = %q, want plurals", got)
	}
	elements := model.Words.Elements()
	if len(elements) != 2 {
		t.Fatalf("words = %#v, want 2 values", elements)
	}
	if !model.Word.IsNull() {
		t.Fatalf("word = %v, want null", model.Word)
	}
}

func TestFlattenDictionaryEntryCompounds(t *testing.T) {
	language := search.SUPPORTED_LANGUAGE_DE
	entry := search.NewDictionaryEntry(
		"obj-3",
		search.WithDictionaryEntryLanguage(language),
		search.WithDictionaryEntryWord("kopfschmerzen"),
		search.WithDictionaryEntryDecomposition([]string{"kopf", "schmerzen"}),
	)

	var model DictionaryEntryResourceModel
	diags := flattenDictionaryEntry(search.DICTIONARY_TYPE_COMPOUNDS, entry, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.Word.ValueString(); got != "kopfschmerzen" {
		t.Fatalf("word = %q, want kopfschmerzen", got)
	}
	elements := model.Decomposition.Elements()
	if len(elements) != 2 {
		t.Fatalf("decomposition = %#v, want 2 values", elements)
	}
}
