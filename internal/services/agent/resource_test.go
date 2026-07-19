package agent_test

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"

	agentStudio "github.com/algolia/algoliasearch-client-go/v4/algolia/agent-studio"
	"github.com/algolia/terraform-provider-algolia/internal/provider"
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
	client, err := agentStudio.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		return fmt.Errorf("creating agent studio client: %w", err)
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_agent" {
			continue
		}
		_, err := client.GetAgent(client.NewApiGetAgentRequest(rs.Primary.ID))
		if err == nil {
			return fmt.Errorf("agent %s still exists", rs.Primary.ID)
		}

		var apiErr *agentStudio.APIError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
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
	testAccRequireOpenAIKey(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAgentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentResourceConfig_published(agentName),
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
	testAccRequireOpenAIKey(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAgentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentResourceConfig_published(agentName),
			},
			{
				ResourceName:            "algolia_agent.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"deletion_protection"},
			},
			{
				Config: testAccAgentResourceConfig_published(agentName),
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
	testAccRequireOpenAIKey(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAgentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentResourceConfig_published(agentName),
			},
			{
				Config:      testAccAgentResourceConfig_unpublished(agentName),
				ExpectError: regexp.MustCompile(`(?i)unpublish.*not supported`),
			},
			{
				Config: testAccAgentResourceConfig_published(agentName),
			},
		},
	})
}

func TestAccAgentResource_updatePublished(t *testing.T) {
	agentName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccRequireOpenAIKey(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAgentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentResourceConfig_published(agentName),
			},
			{
				Config: testAccAgentResourceConfig_publishedUpdated(agentName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_agent.test", "description", "Updated while still published"),
					resource.TestCheckResourceAttr("algolia_agent.test", "status", "published"),
					resource.TestCheckResourceAttr("algolia_agent.test", "publish", "true"),
				),
			},
		},
	})
}

func testAccRequireOpenAIKey(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	if os.Getenv("ALGOLIA_APP_ID") == "" || os.Getenv("ALGOLIA_API_KEY") == "" {
		t.Skip("ALGOLIA_APP_ID and ALGOLIA_API_KEY must be set for acceptance tests")
	}

	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY must be set to test published agents")
	}
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

func testAccAgentResourceConfig_published(name string) string {
	return fmt.Sprintf(`
resource "algolia_agent_provider" "test" {
  name          = "tf-provider-%[1]s"
  provider_name = "openai"

  openai {
    api_key = %[2]q
  }
}

resource "algolia_agent" "test" {
  name                = %[3]q
  instructions        = "You are a helpful test agent."
  provider_id         = algolia_agent_provider.test.id
  model               = "gpt-4.1-mini"
  publish             = true
  deletion_protection = false
}
`, name, os.Getenv("OPENAI_API_KEY"), name)
}

func testAccAgentResourceConfig_unpublished(name string) string {
	return fmt.Sprintf(`
resource "algolia_agent_provider" "test" {
  name          = "tf-provider-%[1]s"
  provider_name = "openai"

  openai {
    api_key = %[2]q
  }
}

resource "algolia_agent" "test" {
  name                = %[3]q
  instructions        = "You are a helpful test agent."
  provider_id         = algolia_agent_provider.test.id
  model               = "gpt-4.1-mini"
  publish             = false
  deletion_protection = false
}
`, name, os.Getenv("OPENAI_API_KEY"), name)
}

func testAccAgentResourceConfig_publishedUpdated(name string) string {
	return fmt.Sprintf(`
resource "algolia_agent_provider" "test" {
  name          = "tf-provider-%[1]s"
  provider_name = "openai"

  openai {
    api_key = %[2]q
  }
}

resource "algolia_agent" "test" {
  name                = %[3]q
  description         = "Updated while still published"
  instructions        = "You are a helpful test agent."
  provider_id         = algolia_agent_provider.test.id
  model               = "gpt-4.1-mini"
  publish             = true
  deletion_protection = false
}
`, name, os.Getenv("OPENAI_API_KEY"), name)
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
