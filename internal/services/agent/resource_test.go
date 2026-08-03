package agent_test

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"

	agentStudio "github.com/algolia/algoliasearch-client-go/v4/algolia/agent-studio"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/utils"
	"github.com/algolia/terraform-provider-algolia/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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
					// The applied schema must be the configured bytes, keywords
					// and formatting included.
					resource.TestCheckResourceAttr("algolia_agent.test", "tool_client_side.0.input_schema", testAccAgentInputSchema),
				),
			},
			{
				// A second apply of the same configuration must be a no-op: a
				// re-encoded schema in state would show up as a permanent diff.
				Config:   testAccAgentResourceConfig_withTools(agentName),
				PlanOnly: true,
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

// testAccAgentInputSchema is a pretty-printed JSON Schema carrying keywords the
// Algolia client does not model, and is asserted verbatim on the applied state.
//
// The previous fixture used jsonencode with exactly `{type, properties,
// required}` in alphabetical order, the one shape that survived a round trip
// through agentStudio.ClientToolsArgsSchema, so it could see neither half of the
// defect: `$schema` and `additionalProperties` never reached Algolia, and the
// user's formatting was replaced by the client's. input_schema is Required and so
// has no Computed escape hatch, which makes either one enough to fail the apply
// with "Provider produced inconsistent result after apply".
const testAccAgentInputSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "order_id": {
      "type": "string",
      "description": "The order to look up."
    }
  },
  "required": ["order_id"]
}`

func testAccAgentResourceConfig_withTools(name string) string {
	return fmt.Sprintf(`
resource "algolia_agent" "test" {
  name                = %[1]q
  instructions        = "Use the tools."
  deletion_protection = false

  tool_client_side {
    name         = "get_order"
    description  = "Get order status"
    input_schema = %[2]q
  }
}
`, name, testAccAgentInputSchema)
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

// The two tests below reach past Terraform to the API to force drift, which nothing in
// this package did before: every other agent test only ever drives Terraform, so an
// agent edited or deleted in the dashboard was untested. Both work on an *unpublished*
// agent and therefore need no OPENAI_API_KEY, because Algolia only validates against
// the vendor when an agent is published.

// TestAccAgentResource_recoversFromOutOfBandDeletion covers an agent deleted in the
// dashboard. Read has to drop it from state so the next plan recreates it. Raising an
// error there would fail refresh, and with it plan, apply and destroy together, leaving
// `terraform state rm` as the only way out. algolia_index has this same test for the
// same reason.
func TestAccAgentResource_recoversFromOutOfBandDeletion(t *testing.T) {
	agentName := fmt.Sprintf("tf-test-gone-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	var agentID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAgentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentResourceConfig_basic(agentName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_agent.test", "name", agentName),
					// The API addresses an agent by a generated id, so capture it here:
					// PreConfig below runs before any state is available to it.
					testAccCaptureAgentID("algolia_agent.test", &agentID),
				),
			},
			{
				PreConfig: func() {
					client := testAccAgentStudioClient(t)
					if err := client.DeleteAgent(client.NewApiDeleteAgentRequest(agentID)); err != nil {
						t.Fatalf("deleting agent %s out of band: %v", agentID, err)
					}
				},
				Config: testAccAgentResourceConfig_basic(agentName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						// A create, not an error and not an empty plan.
						plancheck.ExpectResourceAction("algolia_agent.test", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_agent.test", "name", agentName),
				),
			},
		},
	})
}

// TestAccAgentResource_drift covers an agent edited in the dashboard: the refresh has to
// notice, and the plan has to put the configured value back rather than adopting the
// change silently.
func TestAccAgentResource_drift(t *testing.T) {
	agentName := fmt.Sprintf("tf-test-drift-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	const configured = "You are a helpful test agent."

	var agentID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAgentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentResourceConfig_basic(agentName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_agent.test", "instructions", configured),
					testAccCaptureAgentID("algolia_agent.test", &agentID),
				),
			},
			{
				PreConfig: func() {
					client := testAccAgentStudioClient(t)
					update := &agentStudio.AgentConfigUpdate{
						Instructions: *utils.NewNullable(utils.ToPtr("Drifted outside Terraform.")),
					}
					if _, err := client.UpdateAgent(client.NewApiUpdateAgentRequest(agentID, update)); err != nil {
						t.Fatalf("mutating agent %s out of band: %v", agentID, err)
					}
				},
				Config: testAccAgentResourceConfig_basic(agentName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("algolia_agent.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_agent.test", "instructions", configured),
					// Assert Algolia's own view too, not just state: the point of driving
					// drift through the API is that state alone could agree while the
					// remote value stayed wrong.
					testAccCheckAgentInstructions(&agentID, configured),
				),
			},
		},
	})
}

func testAccAgentStudioClient(t *testing.T) *agentStudio.APIClient {
	t.Helper()

	client, err := agentStudio.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		t.Fatalf("creating agent studio client: %v", err)
	}

	return client
}

// testAccCaptureAgentID records the generated agent id so a later step's PreConfig, which
// receives no state, can address the agent through the API.
func testAccCaptureAgentID(resourceName string, into *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("%s not found in state", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("%s has an empty id in state", resourceName)
		}
		*into = rs.Primary.ID

		return nil
	}
}

func testAccCheckAgentInstructions(agentID *string, want string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client, err := agentStudio.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
		if err != nil {
			return fmt.Errorf("creating agent studio client: %w", err)
		}

		doc, err := client.GetAgent(client.NewApiGetAgentRequest(*agentID))
		if err != nil {
			return fmt.Errorf("reading agent %s from the API: %w", *agentID, err)
		}
		if got := doc.GetInstructions(); got != want {
			return fmt.Errorf("agent %s instructions = %q, want %q", *agentID, got, want)
		}

		return nil
	}
}
