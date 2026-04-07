package apikey

import "github.com/hashicorp/terraform-plugin-framework/types"

type APIKeyResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	ACL                   types.Set    `tfsdk:"acl"`
	Description           types.String `tfsdk:"description"`
	ExpiresAt             types.String `tfsdk:"expires_at"`
	Indexes               types.Set    `tfsdk:"indexes"`
	Referers              types.Set    `tfsdk:"referers"`
	MaxHitsPerQuery       types.Int64  `tfsdk:"max_hits_per_query"`
	MaxQueriesPerIPPerHour types.Int64 `tfsdk:"max_queries_per_ip_per_hour"`
	CreatedAt             types.String `tfsdk:"created_at"`
}
