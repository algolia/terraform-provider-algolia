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

func transformationResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages an Algolia Ingestion transformation resource: code or no-code logic applied to " +
			"records as they flow from a source to a destination. The Ingestion API is region-routed, so the " +
			"provider's `analytics_region` (or the `ALGOLIA_ANALYTICS_REGION` environment variable) must be " +
			"configured.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform identifier for the resource. Equal to `transformation_id`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"transformation_id": schema.StringAttribute{
				Description: "Universally unique identifier (UUID) of the transformation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Descriptive, uniquely identified name for the transformation.",
				Required:    true,
			},
			"code": schema.StringAttribute{
				Description: "The transformation's source code (for `type = \"code\"` transformations). This is " +
					"the deprecated, legacy way of specifying a code transformation's logic directly - the " +
					"Ingestion API recommends `input` with a matching `type` instead. Leave it unset for no-code " +
					"transformations (which have no `code`); an unset `code` reads back as null. The Ingestion API " +
					"returns `code` in full (nothing is redacted), so this attribute is refreshed on read.",
				Optional: true,
			},
			"type": schema.StringAttribute{
				Description: "Type of transformation. One of: " + strings.Join(allowedTransformationTypeStrings(), ", ") +
					". The Ingestion API's transformation update endpoint accepts the same body as create " +
					"(including `type`), so changing this does not force replacement - unlike " +
					"`algolia_ingestion_source`/`algolia_ingestion_destination`, whose update endpoints have no " +
					"`type` field at all.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedTransformationTypeStrings()...),
				},
			},
			"input": schema.StringAttribute{
				Description: "JSON-encoded configuration matching `type` (e.g. `jsonencode({ steps = [...] })` " +
					"for a no-code transformation, or `jsonencode({ code = \"...\" })` for a code transformation). " +
					"Optional: a transformation's logic can instead be supplied via the legacy `code` attribute. " +
					"The Ingestion API returns a transformation's `input` in full when reading it back (nothing is " +
					"redacted), so this attribute is refreshed on read. To avoid a perpetual diff caused by " +
					"harmless JSON differences (object key order, and the order of arrays of scalars), the refresh " +
					"only replaces the configured value when it is not semantically equivalent to what the API " +
					"returned. Note the order of arrays of objects (e.g. `steps`) is significant and preserved.",
				Optional: true,
			},
			"description": schema.StringAttribute{
				Description: "A descriptive name for the transformation explaining what it does.",
				Optional:    true,
			},
			"authentication_ids": schema.ListAttribute{
				Description: "Universally unique identifiers (UUIDs) of the `algolia_ingestion_authentication` " +
					"resources associated with this transformation.",
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

// allowedTransformationTypeStrings derives the list of valid `type` values
// from the Go client's TransformationType enum rather than hard-coding it,
// so a new transformation type added upstream doesn't require a provider
// code change to become selectable (only a client bump).
func allowedTransformationTypeStrings() []string {
	values := make([]string, 0, len(ingestionapi.AllowedTransformationTypeEnumValues))
	for _, v := range ingestionapi.AllowedTransformationTypeEnumValues {
		values = append(values, string(v))
	}

	return values
}
