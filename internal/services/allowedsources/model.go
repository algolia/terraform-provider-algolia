package allowedsources

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AllowedSourcesResourceModel describes the algolia_allowed_sources
// resource/data source model. This is an application-level singleton: it
// manages the complete allowlist of source IP addresses/ranges permitted to
// use the Algolia API for the application (the "Sources" security setting).
type AllowedSourcesResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Source types.Set    `tfsdk:"source"`
}

type AllowedSourcesDataSourceModel = AllowedSourcesResourceModel

// SourceModel describes a single entry in the source set: an allowed IP
// address or CIDR range, with an optional human-readable description.
type SourceModel struct {
	Source      types.String `tfsdk:"source"`
	Description types.String `tfsdk:"description"`
}

// sourceAttrTypes mirrors the source nested object schema exactly; used to
// convert to/from types.Set.
var sourceAttrTypes = map[string]attr.Type{
	"source":      types.StringType,
	"description": types.StringType,
}
