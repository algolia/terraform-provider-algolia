package collection

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// parseCollectionImportID splits an import identifier of the form
// "<index_name>/<collection_id>" into its two parts. The compound form is
// needed because the Collections API's GET response omits the indexName and
// we can't reconstruct it from a UUID alone.
func parseCollectionImportID(id string) (string, string, error) {
	i := strings.Index(id, "/")
	if i <= 0 || i == len(id)-1 {
		return "", "", fmt.Errorf("expected import ID in the form <index_name>/<collection_id>, got: %q", id)
	}
	return id[:i], id[i+1:], nil
}

// hydrateCollectionResourceState fills the resource model from an API response,
// preserving the local-only deletion_protection and commit flags.
func hydrateCollectionResourceState(
	ctx context.Context,
	resp *CollectionResponse,
	commit, deletionProtection types.Bool,
	model *CollectionResourceModel,
) diag.Diagnostics {
	diags := flattenCollectionResponse(ctx, resp, model)
	if diags.HasError() {
		return diags
	}

	model.Commit = commitValue(commit)
	model.DeletionProtection = deletionProtectionValue(deletionProtection)

	return diags
}

// hydrateImportedCollectionResourceState is the entry point for terraform import.
// It always enables deletion protection and assumes commit=true so subsequent
// applies don't silently disable auto-commit.
func hydrateImportedCollectionResourceState(ctx context.Context, resp *CollectionResponse, model *CollectionResourceModel) diag.Diagnostics {
	return hydrateCollectionResourceState(ctx, resp, types.BoolValue(true), types.BoolValue(true), model)
}

// commitValue defaults the commit flag to true when unknown or null.
func commitValue(v types.Bool) types.Bool {
	if v.IsNull() || v.IsUnknown() {
		return types.BoolValue(true)
	}
	return v
}

// deletionProtectionValue defaults deletion protection to true when unknown or null.
func deletionProtectionValue(v types.Bool) types.Bool {
	if v.IsNull() || v.IsUnknown() {
		return types.BoolValue(true)
	}
	return v
}
