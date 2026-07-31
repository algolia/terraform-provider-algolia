package ingestion

import (
	"strings"

	"github.com/algolia/terraform-provider-algolia/internal/deletionprotection"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
					"returns `code` in full (nothing is redacted), so this attribute is refreshed on read. " +
					"Computed because the API derives it from `input.code` when the logic is supplied that way.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("input")),
					// The API rejects a payload carrying `type` with a top-level
					// `code` and no `input`: "'input' is required if 'Type' is
					// present". Caught at plan time rather than as an opaque 400.
					stringvalidator.ConflictsWith(path.MatchRoot("type")),
				},
				// Deliberately no UseStateForUnknown: the API re-derives `code`
				// whenever `input` changes, so pinning the prior value would
				// report the stale code as the applied result.
			},
			"type": schema.StringAttribute{
				Description: "Type of transformation. One of: " + strings.Join(allowedTransformationTypeStrings(), ", ") +
					". The Ingestion API's transformation update endpoint accepts the same body as create " +
					"(including `type`), so changing this does not force replacement - unlike " +
					"`algolia_ingestion_source`/`algolia_ingestion_destination`, whose update endpoints have no " +
					"`type` field at all.\n\n" +
					"Leave it unset when supplying the logic through the legacy `code` attribute: the API " +
					"derives a type in that case, and this attribute stays null rather than adopting the " +
					"derived value, so that switching between the two forms does not send a type that " +
					"contradicts the logic.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedTransformationTypeStrings()...),
				},
				// Deliberately neither Computed nor UseStateForUnknown. Both were
				// tried: making `type` Computed lets the API's derived value be
				// stored, but UseStateForUnknown is then needed to stop it planning
				// as "known after apply" forever - and that combination replays the
				// prior type on an update. Switching an input-based transformation
				// to `code` while omitting `type` then sent the old type alongside
				// the new code, which the API rejects with "'input' is required if
				// 'Type' is present". The derived value is dropped on read instead;
				// see flattenTransformation.
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
			"deletion_protection": deletionprotection.Attribute("ingestion transformation"),
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
