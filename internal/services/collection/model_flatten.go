package collection

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var filterGroupAttrTypes = map[string]attr.Type{
	"filters": types.ListType{ElemType: types.StringType},
}

var filterGroupObjectType = types.ObjectType{AttrTypes: filterGroupAttrTypes}

var conditionsAttrTypes = map[string]attr.Type{
	"facet_filter":   types.ListType{ElemType: filterGroupObjectType},
	"numeric_filter": types.ListType{ElemType: filterGroupObjectType},
}

// flattenCollectionResponse populates the resource model's common fields from
// an API response. Resource-only flags (Commit, DeletionProtection) are left
// untouched so callers can preserve them from prior state.
func flattenCollectionResponse(ctx context.Context, resp *CollectionResponse, model *CollectionResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(resp.ID)
	model.Name = types.StringValue(resp.Name)
	model.IndexName = types.StringValue(resp.IndexName)
	model.Description = flattenNullableString(resp.Description)
	model.CreatedAt = types.StringValue(resp.CreatedAt)
	model.UpdatedAt = flattenNullableString(resp.UpdatedAt)
	model.Status = flattenNullableString(resp.Status)

	model.Records = flattenStringList(ctx, resp.RecordIDs())

	condObj, d := flattenConditions(ctx, resp.Conditions)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	model.Conditions = condObj

	return diags
}

// flattenCollectionResponseForDataSource fills the data source model.
func flattenCollectionResponseForDataSource(ctx context.Context, resp *CollectionResponse, model *CollectionDataSourceModel) diag.Diagnostics {
	resource := &CollectionResourceModel{}
	diags := flattenCollectionResponse(ctx, resp, resource)
	if diags.HasError() {
		return diags
	}

	model.ID = resource.ID
	model.Name = resource.Name
	model.IndexName = resource.IndexName
	model.Description = resource.Description
	model.Records = resource.Records
	model.Conditions = resource.Conditions
	model.Status = resource.Status
	model.CreatedAt = resource.CreatedAt
	model.UpdatedAt = resource.UpdatedAt

	return diags
}

func flattenConditions(ctx context.Context, c *Conditions) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if c == nil {
		return types.ObjectNull(conditionsAttrTypes), diags
	}

	facet, d := flattenFilterGroups(ctx, c.FacetFilters)
	diags.Append(d...)
	if diags.HasError() {
		return types.ObjectNull(conditionsAttrTypes), diags
	}
	numeric, d := flattenFilterGroups(ctx, c.NumericFilters)
	diags.Append(d...)
	if diags.HasError() {
		return types.ObjectNull(conditionsAttrTypes), diags
	}

	if facet.IsNull() && numeric.IsNull() {
		return types.ObjectNull(conditionsAttrTypes), diags
	}

	obj, d := types.ObjectValueFrom(ctx, conditionsAttrTypes, &ConditionsModel{
		FacetFilter:   facet,
		NumericFilter: numeric,
	})
	diags.Append(d...)
	return obj, diags
}

// flattenFilterGroups converts the API's wire shape back into a list of
// FilterGroupModel. Strings become single-filter groups; arrays become
// OR-groups. Anything else is dropped defensively with a warning log so
// future server-side schema changes don't panic the provider.
func flattenFilterGroups(ctx context.Context, items any) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	raw, ok := items.([]any)
	if !ok || len(raw) == 0 {
		return types.ListNull(filterGroupObjectType), diags
	}

	groups := make([]FilterGroupModel, 0, len(raw))
	for _, elem := range raw {
		switch v := elem.(type) {
		case string:
			list, d := types.ListValueFrom(ctx, types.StringType, []string{v})
			diags.Append(d...)
			if diags.HasError() {
				return types.ListNull(filterGroupObjectType), diags
			}
			groups = append(groups, FilterGroupModel{Filters: list})
		case []any:
			strs := make([]string, 0, len(v))
			for _, inner := range v {
				s, ok := inner.(string)
				if !ok {
					tflog.Warn(ctx, "Dropping non-string filter element", map[string]interface{}{"element": inner})
					continue
				}
				strs = append(strs, s)
			}
			if len(strs) == 0 {
				continue
			}
			list, d := types.ListValueFrom(ctx, types.StringType, strs)
			diags.Append(d...)
			if diags.HasError() {
				return types.ListNull(filterGroupObjectType), diags
			}
			groups = append(groups, FilterGroupModel{Filters: list})
		default:
			tflog.Warn(ctx, "Dropping unrecognized filter shape", map[string]interface{}{"element": elem})
		}
	}

	if len(groups) == 0 {
		return types.ListNull(filterGroupObjectType), diags
	}

	list, d := types.ListValueFrom(ctx, filterGroupObjectType, groups)
	diags.Append(d...)
	return list, diags
}

func flattenNullableString(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

// flattenStringList converts a Go []string to a Terraform types.List of strings.
// A nil or empty slice becomes a null list so unset records don't show spurious drift.
func flattenStringList(ctx context.Context, values []string) types.List {
	if len(values) == 0 {
		return types.ListNull(types.StringType)
	}
	list, _ := types.ListValueFrom(ctx, types.StringType, values)
	return list
}
