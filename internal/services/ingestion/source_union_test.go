package ingestion

import (
	"encoding/json"
	"strings"
	"testing"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SourceInput and SourceUpdateInput have the same missing-discriminator defect
// as AuthInput (see authentication_union_test.go): SourceJSON and SourceCSV are
// decoded unconditionally and always "succeed", and SourceCSV is reached early
// in MarshalJSON's fixed order, so a docker/shopify/commercetools source used to
// go on the wire as {"url":""}.

// sourceInputCase pairs a source type with a representative `input` payload for
// it. The same table drives the expand (Terraform -> API) and flatten
// (API -> Terraform) directions.
type sourceInputCase struct {
	name       string
	sourceType ingestionapi.SourceType
	input      string
	// updateInput is the payload the update endpoint accepts, which for some
	// types is a strict subset of input: the Ingestion API has no update shape
	// for a source's immutable fields (docker `image`, shopify `shopURL`,
	// commercetools `projectKey`). Empty means "same as input".
	updateInput string
	// noUpdateVariant marks a type the client offers no SourceUpdateInput
	// variant for at all.
	noUpdateVariant bool
}

func sourceInputCases() []sourceInputCase {
	return []sourceInputCase{
		{
			name:       "algoliaIndex",
			sourceType: ingestionapi.SOURCE_TYPE_ALGOLIA_INDEX,
			input:      `{"indexName":"products","filters":"price > 10"}`,
		},
		{
			name:            "bigcommerce",
			sourceType:      ingestionapi.SOURCE_TYPE_BIGCOMMERCE,
			input:           `{"storeHash":"abc123","customFields":["color","size"]}`,
			noUpdateVariant: true,
		},
		{
			name:       "bigquery",
			sourceType: ingestionapi.SOURCE_TYPE_BIGQUERY,
			input:      `{"projectID":"my-project","datasetID":"my-dataset","table":"products"}`,
		},
		{
			name:        "commercetools",
			sourceType:  ingestionapi.SOURCE_TYPE_COMMERCETOOLS,
			input:       `{"url":"https://api.example.commercetools.com","projectKey":"my-project","locales":["en-GB"]}`,
			updateInput: `{"url":"https://api.example.commercetools.com","locales":["en-GB"]}`,
		},
		{
			name:       "csv",
			sourceType: ingestionapi.SOURCE_TYPE_CSV,
			input:      `{"url":"https://example.com/products.csv","uniqueIDColumn":"id","delimiter":";"}`,
		},
		{
			name:        "docker",
			sourceType:  ingestionapi.SOURCE_TYPE_DOCKER,
			input:       `{"image":"algolia/connector","configuration":{"apiKey":"k","entity":"product"}}`,
			updateInput: `{"configuration":{"apiKey":"k","entity":"product"}}`,
		},
		{
			name:       "ga4BigqueryExport",
			sourceType: ingestionapi.SOURCE_TYPE_GA4_BIGQUERY_EXPORT,
			input:      `{"projectID":"my-project","datasetID":"analytics_123","tablePrefix":"events_"}`,
		},
		{
			name:       "json",
			sourceType: ingestionapi.SOURCE_TYPE_JSON,
			input:      `{"url":"https://example.com/products.json","uniqueIDColumn":"objectID"}`,
		},
		{
			name:        "shopify",
			sourceType:  ingestionapi.SOURCE_TYPE_SHOPIFY,
			input:       `{"shopURL":"https://store.myshopify.com","featureFlags":{"metafields":true}}`,
			updateInput: `{"featureFlags":{"metafields":true}}`,
		},
	}
}

func TestExpandSourceCreate_MarshalsDeclaredVariant(t *testing.T) {
	for _, tt := range sourceInputCases() {
		t.Run(tt.name, func(t *testing.T) {
			model := &SourceResourceModel{
				Type:  types.StringValue(string(tt.sourceType)),
				Name:  types.StringValue("source-" + tt.name),
				Input: types.StringValue(tt.input),
			}

			create, diags := expandSourceCreate(model)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags.Errors())
			}

			encoded, err := json.Marshal(create.Input)
			if err != nil {
				t.Fatalf("marshaling SourceInput: %v", err)
			}

			if !jsonSemanticallyEqual(string(encoded), tt.input) {
				t.Fatalf("SourceInput marshaled to %s, want %s", encoded, tt.input)
			}
		})
	}
}

func TestExpandSourceUpdate_MarshalsDeclaredVariant(t *testing.T) {
	for _, tt := range sourceInputCases() {
		t.Run(tt.name, func(t *testing.T) {
			model := &SourceResourceModel{
				Type:  types.StringValue(string(tt.sourceType)),
				Name:  types.StringValue("source-" + tt.name),
				Input: types.StringValue(tt.input),
			}

			update, diags := expandSourceUpdate(model, types.StringNull())

			if tt.noUpdateVariant {
				if !diags.HasError() {
					t.Fatalf("expected a diagnostic: the client has no SourceUpdateInput variant for %q", tt.sourceType)
				}
				return
			}

			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags.Errors())
			}

			want := tt.updateInput
			if want == "" {
				want = tt.input
			}

			encoded, err := json.Marshal(update.Input)
			if err != nil {
				t.Fatalf("marshaling SourceUpdateInput: %v", err)
			}

			if !jsonSemanticallyEqual(string(encoded), want) {
				t.Fatalf("SourceUpdateInput marshaled to %s, want %s", encoded, want)
			}
		})
	}
}

// TestFlattenSource_ReadsDeclaredVariantFromAPIResponse decodes the API payload
// the way the HTTP layer does - straight into the union via json.Unmarshal -
// rather than through the client's ...AsSourceInput helpers, which set exactly
// one pointer and therefore hide the defect.
func TestFlattenSource_ReadsDeclaredVariantFromAPIResponse(t *testing.T) {
	for _, tt := range sourceInputCases() {
		t.Run(tt.name, func(t *testing.T) {
			source := &ingestionapi.Source{
				SourceID:  "source-1",
				Type:      tt.sourceType,
				Name:      "source-" + tt.name,
				Input:     decodeSourceInput(t, tt.input),
				CreatedAt: "2024-01-01T00:00:00Z",
				UpdatedAt: "2024-01-02T00:00:00Z",
			}

			var model SourceResourceModel
			diags := flattenSource(source, &model)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags.Errors())
			}

			if !jsonSemanticallyEqual(model.Input.ValueString(), tt.input) {
				t.Fatalf("flattened input = %s, want %s", model.Input.ValueString(), tt.input)
			}
		})
	}
}

func TestFlattenSourceDataSource_ReadsDeclaredVariantFromAPIResponse(t *testing.T) {
	for _, tt := range sourceInputCases() {
		t.Run(tt.name, func(t *testing.T) {
			source := &ingestionapi.Source{
				SourceID:  "source-1",
				Type:      tt.sourceType,
				Name:      "source-" + tt.name,
				Input:     decodeSourceInput(t, tt.input),
				CreatedAt: "2024-01-01T00:00:00Z",
				UpdatedAt: "2024-01-02T00:00:00Z",
			}

			var model SourceDataSourceModel
			diags := flattenSourceDataSource(source, &model)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags.Errors())
			}

			if !jsonSemanticallyEqual(model.Input.ValueString(), tt.input) {
				t.Fatalf("flattened input = %s, want %s", model.Input.ValueString(), tt.input)
			}
		})
	}
}

func TestExpandSourceInput_RejectsInputForAnotherType(t *testing.T) {
	model := &SourceResourceModel{
		Type:  types.StringValue(string(ingestionapi.SOURCE_TYPE_CSV)),
		Name:  types.StringValue("mismatched"),
		Input: types.StringValue(`{"shopURL":"https://store.myshopify.com"}`),
	}

	if _, diags := expandSourceCreate(model); !diags.HasError() {
		t.Fatal("expected a diagnostic for input that does not match the declared type")
	}
}

// A "push" source has no input shape at all, so configuring `input` for one is
// a configuration error rather than something to silently drop or coerce into
// another variant.
func TestExpandSourceInput_RejectsInputForPush(t *testing.T) {
	model := &SourceResourceModel{
		Type:  types.StringValue(string(ingestionapi.SOURCE_TYPE_PUSH)),
		Name:  types.StringValue("push-with-input"),
		Input: types.StringValue(`{"url":"https://example.com/data.csv"}`),
	}

	if _, diags := expandSourceCreate(model); !diags.HasError() {
		t.Fatal("expected a diagnostic for a push source configured with input")
	}

	if _, diags := expandSourceUpdate(model, types.StringNull()); !diags.HasError() {
		t.Fatal("expected a diagnostic for a push source configured with input on update")
	}
}

func decodeSourceInput(t *testing.T, raw string) *ingestionapi.SourceInput {
	t.Helper()

	var input ingestionapi.SourceInput
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		t.Fatalf("decoding %s into SourceInput: %v", raw, err)
	}

	return &input
}

// TestExpandSourceUpdate_UnchangedInputIsOmitted covers a regression found in
// review: bigcommerce has no SourceUpdateInput variant, so refusing on the mere
// presence of `input` blocked every update to such a source -- including changes
// to unrelated fields like name. Only an actual attempt to change the input of a
// source whose type has no update variant may fail.
func TestExpandSourceUpdate_UnchangedInputIsOmitted(t *testing.T) {
	const bigcommerceInput = `{"storeHash":"abc123","channel":"web"}`

	t.Run("bigcommerce rename succeeds when input is unchanged", func(t *testing.T) {
		model := &SourceResourceModel{
			Type:  types.StringValue(string(ingestionapi.SOURCE_TYPE_BIGCOMMERCE)),
			Name:  types.StringValue("renamed"),
			Input: types.StringValue(bigcommerceInput),
		}

		update, diags := expandSourceUpdate(model, types.StringValue(bigcommerceInput))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics renaming a bigcommerce source: %v", diags.Errors())
		}
		if update.Input != nil {
			t.Errorf("Input = %+v, want nil so the API keeps the existing input", update.Input)
		}
		if update.GetName() != "renamed" {
			t.Errorf("Name = %q, want %q", update.GetName(), "renamed")
		}
	})

	t.Run("bigcommerce input change still fails loudly", func(t *testing.T) {
		model := &SourceResourceModel{
			Type:  types.StringValue(string(ingestionapi.SOURCE_TYPE_BIGCOMMERCE)),
			Name:  types.StringValue("same"),
			Input: types.StringValue(`{"storeHash":"CHANGED","channel":"web"}`),
		}

		if _, diags := expandSourceUpdate(model, types.StringValue(bigcommerceInput)); !diags.HasError() {
			t.Error("changing a bigcommerce source's input should fail: the client has no update variant for it")
		}
	})

	t.Run("reformatted input is not treated as a change", func(t *testing.T) {
		model := &SourceResourceModel{
			Type:  types.StringValue(string(ingestionapi.SOURCE_TYPE_BIGCOMMERCE)),
			Name:  types.StringValue("same"),
			Input: types.StringValue(`{ "channel": "web",  "storeHash": "abc123" }`),
		}

		if _, diags := expandSourceUpdate(model, types.StringValue(bigcommerceInput)); diags.HasError() {
			t.Errorf("semantically equal input should not count as a change: %v", diags.Errors())
		}
	})

	t.Run("updatable type still sends changed input", func(t *testing.T) {
		model := &SourceResourceModel{
			Type:  types.StringValue(string(ingestionapi.SOURCE_TYPE_ALGOLIA_INDEX)),
			Name:  types.StringValue("same"),
			Input: types.StringValue(`{"indexName":"products","filters":"brand:new"}`),
		}

		update, diags := expandSourceUpdate(model, types.StringValue(`{"indexName":"products","filters":"brand:old"}`))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags.Errors())
		}
		if update.Input == nil {
			t.Fatal("Input = nil, want the changed input to be sent")
		}
		out, err := json.Marshal(update.Input)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if !strings.Contains(string(out), "brand:new") {
			t.Errorf("Input marshaled to %s, want it to carry the new filters", string(out))
		}
	})
}
