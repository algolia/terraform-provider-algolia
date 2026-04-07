package synonym

import "github.com/hashicorp/terraform-plugin-framework/types"

type SynonymResourceModel struct {
	ID           types.String `tfsdk:"id"`
	IndexName    types.String `tfsdk:"index_name"`
	ObjectID     types.String `tfsdk:"object_id"`
	Type         types.String `tfsdk:"type"`
	Synonyms     types.Set    `tfsdk:"synonyms"`
	Input        types.String `tfsdk:"input"`
	Word         types.String `tfsdk:"word"`
	Corrections  types.Set    `tfsdk:"corrections"`
	Placeholder  types.String `tfsdk:"placeholder"`
	Replacements types.Set    `tfsdk:"replacements"`
}

