package synonym

import (
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildSynonymRequestRegular(t *testing.T) {
	model := SynonymResourceModel{
		IndexName:    types.StringValue("products"),
		ObjectID:     types.StringValue("syn-1"),
		Type:         types.StringValue("synonym"),
		Synonyms:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("iphone"), types.StringValue("ios phone")}),
		Input:        types.StringNull(),
		Word:         types.StringNull(),
		Corrections:  types.SetNull(types.StringType),
		Placeholder:  types.StringNull(),
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

func TestHydrateSynonymModelPreservesPriorEmptiness(t *testing.T) {
	emptySet := types.SetValueMust(types.StringType, []attr.Value{})
	valuedSet := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("ios phone")})

	tests := []struct {
		name        string
		prior       types.Set
		apiSynonyms []string
		want        types.Set
	}{
		{
			name:  "API empty and prior null stays null",
			prior: types.SetNull(types.StringType),
			want:  types.SetNull(types.StringType),
		},
		{
			name:  "API empty and prior known empty stays known empty",
			prior: emptySet,
			want:  emptySet,
		},
		{
			name:        "API values win over the prior",
			prior:       emptySet,
			apiSynonyms: []string{"ios phone"},
			want:        valuedSet,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hit := search.NewSynonymHit("syn-4", search.SYNONYM_TYPE_SYNONYM)
			if test.apiSynonyms != nil {
				hit.Synonyms = test.apiSynonyms
			}

			// Every collection attribute shares the same contract, so all three
			// are driven off the same prior in each case.
			model := SynonymResourceModel{
				Synonyms:     test.prior,
				Corrections:  test.prior,
				Replacements: test.prior,
			}

			if diags := hydrateSynonymModel("products", hit, &model); diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}

			if !model.Synonyms.Equal(test.want) {
				t.Errorf("synonyms = %s, want %s", model.Synonyms, test.want)
			}

			wantOthers := test.prior
			if test.prior.IsNull() {
				wantOthers = types.SetNull(types.StringType)
			}
			if !model.Corrections.Equal(wantOthers) {
				t.Errorf("corrections = %s, want %s", model.Corrections, wantOthers)
			}
			if !model.Replacements.Equal(wantOthers) {
				t.Errorf("replacements = %s, want %s", model.Replacements, wantOthers)
			}
		})
	}
}

func TestHydrateSynonymModelWithoutPriorState(t *testing.T) {
	// Imports and data source reads start from a zero-valued model, where every
	// collection is null, so an API response without collections stays null.
	hit := search.NewSynonymHit("syn-5", search.SYNONYM_TYPE_SYNONYM)

	model := SynonymResourceModel{}
	if diags := hydrateSynonymModel("products", hit, &model); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	for name, value := range map[string]types.Set{
		"synonyms":     model.Synonyms,
		"corrections":  model.Corrections,
		"replacements": model.Replacements,
	} {
		if !value.Equal(types.SetNull(types.StringType)) {
			t.Errorf("%s = %s, want null", name, value)
		}
	}
}
