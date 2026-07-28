package agentprovider

import (
	"context"
	"encoding/json"
	"testing"

	agentStudio "github.com/algolia/algoliasearch-client-go/v4/algolia/agent-studio"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// agentStudio.ProviderInput is a oneOf with no discriminator field: its
// generated UnmarshalJSON tries every variant, never returns early, and no
// variant struct rejects unknown or missing keys, so decoding an API response
// leaves several pointers non-nil at once. MarshalJSON then serializes
// whichever pointer comes first in its fixed order - AnthropicProviderInput -
// which declares only apiKey and baseUrl. Reading an azure_openai provider
// back therefore used to lose azureEndpoint, azureDeployment and apiVersion,
// leaving three Required attributes null in state.
//
// These tests decode the API payload the way the HTTP layer does, via
// json.Unmarshal into the union, rather than through the client's
// ...AsProviderInput helpers, which set exactly one pointer and so never
// reproduce the defect.

func TestHydrateAgentProviderResourceState_KeepsAllRemoteFields(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		apiJSON      string
		want         map[string]string
	}{
		{
			name:         "openai",
			providerName: "openai",
			apiJSON:      `{"apiKey":"masked","baseUrl":"https://proxy.example.com/v1"}`,
			want:         map[string]string{"base_url": "https://proxy.example.com/v1"},
		},
		{
			name:         "azure_openai",
			providerName: "azure_openai",
			apiJSON:      `{"apiKey":"masked","azureEndpoint":"https://example.openai.azure.com","azureDeployment":"gpt-4o","apiVersion":"2024-10-21"}`,
			want: map[string]string{
				"base_url":        "https://example.openai.azure.com",
				"deployment_name": "gpt-4o",
				"api_version":     "2024-10-21",
			},
		},
		{
			name:         "google_genai",
			providerName: "google_genai",
			apiJSON:      `{"apiKey":"masked"}`,
			want:         map[string]string{},
		},
		{
			name:         "anthropic",
			providerName: "anthropic",
			apiJSON:      `{"apiKey":"masked","baseUrl":"https://anthropic.example.com"}`,
			want:         map[string]string{"base_url": "https://anthropic.example.com"},
		},
		{
			name:         "openai_compatible",
			providerName: "openai_compatible",
			apiJSON:      `{"apiKey":"masked","baseUrl":"https://llm.example.com/v1","defaultModel":"llama-3.1-70b"}`,
			want:         map[string]string{"base_url": "https://llm.example.com/v1"},
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &agentStudio.ProviderAuthenticationResponse{
				Id:           "provider-1",
				Name:         "provider " + tt.name,
				ProviderName: tt.providerName,
				Input:        decodeProviderInput(t, tt.apiJSON),
				CreatedAt:    "2026-01-01T00:00:00Z",
				UpdatedAt:    "2026-01-02T00:00:00Z",
			}

			model := &AgentProviderResourceModel{}
			diags := hydrateAgentProviderResourceState(ctx, resp, AgentProviderResourceModel{}, model)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags.Errors())
			}

			block := providerBlockValue(*model, mustProviderSpec(tt.providerName))
			if block.IsNull() || block.IsUnknown() {
				t.Fatalf("expected %s block to be populated, got %#v", tt.providerName, block)
			}

			for name, want := range tt.want {
				attrValue, ok := block.Attributes()[name]
				if !ok {
					t.Fatalf("expected attribute %q on the %s block", name, tt.providerName)
				}

				got := attrValue.(types.String)
				if got.ValueString() != want {
					t.Fatalf("%s.%s = %q, want %q", tt.providerName, name, got.ValueString(), want)
				}
			}
		})
	}
}

func TestHydrateAgentProviderDataSourceState_KeepsAllRemoteFields(t *testing.T) {
	ctx := context.Background()
	model := &AgentProviderDataSourceModel{}

	resp := &agentStudio.ProviderAuthenticationResponse{
		Id:           "provider-2",
		Name:         "Azure DS",
		ProviderName: "azure_openai",
		Input:        decodeProviderInput(t, `{"apiKey":"masked","azureEndpoint":"https://example.openai.azure.com","azureDeployment":"gpt-4o","apiVersion":"2024-10-21"}`),
		CreatedAt:    "2026-01-01T00:00:00Z",
		UpdatedAt:    "2026-01-02T00:00:00Z",
	}

	diags := hydrateAgentProviderDataSourceState(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags.Errors())
	}

	block := model.AzureOpenAI.Attributes()
	for name, want := range map[string]string{
		"base_url":        "https://example.openai.azure.com",
		"deployment_name": "gpt-4o",
		"api_version":     "2024-10-21",
	} {
		attrValue, ok := block[name]
		if !ok {
			t.Fatalf("expected attribute %q on the azure_openai data source block", name)
		}

		if got := attrValue.(types.String).ValueString(); got != want {
			t.Fatalf("azure_openai.%s = %q, want %q", name, got, want)
		}
	}
}

func TestHydrateImportedAgentProviderResourceState_KeepsAllRemoteFields(t *testing.T) {
	ctx := context.Background()
	model := &AgentProviderResourceModel{}

	resp := &agentStudio.ProviderAuthenticationResponse{
		Id:           "provider-3",
		Name:         "Azure Import",
		ProviderName: "azure_openai",
		Input:        decodeProviderInput(t, `{"apiKey":"masked","azureEndpoint":"https://example.openai.azure.com","azureDeployment":"gpt-4o","apiVersion":"2024-10-21"}`),
		CreatedAt:    "2026-01-01T00:00:00Z",
		UpdatedAt:    "2026-01-02T00:00:00Z",
	}

	diags := hydrateImportedAgentProviderResourceState(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags.Errors())
	}

	block := model.AzureOpenAI.Attributes()
	if got := block["deployment_name"].(types.String).ValueString(); got != "gpt-4o" {
		t.Fatalf("azure_openai.deployment_name = %q, want gpt-4o", got)
	}
	if got := block["api_version"].(types.String).ValueString(); got != "2024-10-21" {
		t.Fatalf("azure_openai.api_version = %q, want 2024-10-21", got)
	}
}

// TestProviderRequestFromModel_MarshalsDeclaredVariant guards the expand
// direction: buildProviderInput selects the variant from provider_name, so the
// union it produces has exactly one pointer set and must marshal to the full
// payload rather than to AnthropicProviderInput's apiKey/baseUrl subset.
func TestProviderRequestFromModel_MarshalsDeclaredVariant(t *testing.T) {
	model := AgentProviderResourceModel{
		Name:         types.StringValue("Azure Prod"),
		ProviderName: types.StringValue("azure_openai"),
		AzureOpenAI: types.ObjectValueMust(providerBlockAttrTypes(mustProviderSpec("azure_openai")), map[string]attr.Value{
			"api_key":         types.StringValue("secret"),
			"base_url":        types.StringValue("https://example.openai.azure.com"),
			"deployment_name": types.StringValue("gpt-4o"),
			"api_version":     types.StringValue("2024-10-21"),
		}),
	}

	apiReq, diags := providerRequestFromModel(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags.Errors())
	}

	encoded, err := json.Marshal(apiReq.Input)
	if err != nil {
		t.Fatalf("marshaling ProviderInput: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("decoding marshaled ProviderInput: %v", err)
	}

	for key, want := range map[string]string{
		"apiKey":          "secret",
		"azureEndpoint":   "https://example.openai.azure.com",
		"azureDeployment": "gpt-4o",
		"apiVersion":      "2024-10-21",
	} {
		if got[key] != want {
			t.Fatalf("ProviderInput[%q] = %v, want %q (full payload: %s)", key, got[key], want, encoded)
		}
	}
}

func decodeProviderInput(t *testing.T, raw string) agentStudio.ProviderInput {
	t.Helper()

	var input agentStudio.ProviderInput
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		t.Fatalf("decoding %s into ProviderInput: %v", raw, err)
	}

	return input
}
