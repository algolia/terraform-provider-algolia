package dictionary

import (
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// expandDictionaryEntry converts the Terraform resource model into an
// Algolia DictionaryEntry, applying the field requirements for the entry's
// dictionary type ("stopwords", "plurals", or "compounds").
func expandDictionaryEntry(objectID string, model *DictionaryEntryResourceModel) (*search.DictionaryEntry, diag.Diagnostics) {
	var diags diag.Diagnostics

	if model.Language.IsNull() || model.Language.IsUnknown() || model.Language.ValueString() == "" {
		diags.AddError("Missing language", "language is required for dictionary entries.")
		return nil, diags
	}

	dictionaryType := model.Dictionary.ValueString()
	entry := search.NewDictionaryEntry(objectID)
	language := search.SupportedLanguage(model.Language.ValueString())
	entry.Language = &language

	switch search.DictionaryType(dictionaryType) {
	case search.DICTIONARY_TYPE_STOPWORDS:
		if model.Word.IsNull() || model.Word.IsUnknown() || model.Word.ValueString() == "" {
			diags.AddError("Missing word", "dictionary = \"stopwords\" requires word.")
			return nil, diags
		}
		word := model.Word.ValueString()
		entry.Word = &word
	case search.DICTIONARY_TYPE_COMPOUNDS:
		if model.Word.IsNull() || model.Word.IsUnknown() || model.Word.ValueString() == "" {
			diags.AddError("Missing word", "dictionary = \"compounds\" requires word.")
			return nil, diags
		}
		decomposition := expandStringList(model.Decomposition)
		if len(decomposition) == 0 {
			diags.AddError("Missing decomposition", "dictionary = \"compounds\" requires decomposition.")
			return nil, diags
		}
		word := model.Word.ValueString()
		entry.Word = &word
		entry.Decomposition = decomposition
	case search.DICTIONARY_TYPE_PLURALS:
		words := expandStringList(model.Words)
		if len(words) == 0 {
			diags.AddError("Missing words", "dictionary = \"plurals\" requires words.")
			return nil, diags
		}
		entry.Words = words
	default:
		diags.AddError("Unsupported dictionary", "Unknown dictionary type "+dictionaryType)
		return nil, diags
	}

	if !model.State.IsNull() && !model.State.IsUnknown() && model.State.ValueString() != "" {
		state := search.DictionaryEntryState(model.State.ValueString())
		entry.State = &state
	} else if search.DictionaryType(dictionaryType) == search.DICTIONARY_TYPE_STOPWORDS {
		defaultState := search.DICTIONARY_ENTRY_STATE_ENABLED
		entry.State = &defaultState
	}

	return entry, diags
}

func expandStringList(list types.List) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	values := make([]string, 0, len(list.Elements()))
	for _, element := range list.Elements() {
		if value, ok := element.(types.String); ok && !value.IsNull() && !value.IsUnknown() {
			values = append(values, value.ValueString())
		}
	}

	return values
}

func parseDictionaryEntryImportID(id string) (string, string, error) {
	index := strings.Index(id, "/")
	if index <= 0 || index == len(id)-1 {
		return "", "", fmt.Errorf("expected import ID in the form <dictionary>/<object_id>")
	}

	return id[:index], id[index+1:], nil
}

func dictionaryEntryResourceID(dictionary, objectID string) string {
	return dictionary + "/" + objectID
}

// generateObjectID returns a randomly generated RFC 4122 version 4 UUID,
// used as the object_id when the user does not supply one.
func generateObjectID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("could not generate random object_id: %w", err)
	}

	buf[6] = (buf[6] & 0x0f) | 0x40 // version 4
	buf[8] = (buf[8] & 0x3f) | 0x80 // variant 10

	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16]), nil
}
