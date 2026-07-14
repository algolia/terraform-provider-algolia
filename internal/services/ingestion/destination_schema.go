package ingestion

import (
	"strings"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func destinationResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages an Algolia Ingestion destination resource: the Algolia index or Insights event " +
			"stream a task writes records to. The Ingestion API is region-routed, so the provider's " +
			"`analytics_region` (or the `ALGOLIA_ANALYTICS_REGION` environment variable) must be configured.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform identifier for the resource. Equal to `destination_id`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"destination_id": schema.StringAttribute{
				Description: "Universally unique identifier (UUID) of the destination.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Type of destination. One of: " + strings.Join(allowedDestinationTypeStrings(), ", ") +
					". Changing this forces replacement: the Ingestion API's destination update endpoint has no " +
					"way to change a destination's type after creation.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedDestinationTypeStrings()...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Descriptive name for the destination.",
				Required:    true,
			},
			"input": schema.StringAttribute{
				Description: "JSON-encoded configuration matching `type` (e.g. `jsonencode({ indexName = \"...\" " +
					"})`). Unlike `algolia_ingestion_source`'s `input`, a destination's `input` is required: every " +
					"destination writes to a specific `indexName`. The Ingestion API returns a destination's " +
					"`input` in full when reading it back (nothing is redacted), so this attribute is refreshed " +
					"on read. To avoid a perpetual diff caused by harmless JSON differences (key order, array " +
					"order), the refresh only replaces the configured value when it is not semantically " +
					"equivalent to what the API returned.",
				Required: true,
			},
			"authentication_id": schema.StringAttribute{
				Description: "Universally unique identifier (UUID) of the `algolia_ingestion_authentication` " +
					"resource this destination uses to connect to its underlying platform, if any.",
				Optional: true,
			},
			"transformation_ids": schema.ListAttribute{
				Description: "Universally unique identifiers (UUIDs) of the `algolia_ingestion_transformation` " +
					"resources applied to records before they reach this destination, in order.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"created_at": schema.StringAttribute{
				Description: "Date and time when the resource was created, in RFC 3339 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Date and time when the resource was last updated, in RFC 3339 format.",
				Computed:    true,
			},
		},
	}
}

// allowedDestinationTypeStrings derives the list of valid `type` values
// from the Go client's DestinationType enum rather than hard-coding it, so
// a new destination type added upstream doesn't require a provider code
// change to become selectable (only a client bump).
func allowedDestinationTypeStrings() []string {
	values := make([]string, 0, len(ingestionapi.AllowedDestinationTypeEnumValues))
	for _, v := range ingestionapi.AllowedDestinationTypeEnumValues {
		values = append(values, string(v))
	}

	return values
}
