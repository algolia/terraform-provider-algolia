package mcm

import (
	"context"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenClustersDataSource_Basic(t *testing.T) {
	resp := search.NewListClustersResponse([]string{"c1-test", "c2-test"})

	var model ClustersDataSourceModel
	diags := flattenClustersDataSource(context.Background(), resp, "APPID123", &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.ID.ValueString(); got != "APPID123" {
		t.Fatalf("id = %q, want %q", got, "APPID123")
	}

	elements := model.Clusters.Elements()
	if len(elements) != 2 {
		t.Fatalf("clusters = %#v, want 2 entries", elements)
	}

	first := elements[0].(types.Object).Attributes()
	if v, ok := first["cluster_name"].(types.String); !ok || v.ValueString() != "c1-test" {
		t.Fatalf("cluster_name = %#v, want %q", first["cluster_name"], "c1-test")
	}

	second := elements[1].(types.Object).Attributes()
	if v, ok := second["cluster_name"].(types.String); !ok || v.ValueString() != "c2-test" {
		t.Fatalf("cluster_name = %#v, want %q", second["cluster_name"], "c2-test")
	}
}

func TestFlattenClustersDataSource_Empty(t *testing.T) {
	resp := search.NewListClustersResponse(nil)

	var model ClustersDataSourceModel
	diags := flattenClustersDataSource(context.Background(), resp, "APPID123", &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.Clusters.IsNull() {
		t.Fatal("expected a non-null clusters list")
	}
	if len(model.Clusters.Elements()) != 0 {
		t.Fatalf("clusters = %#v, want empty", model.Clusters.Elements())
	}
}

func TestFlattenClustersDataSource_NilResponse(t *testing.T) {
	var model ClustersDataSourceModel
	diags := flattenClustersDataSource(context.Background(), nil, "APPID123", &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if len(model.Clusters.Elements()) != 0 {
		t.Fatalf("clusters = %#v, want empty", model.Clusters.Elements())
	}
}
