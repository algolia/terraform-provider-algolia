package synonym

import (
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildSynonymRequestRegular(t *testing.T) {
	model := SynonymResourceModel{
		IndexName: types.StringValue("products"),
		ObjectID:  types.StringValue("syn-1"),
		Type:      types.StringValue("synonym"),
		Synonyms:  types.SetValueMust(types.StringType, []attr.Value{types.StringValue("iphone"), types.StringValue("ios phone")}),
		Input:     types.StringNull(),
		Word:      types.StringNull(),
		Corrections: types.SetNull(types.StringType),
		Placeholder: types.StringNull(),
		Replacements: types.SetNull(types.StringType),
	}

	hit, diags := buildSynonymHit(&model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := hit.GetObjectID(); got != "syn-1" {
		t.Fatalf("object_id = %q, want %q", got, "syn-1")
	}
	if got := canonicalSynonymType(string(hit.GetType())); got != "synonym" {
		t.Fatalf("type = %q, want %q", got, "synonym")
	}
	if got := hit.GetSynonyms(); len(got) != 2 {
		t.Fatalf("synonyms = %#v, want 2 values", got)
	}
}

func TestBuildSynonymRequestPlaceholder(t *testing.T) {
	model := SynonymResourceModel{
		IndexName:    types.StringValue("products"),
		ObjectID:     types.StringValue("syn-2"),
		Type:         types.StringValue("placeholder"),
		Synonyms:     types.SetNull(types.StringType),
		Input:        types.StringNull(),
		Word:         types.StringNull(),
		Corrections:  types.SetNull(types.StringType),
		Placeholder:  types.StringValue("<brand>"),
		Replacements: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("apple"), types.StringValue("samsung")}),
	}

	hit, diags := buildSynonymHit(&model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := hit.GetPlaceholder(); got != "<brand>" {
		t.Fatalf("placeholder = %q, want %q", got, "<brand>")
	}
	if got := hit.GetReplacements(); len(got) != 2 {
		t.Fatalf("replacements = %#v, want 2 values", got)
	}
}

func TestHydrateSynonymModel(t *testing.T) {
	hit := search.NewSynonymHit(
		"syn-3",
		search.SYNONYM_TYPE_ONE_WAY_SYNONYM,
		search.WithSynonymHitInput("iphone"),
		search.WithSynonymHitSynonyms([]string{"ios phone", "apple phone"}),
	)

	model := SynonymResourceModel{}
	diags := hydrateSynonymModel("products", hit, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.ID.ValueString(); got != "products/syn-3" {
		t.Fatalf("id = %q, want composite id", got)
	}
	if got := model.Type.ValueString(); got != "oneWaySynonym" {
		t.Fatalf("type = %q, want oneWaySynonym", got)
	}
	if got := model.Input.ValueString(); got != "iphone" {
		t.Fatalf("input = %q, want iphone", got)
	}
}
