package agent_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"testing"

	"github.com/algolia/terraform-provider-algolia/internal/provider"
	"github.com/algolia/terraform-provider-algolia/internal/services/agent"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"algolia": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func testAccCheckAgentDestroy(s *terraform.State) error {
	client := agent.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_agent" {
			continue
		}
		_, err := client.GetAgent(context.Background(), rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("agent %s still exists", rs.Primary.ID)
		}

		var apiErr *agent.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			continue
		}

		return fmt.Errorf("checking agent %s destroy state: %w", rs.Primary.ID, err)
	}
	return nil
}

func TestAccAgentResource_basic(t *testing.T) {
	agentName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAgentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentResourceConfig_basic(agentName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_agent.test", "name", agentName),
					resource.TestCheckResourceAttr("algolia_agent.test", "instructions", "You are a helpful test agent."),
					resource.TestCheckResourceAttr("algolia_agent.test", "deletion_protection", "false"),
					resource.TestCheckResourceAttr("algolia_agent.test", "status", "draft"),
					resource.TestCheckResourceAttrSet("algolia_agent.test", "id"),
					resource.TestCheckResourceAttrSet("algolia_agent.test", "created_at"),
				),
			},
		},
	})
}

func TestAccAgentResource_publish(t *testing.T) {
	agentName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	openAIProvider := testAccOpenAIProvider(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAgentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentResourceConfig_published(agentName, openAIProvider.ID, openAIProvider.Model),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_agent.test", "name", agentName),
					resource.TestCheckResourceAttr("algolia_agent.test", "status", "published"),
					resource.TestCheckResourceAttr("algolia_agent.test", "publish", "true"),
				),
			},
		},
	})
}

func TestAccAgentResource_update(t *testing.T) {
	agentName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAgentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentResourceConfig_basic(agentName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_agent.test", "instructions", "You are a helpful test agent."),
				),
			},
			{
				Config: testAccAgentResourceConfig_updated(agentName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_agent.test", "name", agentName),
					resource.TestCheckResourceAttr("algolia_agent.test", "instructions", "You are an updated test agent."),
					resource.TestCheckResourceAttr("algolia_agent.test", "description", "Updated description"),
				),
			},
		},
	})
}

func TestAccAgentResource_withTools(t *testing.T) {
	agentName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAgentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentResourceConfig_withTools(agentName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_agent.test", "name", agentName),
					resource.TestCheckResourceAttr("algolia_agent.test", "tool_client_side.#", "1"),
					resource.TestCheckResourceAttr("algolia_agent.test", "tool_client_side.0.name", "get_order"),
					resource.TestCheckResourceAttr("algolia_agent.test", "tool_client_side.0.description", "Get order status"),
				),
			},
		},
	})
}

func TestAccAgentResource_deletionProtection(t *testing.T) {
	agentName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentResourceConfig_deletionProtection(agentName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_agent.test", "deletion_protection", "true"),
				),
			},
			{
				Config:      testAccAgentResourceConfig_deletionProtection(agentName),
				Destroy:     true,
				ExpectError: regexp.MustCompile("Deletion Protection Enabled"),
			},
			{
				// Disable deletion protection so the test can clean up.
				Config: testAccAgentResourceConfig_basic(agentName),
			},
		},
	})
}

func TestAccAgentResource_import(t *testing.T) {
	agentName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAgentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentResourceConfig_basic(agentName),
			},
			{
				ResourceName:            "algolia_agent.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"deletion_protection"},
			},
			{
				Config: testAccAgentResourceConfig_basic(agentName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_agent.test", "publish", "false"),
					resource.TestCheckResourceAttr("algolia_agent.test", "deletion_protection", "false"),
				),
			},
		},
	})
}

func TestAccAgentResource_importPublished(t *testing.T) {
	agentName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	openAIProvider := testAccOpenAIProvider(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAgentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentResourceConfig_published(agentName, openAIProvider.ID, openAIProvider.Model),
			},
			{
				ResourceName:            "algolia_agent.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"deletion_protection"},
			},
			{
				Config: testAccAgentResourceConfig_published(agentName, openAIProvider.ID, openAIProvider.Model),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_agent.test", "publish", "true"),
					resource.TestCheckResourceAttr("algolia_agent.test", "status", "published"),
				),
			},
		},
	})
}

func TestAccAgentResource_publishCannotUnpublish(t *testing.T) {
	agentName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	openAIProvider := testAccOpenAIProvider(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAgentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentResourceConfig_published(agentName, openAIProvider.ID, openAIProvider.Model),
			},
			{
				Config:      testAccAgentResourceConfig_unpublished(agentName, openAIProvider.ID, openAIProvider.Model),
				ExpectError: regexp.MustCompile(`(?i)unpublish.*not supported`),
			},
			{
				Config: testAccAgentResourceConfig_published(agentName, openAIProvider.ID, openAIProvider.Model),
			},
		},
	})
}

type testAccProviderDetails struct {
	ID    string
	Model string
}

func testAccOpenAIProvider(t *testing.T) testAccProviderDetails {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	if os.Getenv("ALGOLIA_APP_ID") == "" || os.Getenv("ALGOLIA_API_KEY") == "" {
		t.Skip("ALGOLIA_APP_ID and ALGOLIA_API_KEY must be set for acceptance tests")
	}

	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		t.Skip("OPENAI_API_KEY must be set to test published agents")
	}

	client := &http.Client{}
	createResp := struct {
		ID   string `json:"id"`
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}{}

	testAccAgentStudioRequest(t, client, http.MethodPost, "/providers", map[string]any{
		"name":         fmt.Sprintf("tf-test-openai-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)),
		"providerName": "openai",
		"input": map[string]any{
			"apiKey": openAIKey,
		},
	}, &createResp)

	providerID := firstNonEmpty(createResp.ID, createResp.Data.ID)
	if providerID == "" {
		t.Fatal("provider creation response did not include an id")
	}

	t.Cleanup(func() {
		testAccAgentStudioDelete(t, client, "/providers/"+providerID)
	})

	model := testAccProviderModel(t, client, providerID)

	return testAccProviderDetails{
		ID:    providerID,
		Model: model,
	}
}

func testAccProviderModel(t *testing.T, client *http.Client, providerID string) string {
	t.Helper()

	var payload json.RawMessage
	testAccAgentStudioRequest(t, client, http.MethodGet, "/providers/"+providerID+"/models", nil, &payload)

	var models []string
	if err := json.Unmarshal(payload, &models); err == nil && len(models) > 0 {
		return preferredModel(models)
	}

	var wrappedPayload map[string]any
	if err := json.Unmarshal(payload, &wrappedPayload); err != nil {
		t.Fatalf("provider %s models response was not a supported JSON shape: %v", providerID, err)
	}

	rawModels, _ := wrappedPayload["data"].([]any)
	if len(rawModels) == 0 {
		t.Fatalf("provider %s did not return any models", providerID)
	}

	models = make([]string, 0, len(rawModels))
	for _, rawModel := range rawModels {
		modelMap, ok := rawModel.(map[string]any)
		if !ok {
			continue
		}

		model := firstNonEmpty(
			stringValue(modelMap["id"]),
			stringValue(modelMap["name"]),
			stringValue(modelMap["slug"]),
			stringValue(modelMap["model"]),
		)
		if model != "" {
			models = append(models, model)
		}
	}

	if len(models) == 0 {
		t.Fatalf("provider %s models response did not contain a usable model identifier", providerID)
	}

	return preferredModel(models)
}

func preferredModel(models []string) string {
	for _, preferred := range []string{"gpt-4.1-mini", "gpt-4.1", "gpt-4", "gpt-3.5-turbo"} {
		for _, model := range models {
			if model == preferred {
				return model
			}
		}
	}

	return models[0]
}

func testAccAgentStudioDelete(t *testing.T, client *http.Client, apiPath string) {
	t.Helper()

	req, err := testAccAgentStudioHTTPRequest(http.MethodDelete, apiPath, nil)
	if err != nil {
		t.Errorf("creating cleanup request for %s: %v", apiPath, err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("cleanup request for %s failed: %v", apiPath, err)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("cleanup request for %s failed with status %d: %s", apiPath, resp.StatusCode, string(body))
	}
}

func testAccAgentStudioRequest(t *testing.T, client *http.Client, method, apiPath string, body any, result any) {
	t.Helper()

	req, err := testAccAgentStudioHTTPRequest(method, apiPath, body)
	if err != nil {
		t.Fatalf("creating request for %s %s: %v", method, apiPath, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("executing request for %s %s: %v", method, apiPath, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body for %s %s: %v", method, apiPath, err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		t.Fatalf("request %s %s failed with status %d: %s", method, apiPath, resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			t.Fatalf("unmarshalling response for %s %s: %v\nbody: %s", method, apiPath, err, string(respBody))
		}
	}
}

func testAccAgentStudioHTTPRequest(method, apiPath string, body any) (*http.Request, error) {
	appID := os.Getenv("ALGOLIA_APP_ID")
	apiKey := os.Getenv("ALGOLIA_API_KEY")
	url := fmt.Sprintf("https://%s.algolia.net/agent-studio/1%s", appID, apiPath)

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, url, reader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Algolia-API-Key", apiKey)
	req.Header.Set("X-Algolia-Application-Id", appID)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// --- Config helpers ---

func testAccAgentResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "algolia_agent" "test" {
  name                = %[1]q
  instructions        = "You are a helpful test agent."
  deletion_protection = false
}
`, name)
}

func testAccAgentResourceConfig_published(name, providerID, model string) string {
	return fmt.Sprintf(`
resource "algolia_agent" "test" {
  name                = %[1]q
  instructions        = "You are a helpful test agent."
  provider_id         = %[2]q
  model               = %[3]q
  publish             = true
  deletion_protection = false
}
`, name, providerID, model)
}

func testAccAgentResourceConfig_unpublished(name, providerID, model string) string {
	return fmt.Sprintf(`
resource "algolia_agent" "test" {
  name                = %[1]q
  instructions        = "You are a helpful test agent."
  provider_id         = %[2]q
  model               = %[3]q
  publish             = false
  deletion_protection = false
}
`, name, providerID, model)
}

func testAccAgentResourceConfig_updated(name string) string {
	return fmt.Sprintf(`
resource "algolia_agent" "test" {
  name                = %[1]q
  instructions        = "You are an updated test agent."
  description         = "Updated description"
  deletion_protection = false
}
`, name)
}

func testAccAgentResourceConfig_withTools(name string) string {
	return fmt.Sprintf(`
resource "algolia_agent" "test" {
  name                = %[1]q
  instructions        = "Use the tools."
  deletion_protection = false

  tool_client_side {
    name        = "get_order"
    description = "Get order status"
    input_schema = jsonencode({
      type = "object"
      properties = {
        order_id = { type = "string" }
      }
      required = ["order_id"]
    })
  }
}
`, name)
}

func testAccAgentResourceConfig_deletionProtection(name string) string {
	return fmt.Sprintf(`
resource "algolia_agent" "test" {
  name                = %[1]q
  instructions        = "You are a helpful test agent."
  deletion_protection = true
}
`, name)
}
