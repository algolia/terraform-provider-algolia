package dictionary

import (
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// flattenDictionaryEntry converts an Algolia DictionaryEntry into the
// Terraform resource/data source model.
func flattenDictionaryEntry(dictionaryType search.DictionaryType, entry *search.DictionaryEntry, model *DictionaryEntryResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(dictionaryEntryResourceID(string(dictionaryType), entry.GetObjectID()))
	model.Dictionary = types.StringValue(string(dictionaryType))
	model.ObjectID = types.StringValue(entry.GetObjectID())

	if entry.Language != nil {
		model.Language = types.StringValue(string(*entry.Language))
	} else {
		model.Language = types.StringNull()
	}

	model.Word = nullableString(entry.Word)

	words, wordsDiags := flattenStringList(entry.GetWords())
	diags.Append(wordsDiags...)
	model.Words = words

	decomposition, decompositionDiags := flattenStringList(entry.GetDecomposition())
	diags.Append(decompositionDiags...)
	model.Decomposition = decomposition

	// state is a stopwords-only concept. Only default it to "enabled" for
	// stopwords; for plurals/compounds leave it null so the provider does not
	// persist (and later re-send on Update via expand) a field the API does
	// not use for those dictionaries.
	switch {
	case entry.State != nil:
		model.State = types.StringValue(string(*entry.State))
	case dictionaryType == search.DICTIONARY_TYPE_STOPWORDS:
		model.State = types.StringValue(string(search.DICTIONARY_ENTRY_STATE_ENABLED))
	default:
		model.State = types.StringNull()
	}

	return diags
}

func flattenStringList(values []string) (types.List, diag.Diagnostics) {
	if len(values) == 0 {
		return types.ListNull(types.StringType), nil
	}

	attrValues := make([]attr.Value, 0, len(values))
	for _, value := range values {
		attrValues = append(attrValues, types.StringValue(value))
	}

	return types.ListValue(types.StringType, attrValues)
}

func nullableString(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}
