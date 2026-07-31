package ingestion

import (
	"encoding/json"
	"reflect"

	"github.com/algolia/terraform-provider-algolia/internal/deletionprotection"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// flattenSource copies a GetSource response into the Terraform resource
// model.
//
// Unlike flattenAuthentication, this does refresh Input: GetSource returns
// a source's configuration in full - nothing is redacted. But naively
// overwriting model.Input with the API's JSON encoding on every
// Create/Read/Update would cause a perpetual diff whenever the API echoes
// back semantically identical JSON in a different form (key order, array
// order). So flattenSourceInput keeps the model's existing Input string
// as-is when it is semantically equal to what the API returned, and only
// adopts the API's encoding when it actually differs.
func flattenSource(source *ingestionapi.Source, model *SourceResourceModel) diag.Diagnostics {
	// Algolia does not store this flag, so it survives only by being carried through
	// every rebuild of the model. Resolving it here also seeds an import, which
	// arrives with no value at all.
	model.DeletionProtection = deletionprotection.Value(model.DeletionProtection)

	var diags diag.Diagnostics

	model.ID = types.StringValue(source.SourceID)
	model.SourceID = types.StringValue(source.SourceID)
	model.Type = types.StringValue(string(source.Type))
	model.Name = types.StringValue(source.Name)
	model.AuthenticationID = types.StringPointerValue(source.AuthenticationID)
	model.CreatedAt = types.StringValue(source.CreatedAt)
	model.UpdatedAt = types.StringValue(source.UpdatedAt)

	inputValue, inputDiags := flattenSourceInput(string(source.Type), source.Input, model.Input)
	diags.Append(inputDiags...)
	model.Input = inputValue

	return diags
}

// flattenSourceInput JSON-encodes the API's *SourceInput and decides
// whether to adopt it into state or keep the value already configured.
// previous is model.Input's value before this Create/Read/Update call -
// i.e. the plan's configured value (Create/Update) or the prior state
// (Read).
//
// The union is never encoded directly: see sourceInputVariant for why sourceType
// has to pick the variant first.
func flattenSourceInput(sourceType string, input *ingestionapi.SourceInput, previous types.String) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if input == nil {
		// The API returned no input at all (e.g. a "push" source, which has
		// no input shape). Only surface that as null if nothing was
		// configured either; otherwise keep whatever's already in
		// state/plan rather than clobbering a configured value with null.
		if previous.IsNull() || previous.IsUnknown() {
			return types.StringNull(), diags
		}

		return previous, diags
	}

	variant, variantDiags := sourceInputVariant(sourceType, input)
	diags.Append(variantDiags...)
	if diags.HasError() {
		return previous, diags
	}

	encoded, err := json.Marshal(variant)
	if err != nil {
		diags.AddError("Error encoding source input", "Could not JSON-encode the source's input: "+err.Error())
		return previous, diags
	}
	apiValue := string(encoded)

	if !previous.IsNull() && !previous.IsUnknown() && jsonSemanticallyEqual(previous.ValueString(), apiValue) {
		return previous, diags
	}

	return types.StringValue(apiValue), diags
}

// flattenSourceDataSource is the data source counterpart of flattenSource.
// The data source has no prior configuration to preserve, so it always
// surfaces the API's JSON encoding of input verbatim.
func flattenSourceDataSource(source *ingestionapi.Source, model *SourceDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(source.SourceID)
	model.SourceID = types.StringValue(source.SourceID)
	model.Type = types.StringValue(string(source.Type))
	model.Name = types.StringValue(source.Name)
	model.AuthenticationID = types.StringPointerValue(source.AuthenticationID)
	model.CreatedAt = types.StringValue(source.CreatedAt)
	model.UpdatedAt = types.StringValue(source.UpdatedAt)

	if source.Input == nil {
		model.Input = types.StringNull()
		return diags
	}

	variant, variantDiags := sourceInputVariant(string(source.Type), source.Input)
	diags.Append(variantDiags...)
	if diags.HasError() {
		return diags
	}

	encoded, err := json.Marshal(variant)
	if err != nil {
		diags.AddError("Error encoding source input", "Could not JSON-encode the source's input: "+err.Error())
		return diags
	}
	model.Input = types.StringValue(string(encoded))

	return diags
}

// sourceInputVariant returns the member of the SourceInput union that matches
// sourceType.
//
// Encoding the union itself is lossy. SourceInput is a generated oneOf with no
// discriminator field: SourceJSON and SourceCSV are decoded unconditionally and
// always "succeed", so a decoded API response has several pointers set at once,
// and MarshalJSON serializes the first non-nil one in its own fixed order - where
// SourceCSV comes early. Reading a "docker" or "shopify" source back therefore
// produced {"url":""} in state. The source's `type` is the discriminator the API
// itself uses, so it is what selects the variant here.
func sourceInputVariant(sourceType string, input *ingestionapi.SourceInput) (any, diag.Diagnostics) {
	var variant any

	switch ingestionapi.SourceType(sourceType) {
	case ingestionapi.SOURCE_TYPE_ALGOLIA_INDEX:
		variant = input.SourceAlgoliaIndex
	case ingestionapi.SOURCE_TYPE_BIGCOMMERCE:
		variant = input.SourceBigCommerce
	case ingestionapi.SOURCE_TYPE_BIGQUERY:
		variant = input.SourceBigQuery
	case ingestionapi.SOURCE_TYPE_COMMERCETOOLS:
		variant = input.SourceCommercetools
	case ingestionapi.SOURCE_TYPE_CSV:
		variant = input.SourceCSV
	case ingestionapi.SOURCE_TYPE_DOCKER:
		variant = input.SourceDocker
	case ingestionapi.SOURCE_TYPE_GA4_BIGQUERY_EXPORT:
		variant = input.SourceGA4BigQueryExport
	case ingestionapi.SOURCE_TYPE_JSON:
		variant = input.SourceJSON
	case ingestionapi.SOURCE_TYPE_SHOPIFY:
		variant = input.SourceShopify
	default:
		var diags diag.Diagnostics
		diags.AddError(
			"Unexpected input for source type",
			"Algolia returned an `input` for source type \""+sourceType+"\", which the provider has no "+
				"configuration shape for - a \"push\" source has none at all. The provider will not guess a "+
				"shape, since that would write another source type's configuration into state.",
		)

		return nil, diags
	}

	// A typed nil pointer means the response did not carry the keys the declared
	// type's shape requires. Reporting it beats encoding a different variant and
	// writing another source type's configuration into state.
	if reflect.ValueOf(variant).IsNil() {
		var diags diag.Diagnostics
		diags.AddError(
			"Unexpected input for source type",
			"Algolia returned an `input` that does not match source type \""+sourceType+"\". The provider will "+
				"not fall back to another shape, since that would write the wrong configuration into state.",
		)

		return nil, diags
	}

	return variant, nil
}
