package query_suggestions

import (
	"context"
	"encoding/json"
	"fmt"

	suggestions "github.com/algolia/algoliasearch-client-go/v4/algolia/query-suggestions"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// expandConfigurationWithIndex converts the Terraform model into a ConfigurationWithIndex for create/update calls.
func expandConfigurationWithIndex(ctx context.Context, model *QuerySuggestionsConfigResourceModel) (*suggestions.ConfigurationWithIndex, diag.Diagnostics) {
	var diags diag.Diagnostics

	sourceIndices, d := expandSourceIndices(ctx, model.SourceIndices)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	cfg := suggestions.NewConfigurationWithIndex(sourceIndices, model.IndexName.ValueString())

	if langs, d := expandLanguages(ctx, model); d.HasError() {
		diags.Append(d...)
		return nil, diags
	} else if langs != nil {
		cfg.SetLanguages(langs)
	}

	if !model.Exclude.IsNull() && !model.Exclude.IsUnknown() {
		var exclude []string
		diags.Append(model.Exclude.ElementsAs(ctx, &exclude, false)...)
		if diags.HasError() {
			return nil, diags
		}
		cfg.SetExclude(exclude)
	}

	if !model.EnablePersonalization.IsNull() && !model.EnablePersonalization.IsUnknown() {
		cfg.SetEnablePersonalization(model.EnablePersonalization.ValueBool())
	}

	if !model.AllowSpecialCharacters.IsNull() && !model.AllowSpecialCharacters.IsUnknown() {
		cfg.SetAllowSpecialCharacters(model.AllowSpecialCharacters.ValueBool())
	}

	return cfg, diags
}

// expandSourceIndices converts the source_index list into []suggestions.SourceIndex.
func expandSourceIndices(ctx context.Context, list types.List) ([]suggestions.SourceIndex, diag.Diagnostics) {
	var diags diag.Diagnostics

	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}

	var models []SourceIndexModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}

	result := make([]suggestions.SourceIndex, 0, len(models))
	for _, m := range models {
		si := suggestions.NewSourceIndex(m.IndexName.ValueString())

		if !m.Replicas.IsNull() && !m.Replicas.IsUnknown() {
			si.SetReplicas(m.Replicas.ValueBool())
		}

		if !m.AnalyticsTags.IsNull() && !m.AnalyticsTags.IsUnknown() {
			var tags []string
			diags.Append(m.AnalyticsTags.ElementsAs(ctx, &tags, false)...)
			if diags.HasError() {
				return nil, diags
			}
			si.SetAnalyticsTags(tags)
		}

		if !m.MinHits.IsNull() && !m.MinHits.IsUnknown() {
			si.SetMinHits(int32(m.MinHits.ValueInt64()))
		}

		if !m.MinLetters.IsNull() && !m.MinLetters.IsUnknown() {
			si.SetMinLetters(int32(m.MinLetters.ValueInt64()))
		}

		if !m.Generate.IsNull() && !m.Generate.IsUnknown() {
			var generate [][]string
			if err := json.Unmarshal([]byte(m.Generate.ValueString()), &generate); err != nil {
				diags.AddError("Invalid generate JSON", fmt.Sprintf("Could not parse generate field as [][]string: %s", err))
				return nil, diags
			}
			si.SetGenerate(generate)
		}

		if !m.External.IsNull() && !m.External.IsUnknown() {
			var external []string
			diags.Append(m.External.ElementsAs(ctx, &external, false)...)
			if diags.HasError() {
				return nil, diags
			}
			si.SetExternal(external)
		}

		if !m.Facets.IsNull() && !m.Facets.IsUnknown() {
			facets, d := expandFacets(ctx, m.Facets)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			si.SetFacets(facets)
		}

		result = append(result, *si)
	}

	return result, diags
}

// expandFacets converts a types.List of FacetModel into []suggestions.Facet.
func expandFacets(ctx context.Context, list types.List) ([]suggestions.Facet, diag.Diagnostics) {
	var diags diag.Diagnostics

	var models []FacetModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}

	result := make([]suggestions.Facet, 0, len(models))
	for _, m := range models {
		f := suggestions.NewFacet()
		if !m.Attribute.IsNull() && !m.Attribute.IsUnknown() {
			f.SetAttribute(m.Attribute.ValueString())
		}
		if !m.Amount.IsNull() && !m.Amount.IsUnknown() {
			f.SetAmount(int32(m.Amount.ValueInt64()))
		}
		result = append(result, *f)
	}

	return result, diags
}

// expandConfiguration converts the Terraform model into a Configuration for update calls.
// Unlike ConfigurationWithIndex, Configuration does not include IndexName (it is a path parameter).
func expandConfiguration(ctx context.Context, model *QuerySuggestionsConfigResourceModel) (*suggestions.Configuration, diag.Diagnostics) {
	var diags diag.Diagnostics

	sourceIndices, d := expandSourceIndices(ctx, model.SourceIndices)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	cfg := suggestions.NewConfiguration(sourceIndices)

	if langs, d := expandLanguages(ctx, model); d.HasError() {
		diags.Append(d...)
		return nil, diags
	} else if langs != nil {
		cfg.SetLanguages(langs)
	}

	if !model.Exclude.IsNull() && !model.Exclude.IsUnknown() {
		var exclude []string
		diags.Append(model.Exclude.ElementsAs(ctx, &exclude, false)...)
		if diags.HasError() {
			return nil, diags
		}
		cfg.SetExclude(exclude)
	}

	if !model.EnablePersonalization.IsNull() && !model.EnablePersonalization.IsUnknown() {
		cfg.SetEnablePersonalization(model.EnablePersonalization.ValueBool())
	}

	if !model.AllowSpecialCharacters.IsNull() && !model.AllowSpecialCharacters.IsUnknown() {
		cfg.SetAllowSpecialCharacters(model.AllowSpecialCharacters.ValueBool())
	}

	return cfg, diags
}

// expandLanguages builds a *suggestions.Languages from the split bool/list model fields.
// Returns nil if neither field is set.
func expandLanguages(ctx context.Context, model *QuerySuggestionsConfigResourceModel) (*suggestions.Languages, diag.Diagnostics) {
	var diags diag.Diagnostics

	if !model.Languages.IsNull() && !model.Languages.IsUnknown() {
		var langs []string
		diags.Append(model.Languages.ElementsAs(ctx, &langs, false)...)
		if diags.HasError() {
			return nil, diags
		}
		return suggestions.ArrayOfStringAsLanguages(langs), diags
	}

	if !model.LanguagesEnabled.IsNull() && !model.LanguagesEnabled.IsUnknown() {
		return suggestions.BoolAsLanguages(model.LanguagesEnabled.ValueBool()), diags
	}

	return nil, diags
}
