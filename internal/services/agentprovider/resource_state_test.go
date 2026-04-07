package agentprovider

import (
	"context"
	"testing"

	agentstudio "github.com/algolia/terraform-provider-algolia/internal/services/agent"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestHydrateAgentProviderResourceState_PreservesSensitiveFields(t *testing.T) {
	ctx := context.Background()
	model := &AgentProviderResourceModel{}
	preserved := AgentProviderResourceModel{
		OpenAI: types.ObjectValueMust(providerBlockAttrTypes(mustProviderSpec("openai")), map[string]attr.Value{
			"api_key":  types.StringValue("super-secret"),
			"base_url": types.StringValue("https://api.openai.com/v1"),
		}),
	}

	resp := &agentstudio.ProviderResponse{
		ID:           "provider-123",
		Name:         "OpenAI Prod",
		ProviderName: "openai",
		Input: map[string]any{
			"apiKey":  "aV6IA",
			"baseUrl": "https://proxy.example.com/v1",
		},
		CreatedAt: "2026-01-01T00:00:00Z",
		UpdatedAt: "2026-01-02T00:00:00Z",
	}

	diags := hydrateAgentProviderResourceState(ctx, resp, preserved, model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags.Errors())
	}

	block := model.OpenAI.Attributes()
	if got := block["api_key"].(types.String).ValueString(); got != "super-secret" {
		t.Fatalf("expected preserved api_key, got %q", got)
	}

	if got := block["base_url"].(types.String).ValueString(); got != "https://proxy.example.com/v1" {
		t.Fatalf("expected remote base_url, got %q", got)
	}
}

func TestHydrateImportedAgentProviderResourceState_LeavesSensitiveFieldsNull(t *testing.T) {
	ctx := context.Background()
	model := &AgentProviderResourceModel{}

	resp := &agentstudio.ProviderResponse{
		ID:           "provider-456",
		Name:         "Gemini",
		ProviderName: "google_genai",
		Input: map[string]any{
			"apiKey": "mask",
		},
		CreatedAt: "2026-01-01T00:00:00Z",
		UpdatedAt: "2026-01-02T00:00:00Z",
	}

	diags := hydrateImportedAgentProviderResourceState(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags.Errors())
	}

	block := model.GoogleGenAI.Attributes()
	if !block["api_key"].(types.String).IsNull() {
		t.Fatal("expected imported api_key to be null")
	}
}

func TestValidateAgentProviderConfig_RequiresExactlyOneMatchingBlock(t *testing.T) {
	ctx := context.Background()
	diags := validateAgentProviderConfig(ctx, AgentProviderResourceModel{
		ProviderName: types.StringValue("openai"),
		OpenAI: types.ObjectValueMust(providerBlockAttrTypes(mustProviderSpec("openai")), map[string]attr.Value{
			"api_key":  types.StringValue("secret"),
			"base_url": types.StringValue("https://api.openai.com/v1"),
		}),
		Anthropic: types.ObjectValueMust(providerBlockAttrTypes(mustProviderSpec("anthropic")), map[string]attr.Value{
			"api_key":  types.StringValue("secret"),
			"base_url": types.StringNull(),
		}),
	})

	if !diags.HasError() {
		t.Fatal("expected multiple configured blocks to fail validation")
	}
}

func TestValidateAgentProviderConfig_RequiresMandatoryFields(t *testing.T) {
	ctx := context.Background()
	diags := validateAgentProviderConfig(ctx, AgentProviderResourceModel{
		ProviderName: types.StringValue("azure_openai"),
		AzureOpenAI: types.ObjectValueMust(providerBlockAttrTypes(mustProviderSpec("azure_openai")), map[string]attr.Value{
			"api_key":         types.StringValue("secret"),
			"base_url":        types.StringNull(),
			"deployment_name": types.StringValue("chat"),
			"api_version":     types.StringValue("2024-10-21"),
		}),
	})

	if !diags.HasError() {
		t.Fatal("expected missing azure_openai.base_url to fail validation")
	}
}

func TestAgentProviderResourceSchema_RegistersExpectedBlocks(t *testing.T) {
	schema := agentProviderResourceSchema()

	attr, ok := schema.Attributes["provider_name"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatal("expected provider_name to be a string attribute")
	}

	if !attr.Required {
		t.Fatal("expected provider_name to be required")
	}

	for _, blockName := range providerBlockNames() {
		if _, ok := schema.Blocks[blockName]; !ok {
			t.Fatalf("expected block %q to be registered", blockName)
		}
	}
}

func TestAgentProviderModelsDataSourceSchema(t *testing.T) {
	schema := agentProviderModelsDataSourceSchema()

	providerIDAttr, ok := schema.Attributes["provider_id"].(datasourceschema.StringAttribute)
	if !ok || !providerIDAttr.Required {
		t.Fatal("expected provider_id to be a required string attribute")
	}

	modelsAttr, ok := schema.Attributes["models"].(datasourceschema.ListAttribute)
	if !ok || !modelsAttr.Computed {
		t.Fatal("expected models to be a computed list attribute")
	}
}

func mustProviderSpec(name string) providerSpec {
	spec, ok := providerSpecByName(name)
	if !ok {
		panic("missing provider spec: " + name)
	}

	return spec
}
