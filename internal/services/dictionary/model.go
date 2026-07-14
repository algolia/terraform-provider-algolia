package dictionary

import "github.com/hashicorp/terraform-plugin-framework/types"

type DictionaryEntryResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Dictionary    types.String `tfsdk:"dictionary"`
	ObjectID      types.String `tfsdk:"object_id"`
	Language      types.String `tfsdk:"language"`
	Word          types.String `tfsdk:"word"`
	Words         types.List   `tfsdk:"words"`
	Decomposition types.List   `tfsdk:"decomposition"`
	State         types.String `tfsdk:"state"`
}

type DictionaryEntryDataSourceModel = DictionaryEntryResourceModel
