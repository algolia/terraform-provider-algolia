package dictionary

import (
	"context"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// expandDictionarySettings converts the Terraform resource/data source model
// into the Algolia StandardEntries used to disable built-in dictionary
// entries. A null/unknown disable_standard_entries block expands to an empty
// StandardEntries (nothing disabled).
func expandDictionarySettings(ctx context.Context, model *DictionarySettingsResourceModel) (search.StandardEntries, diag.Diagnostics) {
	var diags diag.Diagnostics
	entries := search.StandardEntries{}

	if model.DisableStandardEntries.IsNull() || model.DisableStandardEntries.IsUnknown() {
		return entries, diags
	}

	var block DisableStandardEntriesModel
	diags.Append(model.DisableStandardEntries.As(ctx, &block, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return entries, diags
	}

	entries.Stopwords = expandLanguageBoolMap(block.Stopwords)
	entries.Plurals = expandLanguageBoolMap(block.Plurals)
	entries.Compounds = expandLanguageBoolMap(block.Compounds)

	return entries, diags
}

// expandLanguageBoolMap converts a Terraform map(bool) keyed by language ISO
// code into the map[string]bool the Algolia API expects. Returns nil for a
// null/unknown map so the resulting field is omitted from the request.
func expandLanguageBoolMap(m types.Map) map[string]bool {
	if m.IsNull() || m.IsUnknown() {
		return nil
	}

	result := make(map[string]bool, len(m.Elements()))
	for lang, value := range m.Elements() {
		boolValue, ok := value.(types.Bool)
		if !ok || boolValue.IsNull() || boolValue.IsUnknown() {
			continue
		}
		result[lang] = boolValue.ValueBool()
	}

	return result
}
