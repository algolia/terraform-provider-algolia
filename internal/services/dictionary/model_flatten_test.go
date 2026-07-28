package dictionary

import (
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	if !model.State.IsNull() {
		t.Fatalf("state = %v, want null for plurals (stopwords-only field)", model.State)
	}
}

// TestFlattenStringList covers the null-vs-empty contract for `words` and
// `decomposition`: both are Optional and not Computed, so their planned value
// is the configuration verbatim. Mapping an empty API value to null regardless
// of the prior value aborts the apply of an explicitly configured `words = []`
// with "Provider produced inconsistent result after apply".
func TestFlattenStringList(t *testing.T) {
	emptyList := types.ListValueMust(types.StringType, []attr.Value{})
	configuredList := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("cheval")})

	tests := []struct {
		name   string
		values []string
		prior  types.List
		want   types.List
	}{
		{
			name:   "api empty and prior null stays null",
			values: nil,
			prior:  types.ListNull(types.StringType),
			want:   types.ListNull(types.StringType),
		},
		{
			name:   "api empty and prior known empty stays known empty",
			values: []string{},
			prior:  emptyList,
			want:   emptyList,
		},
		{
			name:   "api non-empty wins",
			values: []string{"cheval"},
			prior:  types.ListNull(types.StringType),
			want:   configuredList,
		},
		{
			name:   "api empty and prior with entries is drift and becomes null",
			values: nil,
			prior:  configuredList,
			want:   types.ListNull(types.StringType),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, diags := flattenStringList(test.values, test.prior)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if !got.Equal(test.want) {
				t.Fatalf("list = %v, want %v", got, test.want)
			}
		})
	}
}

// TestFlattenDictionaryEntryPreservesConfiguredEmptyLists is the end-to-end
// version: the prior value reaches flattenStringList through the model being
// refreshed (the plan on Create/Update, the prior state on Read).
func TestFlattenDictionaryEntryPreservesConfiguredEmptyLists(t *testing.T) {
	language := search.SUPPORTED_LANGUAGE_EN
	entry := search.NewDictionaryEntry(
		"obj-4",
		search.WithDictionaryEntryLanguage(language),
		search.WithDictionaryEntryWord("the"),
	)

	emptyList := types.ListValueMust(types.StringType, []attr.Value{})
	model := DictionaryEntryResourceModel{
		Words:         emptyList,
		Decomposition: emptyList,
	}

	diags := flattenDictionaryEntry(search.DICTIONARY_TYPE_STOPWORDS, entry, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Words.IsNull() || len(model.Words.Elements()) != 0 {
		t.Fatalf("words = %v, want a known empty list (the configured value)", model.Words)
	}
	if model.Decomposition.IsNull() || len(model.Decomposition.Elements()) != 0 {
		t.Fatalf("decomposition = %v, want a known empty list (the configured value)", model.Decomposition)
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
	if !model.State.IsNull() {
		t.Fatalf("state = %v, want null for compounds (stopwords-only field)", model.State)
	}
}
