package ingestion

import (
	"strings"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func sourceResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages an Algolia Ingestion source resource: where a task reads records from (or " +
			"receives records pushed to it) before writing them to a destination index. The Ingestion API is " +
			"region-routed, so the provider's `analytics_region` (or the `ALGOLIA_ANALYTICS_REGION` environment " +
			"variable) must be configured.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform identifier for the resource. Equal to `source_id`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_id": schema.StringAttribute{
				Description: "Universally unique identifier (UUID) of the source.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Type of source. Determines the configuration expected in `input`. One of: " +
					strings.Join(allowedSourceTypeStrings(), ", ") + ". Changing this forces replacement: the " +
					"Ingestion API's source update endpoint has no way to change a source's type after creation.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedSourceTypeStrings()...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Descriptive name for the source.",
				Required:    true,
			},
			"input": schema.StringAttribute{
				Description: "JSON-encoded configuration matching `type` (e.g. `jsonencode({ url = \"...\" })` " +
					"for type \"csv\"). Not every source type requires input - a \"push\" source, for example, " +
					"accepts records pushed directly to it and has no `input` shape, so `input` may be omitted. " +
					"Unlike `algolia_ingestion_authentication`'s `input`, the Ingestion API returns a source's " +
					"`input` in full when reading it back (nothing is redacted), so this attribute is refreshed on " +
					"read. To avoid a perpetual diff caused by harmless JSON differences (key order, array " +
					"order), the refresh only replaces the configured value when it is not semantically " +
					"equivalent to what the API returned. Treated as sensitive: several source types carry " +
					"credentials here - a `docker` source's `configuration` is an arbitrary map holding the " +
					"connector's secrets, and `csv`/`json` take a `url` that is commonly presigned - and since " +
					"the API returns `input` unredacted, whatever it contains is persisted in plaintext in " +
					"Terraform state.",
				Optional:  true,
				Sensitive: true,
			},
			"authentication_id": schema.StringAttribute{
				Description: "Universally unique identifier (UUID) of the `algolia_ingestion_authentication` " +
					"resource this source uses to connect to its underlying platform, if any.",
				Optional: true,
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

// allowedSourceTypeStrings derives the list of valid `type` values from the
// Go client's SourceType enum rather than hard-coding it, so a new source
// type added upstream doesn't require a provider code change to become
// selectable (only a client bump).
func allowedSourceTypeStrings() []string {
	values := make([]string, 0, len(ingestionapi.AllowedSourceTypeEnumValues))
	for _, v := range ingestionapi.AllowedSourceTypeEnumValues {
		values = append(values, string(v))
	}

	return values
}
