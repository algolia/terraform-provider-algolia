package index

import (
	"context"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenIndicesDataSource_Basic(t *testing.T) {
	items := []search.FetchedIndex{
		*search.NewFetchedIndex(
			"products",
			"2026-01-01T00:00:00Z",
			"2026-01-02T00:00:00Z",
			100,
			2048,
			4096,
			5,
			0,
			false,
		),
	}

	var model IndicesDataSourceModel
	diags := flattenIndicesDataSource(context.Background(), items, "APPID123", &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.ID.ValueString(); got != "APPID123" {
		t.Fatalf("id = %q, want %q", got, "APPID123")
	}

	elements := model.Indices.Elements()
	if len(elements) != 1 {
		t.Fatalf("indices = %#v, want 1 entry", elements)
	}

	obj, ok := elements[0].(types.Object)
	if !ok {
		t.Fatalf("indices[0] = %#v, want types.Object", elements[0])
	}
	attrs := obj.Attributes()

	if v, ok := attrs["name"].(types.String); !ok || v.ValueString() != "products" {
		t.Fatalf("name = %#v, want %q", attrs["name"], "products")
	}
	if v, ok := attrs["entries"].(types.Int64); !ok || v.ValueInt64() != 100 {
		t.Fatalf("entries = %#v, want 100", attrs["entries"])
	}
	if v, ok := attrs["data_size"].(types.Int64); !ok || v.ValueInt64() != 2048 {
		t.Fatalf("data_size = %#v, want 2048", attrs["data_size"])
	}
	if v, ok := attrs["file_size"].(types.Int64); !ok || v.ValueInt64() != 4096 {
		t.Fatalf("file_size = %#v, want 4096", attrs["file_size"])
	}
	if v, ok := attrs["pending_task"].(types.Bool); !ok || v.ValueBool() {
		t.Fatalf("pending_task = %#v, want false", attrs["pending_task"])
	}
	if v, ok := attrs["primary"].(types.String); !ok || !v.IsNull() {
		t.Fatalf("primary = %#v, want null", attrs["primary"])
	}
	if v, ok := attrs["virtual"].(types.Bool); !ok || !v.IsNull() {
		t.Fatalf("virtual = %#v, want null", attrs["virtual"])
	}
	replicas, ok := attrs["replicas"].(types.List)
	if !ok || !replicas.IsNull() {
		t.Fatalf("replicas = %#v, want null list", attrs["replicas"])
	}
}

func TestFlattenIndicesDataSource_ReplicaWithVirtualAndReplicas(t *testing.T) {
	items := []search.FetchedIndex{
		*search.NewFetchedIndex(
			"products_replica",
			"2026-01-01T00:00:00Z",
			"2026-01-02T00:00:00Z",
			50,
			1024,
			2048,
			2,
			0,
			false,
			search.WithFetchedIndexPrimary("products"),
			search.WithFetchedIndexVirtual(true),
		),
		*search.NewFetchedIndex(
			"products",
			"2026-01-01T00:00:00Z",
			"2026-01-02T00:00:00Z",
			100,
			2048,
			4096,
			5,
			0,
			false,
			search.WithFetchedIndexReplicas([]string{"products_replica"}),
		),
	}

	var model IndicesDataSourceModel
	diags := flattenIndicesDataSource(context.Background(), items, "APPID123", &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	elements := model.Indices.Elements()
	if len(elements) != 2 {
		t.Fatalf("indices = %#v, want 2 entries", elements)
	}

	replica := elements[0].(types.Object).Attributes()
	if v, ok := replica["primary"].(types.String); !ok || v.ValueString() != "products" {
		t.Fatalf("primary = %#v, want %q", replica["primary"], "products")
	}
	if v, ok := replica["virtual"].(types.Bool); !ok || !v.ValueBool() {
		t.Fatalf("virtual = %#v, want true", replica["virtual"])
	}

	primary := elements[1].(types.Object).Attributes()
	replicasList, ok := primary["replicas"].(types.List)
	if !ok {
		t.Fatalf("replicas = %#v, want types.List", primary["replicas"])
	}
	replicaElements := replicasList.Elements()
	if len(replicaElements) != 1 {
		t.Fatalf("replicas = %#v, want 1 entry", replicaElements)
	}
	if s, ok := replicaElements[0].(types.String); !ok || s.ValueString() != "products_replica" {
		t.Fatalf("replicas[0] = %#v, want %q", replicaElements[0], "products_replica")
	}
}

func TestFlattenIndicesDataSource_Empty(t *testing.T) {
	var model IndicesDataSourceModel
	diags := flattenIndicesDataSource(context.Background(), nil, "APPID123", &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Indices.IsNull() {
		t.Fatal("expected a non-null indices list")
	}
	if len(model.Indices.Elements()) != 0 {
		t.Fatalf("indices = %#v, want empty", model.Indices.Elements())
	}
}
