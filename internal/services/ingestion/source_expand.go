package ingestion

import (
	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// expandSourceCreate converts the Terraform plan into a SourceCreate
// request body for CreateSource.
func expandSourceCreate(model *SourceResourceModel) (*ingestionapi.SourceCreate, diag.Diagnostics) {
	var diags diag.Diagnostics

	input, inputDiags := expandSourceInput(model.Type.ValueString(), model.Input)
	diags.Append(inputDiags...)
	if diags.HasError() {
		return nil, diags
	}

	create := ingestionapi.NewSourceCreate(
		ingestionapi.SourceType(model.Type.ValueString()),
		model.Name.ValueString(),
	)
	create.Input = input
	create.AuthenticationID = model.AuthenticationID.ValueStringPointer()

	return create, diags
}

// expandSourceUpdate converts the Terraform plan into a SourceUpdate
// request body for UpdateSource.
//
// SourceUpdate takes a *SourceUpdateInput - a distinct union type from
// SourceCreate's *SourceInput (compare model_source_input.go and
// model_source_update_input.go in the Go client) - so `input` is decoded
// into a different Go type depending on whether we're creating or
// updating: expandSourceInput for Create, expandSourceUpdateInput for
// Update. There is no `type` field on SourceUpdate at all: the Ingestion
// API gives no way to change a source's type after creation, which is why
// `type` is RequiresReplace in the resource schema.
func expandSourceUpdate(model *SourceResourceModel) (*ingestionapi.SourceUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics

	input, inputDiags := expandSourceUpdateInput(model.Type.ValueString(), model.Input)
	diags.Append(inputDiags...)
	if diags.HasError() {
		return nil, diags
	}

	update := ingestionapi.NewSourceUpdate(ingestionapi.WithSourceUpdateName(model.Name.ValueString()))
	update.Input = input
	update.AuthenticationID = model.AuthenticationID.ValueStringPointer()

	return update, diags
}

// expandSourceInput decodes the `input` attribute into the SourceInput union
// expected by SourceCreate, selecting the variant from the declared source
// `type`.
//
// `input` is Optional: some source types (e.g. "push") need no configuration at
// all, so a null or empty value yields a nil *SourceInput, which SourceCreate's
// MarshalJSON simply omits from the request body.
//
// The variant cannot be inferred from the JSON alone. SourceInput is a
// generated oneOf with no discriminator field: SourceJSON and SourceCSV are
// decoded unconditionally and always "succeed" (nothing rejects unknown or
// missing keys), and MarshalJSON serializes the first non-nil pointer in its own
// fixed order, where SourceCSV comes early. A "docker" or "shopify" source
// therefore reached the API as {"url":""}. `type` is the discriminator the API
// itself uses, so it is what selects the variant here.
func expandSourceInput(sourceType string, input types.String) (*ingestionapi.SourceInput, diag.Diagnostics) {
	if input.IsNull() || input.IsUnknown() || input.ValueString() == "" {
		return nil, nil
	}

	raw := []byte(input.ValueString())

	switch ingestionapi.SourceType(sourceType) {
	case ingestionapi.SOURCE_TYPE_ALGOLIA_INDEX:
		var variant ingestionapi.SourceAlgoliaIndex
		if diags := decodeSourceInputVariant(raw, &variant, sourceType); diags.HasError() {
			return nil, diags
		}

		return ingestionapi.SourceAlgoliaIndexAsSourceInput(&variant), nil
	case ingestionapi.SOURCE_TYPE_BIGCOMMERCE:
		var variant ingestionapi.SourceBigCommerce
		if diags := decodeSourceInputVariant(raw, &variant, sourceType); diags.HasError() {
			return nil, diags
		}

		return ingestionapi.SourceBigCommerceAsSourceInput(&variant), nil
	case ingestionapi.SOURCE_TYPE_BIGQUERY:
		var variant ingestionapi.SourceBigQuery
		if diags := decodeSourceInputVariant(raw, &variant, sourceType); diags.HasError() {
			return nil, diags
		}

		return ingestionapi.SourceBigQueryAsSourceInput(&variant), nil
	case ingestionapi.SOURCE_TYPE_COMMERCETOOLS:
		var variant ingestionapi.SourceCommercetools
		if diags := decodeSourceInputVariant(raw, &variant, sourceType); diags.HasError() {
			return nil, diags
		}

		return ingestionapi.SourceCommercetoolsAsSourceInput(&variant), nil
	case ingestionapi.SOURCE_TYPE_CSV:
		var variant ingestionapi.SourceCSV
		if diags := decodeSourceInputVariant(raw, &variant, sourceType); diags.HasError() {
			return nil, diags
		}

		return ingestionapi.SourceCSVAsSourceInput(&variant), nil
	case ingestionapi.SOURCE_TYPE_DOCKER:
		var variant ingestionapi.SourceDocker
		if diags := decodeSourceInputVariant(raw, &variant, sourceType); diags.HasError() {
			return nil, diags
		}

		return ingestionapi.SourceDockerAsSourceInput(&variant), nil
	case ingestionapi.SOURCE_TYPE_GA4_BIGQUERY_EXPORT:
		var variant ingestionapi.SourceGA4BigQueryExport
		if diags := decodeSourceInputVariant(raw, &variant, sourceType); diags.HasError() {
			return nil, diags
		}

		return ingestionapi.SourceGA4BigQueryExportAsSourceInput(&variant), nil
	case ingestionapi.SOURCE_TYPE_JSON:
		var variant ingestionapi.SourceJSON
		if diags := decodeSourceInputVariant(raw, &variant, sourceType); diags.HasError() {
			return nil, diags
		}

		return ingestionapi.SourceJSONAsSourceInput(&variant), nil
	case ingestionapi.SOURCE_TYPE_SHOPIFY:
		var variant ingestionapi.SourceShopify
		if diags := decodeSourceInputVariant(raw, &variant, sourceType); diags.HasError() {
			return nil, diags
		}

		return ingestionapi.SourceShopifyAsSourceInput(&variant), nil
	default:
		return nil, noSourceInputShapeDiags(sourceType)
	}
}

// expandSourceUpdateInput is the SourceUpdate counterpart of expandSourceInput:
// the update endpoint accepts SourceUpdateInput, a distinct union whose variants
// omit the fields a source cannot change after creation (docker `image`,
// shopify `shopURL`, commercetools `projectKey`). Decoding is therefore lenient
// here - those keys legitimately appear in the configured `input` and are meant
// to be left out of the update request - whereas expandSourceInput is strict.
//
// The client offers no update variant at all for "bigcommerce", so that type is
// reported rather than silently coerced into another variant (it used to be sent
// as {"url":""}).
func expandSourceUpdateInput(sourceType string, input types.String) (*ingestionapi.SourceUpdateInput, diag.Diagnostics) {
	if input.IsNull() || input.IsUnknown() || input.ValueString() == "" {
		return nil, nil
	}

	raw := []byte(input.ValueString())

	switch ingestionapi.SourceType(sourceType) {
	case ingestionapi.SOURCE_TYPE_ALGOLIA_INDEX:
		var variant ingestionapi.SourceUpdateAlgoliaIndex
		if diags := decodeSourceUpdateInputVariant(raw, &variant, sourceType); diags.HasError() {
			return nil, diags
		}

		return ingestionapi.SourceUpdateAlgoliaIndexAsSourceUpdateInput(&variant), nil
	case ingestionapi.SOURCE_TYPE_BIGQUERY:
		var variant ingestionapi.SourceBigQuery
		if diags := decodeSourceUpdateInputVariant(raw, &variant, sourceType); diags.HasError() {
			return nil, diags
		}

		return ingestionapi.SourceBigQueryAsSourceUpdateInput(&variant), nil
	case ingestionapi.SOURCE_TYPE_COMMERCETOOLS:
		var variant ingestionapi.SourceUpdateCommercetools
		if diags := decodeSourceUpdateInputVariant(raw, &variant, sourceType); diags.HasError() {
			return nil, diags
		}

		return ingestionapi.SourceUpdateCommercetoolsAsSourceUpdateInput(&variant), nil
	case ingestionapi.SOURCE_TYPE_CSV:
		var variant ingestionapi.SourceCSV
		if diags := decodeSourceUpdateInputVariant(raw, &variant, sourceType); diags.HasError() {
			return nil, diags
		}

		return ingestionapi.SourceCSVAsSourceUpdateInput(&variant), nil
	case ingestionapi.SOURCE_TYPE_DOCKER:
		var variant ingestionapi.SourceUpdateDocker
		if diags := decodeSourceUpdateInputVariant(raw, &variant, sourceType); diags.HasError() {
			return nil, diags
		}

		return ingestionapi.SourceUpdateDockerAsSourceUpdateInput(&variant), nil
	case ingestionapi.SOURCE_TYPE_GA4_BIGQUERY_EXPORT:
		var variant ingestionapi.SourceGA4BigQueryExport
		if diags := decodeSourceUpdateInputVariant(raw, &variant, sourceType); diags.HasError() {
			return nil, diags
		}

		return ingestionapi.SourceGA4BigQueryExportAsSourceUpdateInput(&variant), nil
	case ingestionapi.SOURCE_TYPE_JSON:
		var variant ingestionapi.SourceJSON
		if diags := decodeSourceUpdateInputVariant(raw, &variant, sourceType); diags.HasError() {
			return nil, diags
		}

		return ingestionapi.SourceJSONAsSourceUpdateInput(&variant), nil
	case ingestionapi.SOURCE_TYPE_SHOPIFY:
		var variant ingestionapi.SourceUpdateShopify
		if diags := decodeSourceUpdateInputVariant(raw, &variant, sourceType); diags.HasError() {
			return nil, diags
		}

		return ingestionapi.SourceUpdateShopifyAsSourceUpdateInput(&variant), nil
	default:
		return nil, noSourceUpdateInputShapeDiags(sourceType)
	}
}

// decodeSourceInputVariant strictly decodes the `input` attribute into the
// configuration struct for sourceType, so configuration written for a different
// source type is reported instead of silently dropped.
func decodeSourceInputVariant(raw []byte, target any, sourceType string) diag.Diagnostics {
	var diags diag.Diagnostics

	if err := decodeJSONStrict(raw, target); err != nil {
		diags.AddError(
			"Invalid input JSON",
			"The `input` attribute must be JSON-encoded configuration matching the source type \""+sourceType+"\" "+
				"(e.g. jsonencode({ url = \"...\" }) for type \"csv\"). Failed to decode: "+err.Error(),
		)
	}

	return diags
}

// decodeSourceUpdateInputVariant decodes the `input` attribute into the update
// shape for sourceType. Unlike decodeSourceInputVariant it is lenient: the
// update shapes deliberately omit a source's immutable fields, which still
// appear in the configured `input`.
func decodeSourceUpdateInputVariant(raw []byte, target any, sourceType string) diag.Diagnostics {
	var diags diag.Diagnostics

	if err := decodeJSONLenient(raw, target); err != nil {
		diags.AddError(
			"Invalid input JSON",
			"The `input` attribute must be JSON-encoded configuration matching the source type \""+sourceType+"\". "+
				"Failed to decode: "+err.Error(),
		)
	}

	return diags
}

// noSourceInputShapeDiags reports a source type the provider cannot map to a
// configuration shape. "push" lands here whenever `input` is set: it accepts
// records pushed directly to it and has no input shape at all. So does a source
// type newly added to the Algolia client - the schema's allowed values are
// derived from that enum - before this file learns its configuration struct.
func noSourceInputShapeDiags(sourceType string) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.AddError(
		"Unexpected input for source type",
		"Source type \""+sourceType+"\" has no `input` shape the provider recognizes, so `input` must be omitted. "+
			"The provider will not guess a shape, since guessing would send another source type's configuration "+
			"to Algolia.",
	)

	return diags
}

// noSourceUpdateInputShapeDiags reports a source type whose `input` the
// Ingestion API cannot update in place. "bigcommerce" lands here: the client's
// SourceUpdateInput has no bigcommerce variant, so its `input` can only be
// changed by replacing the source.
func noSourceUpdateInputShapeDiags(sourceType string) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.AddError(
		"Source input cannot be updated",
		"The Ingestion API has no update shape for the `input` of a \""+sourceType+"\" source, so it cannot be "+
			"changed in place. Recreate the source (e.g. with `terraform taint`) to apply a new `input`.",
	)

	return diags
}
