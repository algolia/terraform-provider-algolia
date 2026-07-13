package collection

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// CollectionResourceModel describes the Terraform resource data model for algolia_collection.
type CollectionResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	IndexName          types.String `tfsdk:"index_name"`
	Description        types.String `tfsdk:"description"`
	Records            types.List   `tfsdk:"records"`
	Commit             types.Bool   `tfsdk:"commit"`
	DeletionProtection types.Bool   `tfsdk:"deletion_protection"`

	Conditions types.Object `tfsdk:"conditions"`

	// Computed
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

// CollectionDataSourceModel describes the Terraform data source model for algolia_collection.
type CollectionDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	IndexName   types.String `tfsdk:"index_name"`
	Description types.String `tfsdk:"description"`
	Records     types.List   `tfsdk:"records"`

	Conditions types.Object `tfsdk:"conditions"`

	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

// ConditionsModel represents the conditions block. Each field is a list of
// FilterGroupModel values; sibling groups are AND-ed on the wire, and items
// inside each group's Filters list are OR-ed.
type ConditionsModel struct {
	FacetFilter   types.List `tfsdk:"facet_filter"`
	NumericFilter types.List `tfsdk:"numeric_filter"`
}

// FilterGroupModel is one AND clause of OR-ed filters.
type FilterGroupModel struct {
	Filters types.List `tfsdk:"filters"`
}
