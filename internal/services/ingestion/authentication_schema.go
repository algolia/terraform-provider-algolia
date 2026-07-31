package ingestion

import (
	"strings"

	"github.com/algolia/terraform-provider-algolia/internal/deletionprotection"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func authenticationResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages an Algolia Ingestion authentication resource: reusable credentials that " +
			"sources and destinations use to connect to a platform or to Algolia itself. The Ingestion API is " +
			"region-routed, so the provider's `analytics_region` (or the `ALGOLIA_ANALYTICS_REGION` environment " +
			"variable) must be configured.\n\n" +
			"Import limitation: the Ingestion API redacts secret values when reading an authentication resource " +
			"back, so `terraform import` cannot recover `input`. After importing, set `input` explicitly in " +
			"configuration; the next `terraform apply` pushes it to Algolia.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform identifier for the resource. Equal to `authentication_id`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"authentication_id": schema.StringAttribute{
				Description: "Universally unique identifier (UUID) of the authentication resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Type of authentication. Determines the credentials expected in `input`. One of: " +
					strings.Join(allowedAuthenticationTypeStrings(), ", ") + ". Changing this forces replacement, " +
					"since it changes the shape of the credentials.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedAuthenticationTypeStrings()...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Descriptive name for the authentication resource.",
				Required:    true,
			},
			"platform": schema.StringAttribute{
				Description: "Name of an ecommerce platform to authenticate with, one of: " +
					strings.Join(allowedPlatformStrings(), ", ") + ". Determines which authentication types are " +
					"selectable for that platform. The Ingestion API's update endpoint does not support changing " +
					"platform after creation, so changing this forces replacement.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedPlatformStrings()...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"input": schema.StringAttribute{
				Description: "JSON-encoded credentials matching `type` (e.g. `jsonencode({ appID = ..., apiKey = ... })` " +
					"for type \"algolia\"). Write-only: the Algolia API redacts secret values when reading " +
					"authentication resources back, so Terraform never refreshes this attribute from the API - " +
					"the value you configure is the value that stays in state until you change it in configuration. " +
					"Because it is never refreshed, out-of-band changes to the actual credential are not detected " +
					"as drift.",
				Required:  true,
				Sensitive: true,
			},
			"created_at": schema.StringAttribute{
				Description: "Date and time when the resource was created, in RFC 3339 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Date and time when the resource was last updated, in RFC 3339 format.",
				Computed:    true,
			},
			"deletion_protection": deletionprotection.Attribute("ingestion authentication"),
		},
	}
}

func allowedAuthenticationTypeStrings() []string {
	values := make([]string, 0, len(ingestionapi.AllowedAuthenticationTypeEnumValues))
	for _, v := range ingestionapi.AllowedAuthenticationTypeEnumValues {
		values = append(values, string(v))
	}

	return values
}

func allowedPlatformStrings() []string {
	values := make([]string, 0, len(ingestionapi.AllowedPlatformEnumValues))
	for _, v := range ingestionapi.AllowedPlatformEnumValues {
		values = append(values, string(v))
	}

	return values
}
