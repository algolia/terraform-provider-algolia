package index

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestVirtualRankingToIndexObject(t *testing.T) {
	virtualRanking, diags := types.ObjectValue(virtualRankingAttrTypes, map[string]attr.Value{
		"custom_ranking":       types.ListValueMust(types.StringType, []attr.Value{types.StringValue("desc(popularity)")}),
		"relevancy_strictness": types.Int64Value(80),
	})
	if diags.HasError() {
		t.Fatalf("build virtual ranking object: %v", diags)
	}

	indexRanking := virtualRankingToIndexObject(virtualRanking)

	if indexRanking.IsNull() || indexRanking.IsUnknown() {
		t.Fatalf("expected known index ranking object, got %v", indexRanking)
	}

	indexAttrs := indexRanking.Attributes()
	if _, ok := indexAttrs["ranking"]; !ok {
		t.Fatalf("expected normalized index ranking object to include ranking field")
	}
	if !indexAttrs["ranking"].IsNull() {
		t.Fatalf("expected normalized ranking field to be null, got %v", indexAttrs["ranking"])
	}
}

func TestIndexToVirtualRankingObject(t *testing.T) {
	indexRanking, diags := types.ObjectValue(rankingAttrTypes, map[string]attr.Value{
		"ranking":              types.ListValueMust(types.StringType, []attr.Value{types.StringValue("typo"), types.StringValue("custom")}),
		"custom_ranking":       types.ListValueMust(types.StringType, []attr.Value{types.StringValue("desc(popularity)")}),
		"relevancy_strictness": types.Int64Value(60),
	})
	if diags.HasError() {
		t.Fatalf("build index ranking object: %v", diags)
	}

	virtualRanking := indexToVirtualRankingObject(indexRanking)

	if virtualRanking.IsNull() || virtualRanking.IsUnknown() {
		t.Fatalf("expected known virtual ranking object, got %v", virtualRanking)
	}

	virtualAttrs := virtualRanking.Attributes()
	if _, ok := virtualAttrs["ranking"]; ok {
		t.Fatalf("expected stripped virtual ranking object to omit ranking field")
	}
	if got := virtualAttrs["relevancy_strictness"].(types.Int64).ValueInt64(); got != 60 {
		t.Fatalf("unexpected relevancy strictness %d", got)
	}
}
