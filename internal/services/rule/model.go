package rule

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type RuleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	IndexName   types.String `tfsdk:"index_name"`
	ObjectID    types.String `tfsdk:"object_id"`
	Description types.String `tfsdk:"description"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Conditions  types.List   `tfsdk:"conditions"`
	Consequence types.List   `tfsdk:"consequence"`
	Validity    types.List   `tfsdk:"validity"`
}

type RuleDataSourceModel = RuleResourceModel

var (
	conditionModelAttrTypes = map[string]attr.Type{
		"pattern":      types.StringType,
		"anchoring":    types.StringType,
		"alternatives": types.BoolType,
		"context":      types.StringType,
		"filters":      types.StringType,
	}
	conditionModelType = types.ObjectType{AttrTypes: conditionModelAttrTypes}

	promoteModelAttrTypes = map[string]attr.Type{
		"object_ids": types.SetType{ElemType: types.StringType},
		"position":   types.Int64Type,
	}
	promoteModelType = types.ObjectType{AttrTypes: promoteModelAttrTypes}

	consequenceModelAttrTypes = map[string]attr.Type{
		"params_json": types.StringType,
		"promote":     types.ListType{ElemType: promoteModelType},
		"hide":        types.SetType{ElemType: types.StringType},
		"user_data":   types.StringType,
	}
	consequenceModelType = types.ObjectType{AttrTypes: consequenceModelAttrTypes}

	validityModelAttrTypes = map[string]attr.Type{
		"from":  types.StringType,
		"until": types.StringType,
	}
	validityModelType = types.ObjectType{AttrTypes: validityModelAttrTypes}
)
