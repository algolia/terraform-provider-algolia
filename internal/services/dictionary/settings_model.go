package dictionary

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DictionarySettingsResourceModel describes the algolia_dictionary_settings
// resource/data source model. This is an application-level singleton: there
// is exactly one dictionary settings configuration per Algolia application.
type DictionarySettingsResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	DisableStandardEntries types.Object `tfsdk:"disable_standard_entries"`
}

type DictionarySettingsDataSourceModel = DictionarySettingsResourceModel

// DisableStandardEntriesModel describes the disable_standard_entries nested
// attribute: for each dictionary type, a map of language ISO code to whether
// Algolia's built-in standard entries for that language are disabled.
type DisableStandardEntriesModel struct {
	Stopwords types.Map `tfsdk:"stopwords"`
	Plurals   types.Map `tfsdk:"plurals"`
	Compounds types.Map `tfsdk:"compounds"`
}

// disableStandardEntriesAttrTypes mirrors the disable_standard_entries
// schema exactly; used to convert to/from types.Object.
var disableStandardEntriesAttrTypes = map[string]attr.Type{
	"stopwords": types.MapType{ElemType: types.BoolType},
	"plurals":   types.MapType{ElemType: types.BoolType},
	"compounds": types.MapType{ElemType: types.BoolType},
}
