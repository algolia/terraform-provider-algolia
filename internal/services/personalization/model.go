package personalization

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const personalizationStrategyID = "default"

type PersonalizationStrategyResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	PersonalizationImpact types.Int64  `tfsdk:"personalization_impact"`
	EventsScoring         types.List   `tfsdk:"events_scoring"`
	FacetsScoring         types.List   `tfsdk:"facets_scoring"`
}

type PersonalizationStrategyDataSourceModel = PersonalizationStrategyResourceModel

var (
	eventsScoringModelAttrTypes = map[string]attr.Type{
		"event_name": types.StringType,
		"event_type": types.StringType,
		"score":      types.Int64Type,
	}
	eventsScoringModelType = types.ObjectType{AttrTypes: eventsScoringModelAttrTypes}

	facetsScoringModelAttrTypes = map[string]attr.Type{
		"facet_name": types.StringType,
		"score":      types.Int64Type,
	}
	facetsScoringModelType = types.ObjectType{AttrTypes: facetsScoringModelAttrTypes}
)
