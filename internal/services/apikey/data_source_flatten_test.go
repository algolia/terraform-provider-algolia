package apikey

import (
	"context"
	"testing"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenAPIKeyDataSource_Full(t *testing.T) {
	createdAt := time.Date(2026, time.April, 7, 12, 0, 0, 0, time.UTC)

	// The API reports createdAt in seconds since the Unix epoch, despite the
	// generated client documenting the field as milliseconds.
	resp := search.NewGetApiKeyResponse(
		"key-value",
		createdAt.Unix(),
		[]search.Acl{search.ACL_SEARCH, search.ACL_BROWSE},
		search.WithGetApiKeyResponseDescription("desc"),
		search.WithGetApiKeyResponseIndexes([]string{"products"}),
		search.WithGetApiKeyResponseReferers([]string{"https://example.com/*"}),
		search.WithGetApiKeyResponseMaxHitsPerQuery(100),
		search.WithGetApiKeyResponseMaxQueriesPerIPPerHour(1000),
		search.WithGetApiKeyResponseQueryParameters("restrictSources=1.2.3.4"),
		search.WithGetApiKeyResponseValidity(3600),
	)

	var model APIKeyDataSourceModel
	diags := flattenAPIKeyDataSource(context.Background(), resp, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.ID.ValueString(); got != "key-value" {
		t.Fatalf("id = %q, want %q", got, "key-value")
	}
	if got := model.Key.ValueString(); got != "key-value" {
		t.Fatalf("key = %q, want %q", got, "key-value")
	}
	if got := model.Description.ValueString(); got != "desc" {
		t.Fatalf("description = %q, want %q", got, "desc")
	}
	if got := model.QueryParameters.ValueString(); got != "restrictSources=1.2.3.4" {
		t.Fatalf("query_parameters = %q, want %q", got, "restrictSources=1.2.3.4")
	}
	if got := model.Validity.ValueInt64(); got != 3600 {
		t.Fatalf("validity = %d, want 3600", got)
	}
	if got := model.MaxHitsPerQuery.ValueInt64(); got != 100 {
		t.Fatalf("max_hits_per_query = %d, want 100", got)
	}
	if got := model.MaxQueriesPerIPPerHour.ValueInt64(); got != 1000 {
		t.Fatalf("max_queries_per_ip_per_hour = %d, want 1000", got)
	}
	if got := model.CreatedAt.ValueString(); got != createdAt.Format(time.RFC3339) {
		t.Fatalf("created_at = %q, want %q", got, createdAt.Format(time.RFC3339))
	}

	aclElements := model.ACL.Elements()
	if len(aclElements) != 2 {
		t.Fatalf("acl = %#v, want 2 entries", aclElements)
	}

	indexElements := model.Indexes.Elements()
	if len(indexElements) != 1 {
		t.Fatalf("indexes = %#v, want 1 entry", indexElements)
	}
	if s, ok := indexElements[0].(types.String); !ok || s.ValueString() != "products" {
		t.Fatalf("indexes[0] = %#v, want %q", indexElements[0], "products")
	}

	refererElements := model.Referers.Elements()
	if len(refererElements) != 1 {
		t.Fatalf("referers = %#v, want 1 entry", refererElements)
	}
}

func TestFlattenAPIKeyDataSource_Minimal(t *testing.T) {
	resp := search.NewGetApiKeyResponse(
		"key-value",
		time.Now().Unix(),
		[]search.Acl{search.ACL_SEARCH},
	)

	var model APIKeyDataSourceModel
	diags := flattenAPIKeyDataSource(context.Background(), resp, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.Description.IsNull() {
		t.Fatalf("description = %#v, want null", model.Description)
	}
	if !model.QueryParameters.IsNull() {
		t.Fatalf("query_parameters = %#v, want null", model.QueryParameters)
	}
	if !model.MaxHitsPerQuery.IsNull() {
		t.Fatalf("max_hits_per_query = %#v, want null", model.MaxHitsPerQuery)
	}
	if !model.Validity.IsNull() {
		t.Fatalf("validity = %#v, want null", model.Validity)
	}
	if len(model.Indexes.Elements()) != 0 {
		t.Fatalf("indexes = %#v, want empty", model.Indexes.Elements())
	}
}

func TestFlattenAPIKeysDataSource_Multiple(t *testing.T) {
	createdAt := time.Date(2026, time.April, 7, 12, 0, 0, 0, time.UTC)

	resp := search.NewListApiKeysResponse([]search.GetApiKeyResponse{
		*search.NewGetApiKeyResponse("key-1", createdAt.Unix(), []search.Acl{search.ACL_SEARCH}),
		*search.NewGetApiKeyResponse(
			"key-2",
			createdAt.Unix(),
			[]search.Acl{search.ACL_BROWSE, search.ACL_SEARCH},
			search.WithGetApiKeyResponseDescription("second key"),
			search.WithGetApiKeyResponseQueryParameters("filters=tenant%3Aacme"),
		),
	})

	var model APIKeysDataSourceModel
	diags := flattenAPIKeysDataSource(context.Background(), resp, "app-123", &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.ID.ValueString(); got != "app-123" {
		t.Fatalf("id = %q, want %q", got, "app-123")
	}

	elements := model.Keys.Elements()
	if len(elements) != 2 {
		t.Fatalf("keys = %#v, want 2 entries", elements)
	}

	first, ok := elements[0].(types.Object)
	if !ok {
		t.Fatalf("keys[0] = %#v, want types.Object", elements[0])
	}
	if v, ok := first.Attributes()["value"].(types.String); !ok || v.ValueString() != "key-1" {
		t.Fatalf("keys[0].value = %#v, want %q", first.Attributes()["value"], "key-1")
	}
	if v, ok := first.Attributes()["created_at"].(types.String); !ok || v.ValueString() != createdAt.Format(time.RFC3339) {
		t.Fatalf("keys[0].created_at = %#v, want %q", first.Attributes()["created_at"], createdAt.Format(time.RFC3339))
	}

	second, ok := elements[1].(types.Object)
	if !ok {
		t.Fatalf("keys[1] = %#v, want types.Object", elements[1])
	}
	if v, ok := second.Attributes()["description"].(types.String); !ok || v.ValueString() != "second key" {
		t.Fatalf("keys[1].description = %#v, want %q", second.Attributes()["description"], "second key")
	}
	if v, ok := second.Attributes()["query_parameters"].(types.String); !ok || v.ValueString() != "filters=tenant%3Aacme" {
		t.Fatalf("keys[1].query_parameters = %#v, want %q", second.Attributes()["query_parameters"], "filters=tenant%3Aacme")
	}
}

func TestFlattenAPIKeysDataSource_Empty(t *testing.T) {
	resp := search.NewListApiKeysResponse(nil)

	var model APIKeysDataSourceModel
	diags := flattenAPIKeysDataSource(context.Background(), resp, "app-123", &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Keys.IsNull() {
		t.Fatal("expected a non-null keys list")
	}
	if len(model.Keys.Elements()) != 0 {
		t.Fatalf("keys = %#v, want empty", model.Keys.Elements())
	}
}

func TestAclStrings(t *testing.T) {
	got := aclStrings([]search.Acl{search.ACL_SEARCH, search.ACL_BROWSE})
	if len(got) != 2 || got[0] != "search" || got[1] != "browse" {
		t.Fatalf("aclStrings = %#v, want [search browse]", got)
	}

	if got := aclStrings(nil); len(got) != 0 {
		t.Fatalf("aclStrings(nil) = %#v, want empty", got)
	}
}
