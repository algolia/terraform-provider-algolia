package allowedsources

import (
	"context"
	"sort"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// expandSources converts the Terraform source set into the Algolia
// []Source used by ReplaceSources. A null/unknown set expands to an empty
// slice; in practice this only happens when the resource has been removed
// from configuration, since the schema requires at least one entry while
// the resource is being managed.
func expandSources(ctx context.Context, set types.Set) ([]search.Source, diag.Diagnostics) {
	var diags diag.Diagnostics

	if set.IsNull() || set.IsUnknown() {
		return nil, diags
	}

	var models []SourceModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}

	sources := make([]search.Source, 0, len(models))
	for _, m := range models {
		if m.Source.IsNull() || m.Source.IsUnknown() {
			continue
		}

		entry := search.NewSource(m.Source.ValueString())
		if !m.Description.IsNull() && !m.Description.IsUnknown() && m.Description.ValueString() != "" {
			entry.SetDescription(m.Description.ValueString())
		}

		sources = append(sources, *entry)
	}

	// Sets are unordered in Terraform; sort by source value so the request
	// body sent to ReplaceSources is deterministic (helpful for tests/logs).
	sort.Slice(sources, func(i, j int) bool { return sources[i].Source < sources[j].Source })

	return sources, diags
}
