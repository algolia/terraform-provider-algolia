package index

import (
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
)

func TestOmitVirtualReplicaForbiddenSettings(t *testing.T) {
	attributeForDistinct := "sku"
	separatorsToIndex := "+-"
	maxFacetHits := int32(42)
	relevancyStrictness := int32(80)
	attributeCriteriaComputedByMinProximity := true

	settings := &search.IndexSettings{
		AttributeForDistinct:                    &attributeForDistinct,
		AttributesForFaceting:                   []string{"brand"},
		AttributesToTransliterate:               []string{"name"},
		DecompoundedAttributes:                  map[string]any{"de": []string{"name"}},
		DisableTypoToleranceOnAttributes:        []string{"sku"},
		IndexLanguages:                          []search.SupportedLanguage{search.SupportedLanguage("en")},
		SearchableAttributes:                    []string{"name"},
		SeparatorsToIndex:                       &separatorsToIndex,
		CustomRanking:                           []string{"desc(popularity)"},
		MaxFacetHits:                            &maxFacetHits,
		RelevancyStrictness:                     &relevancyStrictness,
		DisableTypoToleranceOnWords:             []string{"sku"},
		AttributesToHighlight:                   []string{"name"},
		AttributeCriteriaComputedByMinProximity: &attributeCriteriaComputedByMinProximity,
	}

	omitVirtualReplicaForbiddenSettings(settings)

	if settings.AttributeForDistinct != nil || settings.AttributesForFaceting != nil ||
		settings.AttributesToTransliterate != nil || settings.DecompoundedAttributes != nil ||
		settings.DisableTypoToleranceOnAttributes != nil || settings.IndexLanguages != nil ||
		settings.SearchableAttributes != nil || settings.SeparatorsToIndex != nil {
		t.Errorf("forbidden virtual replica settings were not all omitted: %#v", settings)
	}

	if got, want := settings.CustomRanking, []string{"desc(popularity)"}; len(got) != len(want) || got[0] != want[0] {
		t.Errorf("CustomRanking = %v, want %v", got, want)
	}
	if settings.MaxFacetHits == nil || *settings.MaxFacetHits != maxFacetHits {
		t.Errorf("MaxFacetHits = %v, want %d", settings.MaxFacetHits, maxFacetHits)
	}
	if settings.RelevancyStrictness == nil || *settings.RelevancyStrictness != relevancyStrictness {
		t.Errorf("RelevancyStrictness = %v, want %d", settings.RelevancyStrictness, relevancyStrictness)
	}
	if got, want := settings.DisableTypoToleranceOnWords, []string{"sku"}; len(got) != len(want) || got[0] != want[0] {
		t.Errorf("DisableTypoToleranceOnWords = %v, want %v", got, want)
	}
	if got, want := settings.AttributesToHighlight, []string{"name"}; len(got) != len(want) || got[0] != want[0] {
		t.Errorf("AttributesToHighlight = %v, want %v", got, want)
	}
	if settings.AttributeCriteriaComputedByMinProximity == nil || !*settings.AttributeCriteriaComputedByMinProximity {
		t.Error("AttributeCriteriaComputedByMinProximity was unexpectedly omitted")
	}
}
