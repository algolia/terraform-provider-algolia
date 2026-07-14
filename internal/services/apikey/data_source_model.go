package apikey

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// APIKeyDataSourceModel describes the algolia_api_key data source: a
// read-only lookup of a single API key's metadata by its key value.
type APIKeyDataSourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Key                    types.String `tfsdk:"key"`
	ACL                    types.List   `tfsdk:"acl"`
	Description            types.String `tfsdk:"description"`
	Indexes                types.List   `tfsdk:"indexes"`
	MaxHitsPerQuery        types.Int64  `tfsdk:"max_hits_per_query"`
	MaxQueriesPerIPPerHour types.Int64  `tfsdk:"max_queries_per_ip_per_hour"`
	QueryParameters        types.String `tfsdk:"query_parameters"`
	Referers               types.List   `tfsdk:"referers"`
	Validity               types.Int64  `tfsdk:"validity"`
	CreatedAt              types.String `tfsdk:"created_at"`
}

// APIKeysDataSourceModel describes the algolia_api_keys data source: a
// listing of every API key configured for the application.
type APIKeysDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Keys types.List   `tfsdk:"keys"`
}

// APIKeyListItemModel describes a single entry within the algolia_api_keys
// "keys" list. It mirrors APIKeyDataSourceModel but also carries the key's
// own value, since there's no separate "key" input to bind it to.
type APIKeyListItemModel struct {
	Value                  types.String `tfsdk:"value"`
	ACL                    types.List   `tfsdk:"acl"`
	Description            types.String `tfsdk:"description"`
	Indexes                types.List   `tfsdk:"indexes"`
	MaxHitsPerQuery        types.Int64  `tfsdk:"max_hits_per_query"`
	MaxQueriesPerIPPerHour types.Int64  `tfsdk:"max_queries_per_ip_per_hour"`
	QueryParameters        types.String `tfsdk:"query_parameters"`
	Referers               types.List   `tfsdk:"referers"`
	Validity               types.Int64  `tfsdk:"validity"`
	CreatedAt              types.String `tfsdk:"created_at"`
}

// apiKeyListItemAttrTypes mirrors the "keys" nested object schema exactly;
// used to convert []APIKeyListItemModel to types.List.
var apiKeyListItemAttrTypes = map[string]attr.Type{
	"value":                       types.StringType,
	"acl":                         types.ListType{ElemType: types.StringType},
	"description":                 types.StringType,
	"indexes":                     types.ListType{ElemType: types.StringType},
	"max_hits_per_query":          types.Int64Type,
	"max_queries_per_ip_per_hour": types.Int64Type,
	"query_parameters":            types.StringType,
	"referers":                    types.ListType{ElemType: types.StringType},
	"validity":                    types.Int64Type,
	"created_at":                  types.StringType,
}
