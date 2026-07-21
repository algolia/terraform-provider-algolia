package dictionary

import (
	"context"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// flattenDictionarySettings converts Algolia StandardEntries into the
// Terraform resource/data source model. Dictionary settings are an
// application-level singleton that always exists, so this always produces a
// non-null disable_standard_entries object with (possibly empty) maps,
// rather than ever leaving the field null.
func flattenDictionarySettings(ctx context.Context, entries search.StandardEntries, model *DictionarySettingsResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	block := DisableStandardEntriesModel{
		Stopwords: flattenLanguageBoolMap(entries.Stopwords),
	}

	objVal, d := types.ObjectValueFrom(ctx, disableStandardEntriesAttrTypes, &block)
	diags.Append(d...)
	if !diags.HasError() {
		model.DisableStandardEntries = objVal
	}

	return diags
}

// flattenLanguageBoolMap converts an Algolia language->bool map into a
// Terraform map(bool). A nil/empty input still yields a non-null, empty map.
func flattenLanguageBoolMap(m map[string]bool) types.Map {
	values := make(map[string]attr.Value, len(m))
	for lang, disabled := range m {
		values[lang] = types.BoolValue(disabled)
	}

	return types.MapValueMust(types.BoolType, values)
}
