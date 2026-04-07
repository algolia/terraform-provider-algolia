package apikey

import (
	"testing"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildApiKeyRequest(t *testing.T) {
	now := time.Date(2026, time.April, 7, 12, 0, 0, 0, time.UTC)

	model := APIKeyResourceModel{
		ACL:                    types.SetValueMust(types.StringType, []attr.Value{types.StringValue("search"), types.StringValue("browse")}),
		Description:            types.StringValue("test key"),
		ExpiresAt:              types.StringValue("2026-04-07T13:00:00Z"),
		Indexes:                types.SetValueMust(types.StringType, []attr.Value{types.StringValue("products")}),
		Referers:               types.SetValueMust(types.StringType, []attr.Value{types.StringValue("https://example.com/*")}),
		MaxHitsPerQuery:        types.Int64Value(100),
		MaxQueriesPerIPPerHour: types.Int64Value(200),
	}

	apiKey, diags := buildAPIKeyRequest(&model, now)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := apiKey.GetDescription(); got != "test key" {
		t.Fatalf("description = %q, want %q", got, "test key")
	}
	if got := apiKey.GetValidity(); got != 3600 {
		t.Fatalf("validity = %d, want %d", got, 3600)
	}
	if got := apiKey.GetMaxHitsPerQuery(); got != 100 {
		t.Fatalf("max hits = %d, want %d", got, 100)
	}
	if got := apiKey.GetMaxQueriesPerIPPerHour(); got != 200 {
		t.Fatalf("max queries = %d, want %d", got, 200)
	}
	if got := apiKey.GetIndexes(); len(got) != 1 || got[0] != "products" {
		t.Fatalf("indexes = %#v, want [products]", got)
	}
	if got := apiKey.GetReferers(); len(got) != 1 || got[0] != "https://example.com/*" {
		t.Fatalf("referers = %#v, want [https://example.com/*]", got)
	}
	if got := apiKey.GetAcl(); len(got) != 2 {
		t.Fatalf("acl = %#v, want 2 entries", got)
	}
}

func TestHydrateAPIKeyModelPreservesConfiguredExpiry(t *testing.T) {
	model := APIKeyResourceModel{
		ExpiresAt: types.StringValue("2030-01-01T00:00:00Z"),
	}

	resp := search.NewGetApiKeyResponse(
		"key-value",
		1712486400000,
		[]search.Acl{search.ACL_SEARCH},
		search.WithGetApiKeyResponseDescription("desc"),
		search.WithGetApiKeyResponseIndexes([]string{"products"}),
		search.WithGetApiKeyResponseReferers([]string{"https://example.com/*"}),
	)

	diags := hydrateAPIKeyModel(resp, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.ID.ValueString(); got != "key-value" {
		t.Fatalf("id = %q, want %q", got, "key-value")
	}
	if got := model.Description.ValueString(); got != "desc" {
		t.Fatalf("description = %q, want %q", got, "desc")
	}
	if got := model.ExpiresAt.ValueString(); got != "2030-01-01T00:00:00Z" {
		t.Fatalf("expires_at = %q, want preserved config", got)
	}
	if got := model.CreatedAt.ValueString(); got == "" {
		t.Fatal("created_at should be set")
	}
}

func TestAPIKeyResponseMatches_WithExpiringKeyTolerance(t *testing.T) {
	now := time.Date(2026, time.April, 7, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(2 * time.Hour)

	expected := search.NewApiKey([]search.Acl{search.ACL_BROWSE, search.ACL_SEARCH})
	expected.SetDescription("updated key")
	expected.SetIndexes([]string{"products_*"})
	expected.SetReferers([]string{"https://example.com/*"})
	expected.SetMaxHitsPerQuery(200)
	expected.SetMaxQueriesPerIPPerHour(1000)
	expected.SetValidity(int32(expiresAt.Sub(now).Seconds()))

	response := search.NewGetApiKeyResponse(
		"key-value",
		now.UnixMilli(),
		[]search.Acl{search.ACL_SEARCH, search.ACL_BROWSE},
		search.WithGetApiKeyResponseDescription("updated key"),
		search.WithGetApiKeyResponseIndexes([]string{"products_*"}),
		search.WithGetApiKeyResponseReferers([]string{"https://example.com/*"}),
		search.WithGetApiKeyResponseMaxHitsPerQuery(200),
		search.WithGetApiKeyResponseMaxQueriesPerIPPerHour(1000),
		search.WithGetApiKeyResponseValidity(int32(expiresAt.Sub(now).Seconds()-3)),
	)

	if !apiKeyResponseMatches(response, expected, &expiresAt, now) {
		t.Fatal("expected API key response to match within validity tolerance")
	}
}

func TestAPIKeyResponseMatches_DetectsMismatch(t *testing.T) {
	now := time.Date(2026, time.April, 7, 12, 0, 0, 0, time.UTC)

	expected := search.NewApiKey([]search.Acl{search.ACL_SEARCH})
	expected.SetDescription("expected")

	response := search.NewGetApiKeyResponse(
		"key-value",
		now.UnixMilli(),
		[]search.Acl{search.ACL_SEARCH},
		search.WithGetApiKeyResponseDescription("actual"),
	)

	if apiKeyResponseMatches(response, expected, nil, now) {
		t.Fatal("expected API key response mismatch to be detected")
	}
}
