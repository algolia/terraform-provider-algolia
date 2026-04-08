package personalization

import (
	api "github.com/algolia/algoliasearch-client-go/v4/algolia/personalization"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func personalizationEventTypeValues() []string {
	values := make([]string, 0, len(api.AllowedEventTypeEnumValues))
	for _, value := range api.AllowedEventTypeEnumValues {
		values = append(values, string(value))
	}
	return values
}

func personalizationStrategyResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages the Algolia app-level Personalization strategy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform identifier for the singleton personalization strategy resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"personalization_impact": schema.Int64Attribute{
				Description: "Impact of personalization on search results.",
				Required:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"events_scoring": schema.ListNestedBlock{
				Description: "Scoring rules for tracked events.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"event_name": schema.StringAttribute{Required: true},
						"event_type": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.OneOf(personalizationEventTypeValues()...),
							},
						},
						"score": schema.Int64Attribute{Required: true},
					},
				},
			},
			"facets_scoring": schema.ListNestedBlock{
				Description: "Scoring rules for facet attributes.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"facet_name": schema.StringAttribute{Required: true},
						"score":      schema.Int64Attribute{Required: true},
					},
				},
			},
		},
	}
}

func personalizationStrategyDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read the Algolia app-level Personalization strategy.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Description: "Terraform identifier for the singleton personalization strategy resource.",
				Computed:    true,
			},
			"personalization_impact": datasourceschema.Int64Attribute{
				Description: "Impact of personalization on search results.",
				Computed:    true,
			},
		},
		Blocks: map[string]datasourceschema.Block{
			"events_scoring": datasourceschema.ListNestedBlock{
				Description: "Scoring rules for tracked events.",
				NestedObject: datasourceschema.NestedBlockObject{
					Attributes: map[string]datasourceschema.Attribute{
						"event_name": datasourceschema.StringAttribute{Computed: true},
						"event_type": datasourceschema.StringAttribute{Computed: true},
						"score":      datasourceschema.Int64Attribute{Computed: true},
					},
				},
			},
			"facets_scoring": datasourceschema.ListNestedBlock{
				Description: "Scoring rules for facet attributes.",
				NestedObject: datasourceschema.NestedBlockObject{
					Attributes: map[string]datasourceschema.Attribute{
						"facet_name": datasourceschema.StringAttribute{Computed: true},
						"score":      datasourceschema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}
