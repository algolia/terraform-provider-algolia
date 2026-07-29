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
//
// The model's incoming source set is read as the prior value before being
// overwritten - the plan on create/update, state on read, nothing at all on
// imports and data source reads - so that configured descriptions survive the
// refresh. See flattenDescription for why that matters.
func flattenSources(ctx context.Context, sources []search.Source, model *AllowedSourcesResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	priorDescriptions := priorSourceDescriptions(model.Source)

	models := make([]SourceModel, 0, len(sources))
	for _, s := range sources {
		models = append(models, SourceModel{
			Source:      types.StringValue(s.GetSource()),
			Description: flattenDescription(s.Description, priorDescriptions[s.GetSource()]),
		})
	}

	setVal, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: sourceAttrTypes}, models)
	diags.Append(d...)
	if !diags.HasError() {
		model.Source = setVal
	}

	return diags
}

// priorSourceDescriptions indexes the prior descriptions by source value, which
// is the only key available: the set is unordered, so a response entry can be
// paired with its configured counterpart through nothing but that value. An
// entry with no prior is simply absent from the map, and the zero value the
// lookup then yields is a null string.
func priorSourceDescriptions(prior types.Set) map[string]types.String {
	descriptions := make(map[string]types.String, len(prior.Elements()))
	if prior.IsNull() || prior.IsUnknown() {
		return descriptions
	}

	for _, element := range prior.Elements() {
		objValue, ok := element.(types.Object)
		if !ok {
			continue
		}

		attrs := objValue.Attributes()
		source, ok := attrs["source"].(types.String)
		if !ok || source.IsNull() || source.IsUnknown() {
			continue
		}
		description, ok := attrs["description"].(types.String)
		if !ok {
			continue
		}

		descriptions[source.ValueString()] = description
	}

	return descriptions
}

// flattenDescription resolves one entry's description. `description` is Optional
// and not Computed inside a Required set, so the set Terraform applies has to
// equal the set it planned. An entry configured with an empty description is
// sent without one (see expandSources) and comes back absent, so mapping every
// absent description to null would make Terraform reject that apply as an
// inconsistent result. The configured string is therefore kept whenever it says
// the same thing as the API's response, and only a genuine difference - drift -
// replaces it.
func flattenDescription(description *string, prior types.String) types.String {
	current := ""
	if description != nil {
		current = *description
	}

	if !prior.IsNull() && !prior.IsUnknown() && prior.ValueString() == current {
		return prior
	}

	if current == "" {
		return types.StringNull()
	}

	return types.StringValue(current)
}
