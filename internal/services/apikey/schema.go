package apikey

import (
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	schemavalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func apiKeyResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages an Algolia API key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The actual API key value.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"acl": schema.SetAttribute{
				Description: "Set of permissions associated with the API key.",
				Required:    true,
				ElementType: types.StringType,
				Validators: []schemavalidator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.OneOf(allowedACLStrings()...)),
				},
			},
			"description": schema.StringAttribute{
				Description: "Description of the API key.",
				Optional:    true,
			},
			"expires_at": schema.StringAttribute{
				Description: "RFC3339 timestamp at which the API key expires.",
				Optional:    true,
			},
			"indexes": schema.SetAttribute{
				Description: "Index names or patterns the API key can access.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"referers": schema.SetAttribute{
				Description: "Allowed HTTP referrers for this API key.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"max_hits_per_query": schema.Int64Attribute{
				Description: "Maximum number of results this API key can retrieve in one query.",
				Optional:    true,
			},
			"max_queries_per_ip_per_hour": schema.Int64Attribute{
				Description: "Maximum number of API requests allowed per IP address or user token per hour.",
				Optional:    true,
			},
			"query_parameters": schema.StringAttribute{
				Description: "Query parameters added to every search made with this API key, as a URL query string - " +
					"for example `filters=tenant%3Aacme` to scope the key to one tenant, or `restrictSources=1.2.3.4` to " +
					"restrict it to an IP range. Note that Algolia rejects a `restrictSources` value that does not cover " +
					"the address Terraform itself is applying from.",
				Optional: true,
			},
			"created_at": schema.StringAttribute{
				Description: "RFC3339 timestamp of when the API key was created.",
				Computed:    true,
			},
		},
	}
}

func allowedACLStrings() []string {
	values := make([]string, 0, len(search.AllowedAclEnumValues))
	for _, value := range search.AllowedAclEnumValues {
		values = append(values, string(value))
	}

	return values
}
