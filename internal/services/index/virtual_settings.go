package index

import "github.com/algolia/algoliasearch-client-go/v4/algolia/search"

// omitVirtualReplicaForbiddenSettings removes settings the Search API inherits
// from the primary index and refuses to accept on virtual replicas. Keeping the
// Terraform model intact lets a snapshot describe the complete effective
// configuration while writes still contain only the virtual-replica overrides.
func omitVirtualReplicaForbiddenSettings(settings *search.IndexSettings) {
	settings.AttributeForDistinct = nil
	settings.AttributesForFaceting = nil
	settings.AttributesToTransliterate = nil
	settings.DecompoundedAttributes = nil
	settings.DisableTypoToleranceOnAttributes = nil
	settings.IndexLanguages = nil
	settings.SearchableAttributes = nil
	settings.SeparatorsToIndex = nil
}
