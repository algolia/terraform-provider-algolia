package allowedsources

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	schemavalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func allowedSourcesResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages the Algolia app-level allowed sources: the complete allowlist of source IP " +
			"addresses and CIDR ranges permitted to use the Algolia API for this application (the \"Sources\" " +
			"security setting). This is a singleton resource — there is exactly one allowed sources " +
			"configuration per Algolia application, and every apply REPLACES THE ENTIRE LIST with exactly the " +
			"`source` set in configuration.\n\n" +
			"WARNING — lockout risk: once any source is configured, only the IP addresses/ranges listed in " +
			"`source` may call the Algolia API for this application. This includes the machine or CI runner " +
			"applying this Terraform configuration. If that IP address is not included in `source`, applying " +
			"this resource will lock yourself — and Terraform itself — out of the application's API, and this " +
			"provider cannot be used to undo it (you would need to fix the allowlist via the Algolia dashboard " +
			"or support). Always include the IP address(es)/range(s) of every host that runs Terraform against " +
			"this application before applying.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform identifier for the singleton allowed sources resource. Set to the Algolia application ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source": schema.SetNestedAttribute{
				Description: "The complete set of allowed source IP addresses/ranges. Order-independent; every " +
					"apply replaces the full allowlist with exactly this set, so any address missing here is " +
					"removed from the allowlist — including, potentially, the address Terraform itself is " +
					"running from (see the lockout warning above). Must contain at least one entry; to remove " +
					"all IP restrictions, destroy this resource instead of emptying the set.",
				Required: true,
				Validators: []schemavalidator.Set{
					setvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"source": schema.StringAttribute{
							Description: "The allowed IP address or CIDR range, e.g. \"1.2.3.4\" or \"10.0.0.0/24\".",
							Required:    true,
							Validators: []schemavalidator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
						"description": schema.StringAttribute{
							Description: "Human-readable description of this source.",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}
