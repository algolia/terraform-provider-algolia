package allowedsources

import (
	"context"
	"sort"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// emptyAllowedSourcesDetail explains why an empty allowlist is rejected and
// how to actually clear it. Reused across the null/unknown and zero-length
// guards below.
const emptyAllowedSourcesDetail = "At least one source is required: the Algolia API rejects an empty allowed-sources list. " +
	"To clear the allowlist entirely, remove the algolia_allowed_sources resource from your configuration (destroy) instead."

// expandSources converts the Terraform source set into the Algolia
// []Source used by ReplaceSources. It surfaces explicit diagnostics rather
// than silently producing an empty slice: because this resource controls API
// access, quietly dropping entries could remove the caller's own IP (lockout)
// or produce an empty ReplaceSources payload that only fails with the client's
// generic "source is required" error.
func expandSources(ctx context.Context, set types.Set) ([]search.Source, diag.Diagnostics) {
	var diags diag.Diagnostics

	if set.IsNull() || set.IsUnknown() {
		diags.AddError("Missing allowed sources", emptyAllowedSourcesDetail)
		return nil, diags
	}

	var models []SourceModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}

	sources := make([]search.Source, 0, len(models))
	for _, m := range models {
		if m.Source.IsNull() || m.Source.IsUnknown() || m.Source.ValueString() == "" {
			diags.AddError(
				"Invalid allowed source",
				"Each source entry must have a non-empty \"source\" value (an IP address or CIDR range).",
			)
			continue
		}

		entry := search.NewSource(m.Source.ValueString())
		if !m.Description.IsNull() && !m.Description.IsUnknown() && m.Description.ValueString() != "" {
			entry.SetDescription(m.Description.ValueString())
		}

		sources = append(sources, *entry)
	}

	if diags.HasError() {
		return nil, diags
	}

	if len(sources) == 0 {
		diags.AddError("Missing allowed sources", emptyAllowedSourcesDetail)
		return nil, diags
	}

	// Sets are unordered in Terraform; sort by source value so the request
	// body sent to ReplaceSources is deterministic (helpful for tests/logs).
	sort.Slice(sources, func(i, j int) bool { return sources[i].Source < sources[j].Source })

	return sources, diags
}
