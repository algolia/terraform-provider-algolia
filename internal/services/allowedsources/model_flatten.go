package allowedsources

import (
	"context"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// flattenSources converts the Algolia []Source response into the Terraform
// resource/data source model. The allowed sources allowlist is an
// application-level singleton that always exists (an empty API response
// simply means no IP restrictions are configured), so this always produces
// a non-null source set, never a null one.
func flattenSources(ctx context.Context, sources []search.Source, model *AllowedSourcesResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	models := make([]SourceModel, 0, len(sources))
	for _, s := range sources {
		entry := SourceModel{
			Source:      types.StringValue(s.GetSource()),
			Description: types.StringNull(),
		}
		if desc, ok := s.GetDescriptionOk(); ok && desc != nil && *desc != "" {
			entry.Description = types.StringValue(*desc)
		}
		models = append(models, entry)
	}

	setVal, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: sourceAttrTypes}, models)
	diags.Append(d...)
	if !diags.HasError() {
		model.Source = setVal
	}

	return diags
}
