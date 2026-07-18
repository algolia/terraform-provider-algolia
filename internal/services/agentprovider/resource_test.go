package agentprovider_test

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
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

func TestAccAgentProviderResource_openai(t *testing.T) {
	testAccRequireOpenAIKey(t)

	providerName := fmt.Sprintf("tf-provider-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	updatedName := providerName + "-updated"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAgentProviderDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentProviderResourceConfigOpenAI(providerName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_agent_provider.test", "name", providerName),
					resource.TestCheckResourceAttr("algolia_agent_provider.test", "provider_name", "openai"),
					resource.TestCheckResourceAttrSet("algolia_agent_provider.test", "id"),
					resource.TestCheckResourceAttrSet("algolia_agent_provider.test", "created_at"),
				),
			},
			{
				Config: testAccAgentProviderResourceConfigOpenAI(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_agent_provider.test", "name", updatedName),
				),
			},
			{
				ResourceName: "algolia_agent_provider.test",
				ImportState:  true,
			},
			{
				Config: testAccAgentProviderResourceConfigOpenAI(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_agent_provider.test", "name", updatedName),
				),
			},
		},
	})
}

func TestAccAgentProviderModelsDataSource_openai(t *testing.T) {
	testAccRequireOpenAIKey(t)

	providerName := fmt.Sprintf("tf-provider-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAgentProviderDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentProviderModelsDataSourceConfigOpenAI(providerName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.algolia_agent_provider_models.test", "models.0"),
					testAccCheckProviderModelsContains("data.algolia_agent_provider_models.test", "gpt-4.1-mini"),
				),
			},
		},
	})
}

func TestAccAgentProviderDataSource_openai(t *testing.T) {
	testAccRequireOpenAIKey(t)

	providerName := fmt.Sprintf("tf-provider-ds-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAgentProviderDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentProviderDataSourceConfigOpenAI(providerName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.algolia_agent_provider.test", "provider_id", "algolia_agent_provider.test", "id"),
					resource.TestCheckResourceAttr("data.algolia_agent_provider.test", "name", providerName),
					resource.TestCheckResourceAttr("data.algolia_agent_provider.test", "provider_name", "openai"),
					resource.TestCheckResourceAttr("data.algolia_agent_provider.test", "openai.base_url", "https://api.openai.com/v1"),
				),
			},
		},
	})
}

func TestAccAgentProviderResource_googleGenAI(t *testing.T) {
	testAccRequireGoogleGenAIKey(t)

	providerName := fmt.Sprintf("tf-provider-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAgentProviderDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentProviderResourceConfigGoogleGenAI(providerName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_agent_provider.test", "name", providerName),
					resource.TestCheckResourceAttr("algolia_agent_provider.test", "provider_name", "google_genai"),
				),
			},
		},
	})
}

func testAccCheckAgentProviderDestroy(s *terraform.State) error {
	client, err := agentStudio.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		return fmt.Errorf("creating agent studio client: %w", err)
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_agent_provider" {
			continue
		}

		_, err := client.GetProvider(client.NewApiGetProviderRequest(rs.Primary.ID))
		if err == nil {
			return fmt.Errorf("provider %s still exists", rs.Primary.ID)
		}

		var apiErr *agentStudio.APIError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
			continue
		}

		return fmt.Errorf("checking provider %s destroy state: %w", rs.Primary.ID, err)
	}

	return nil
}

func testAccCheckProviderModelsContains(resourceName, expected string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}

		for key, value := range rs.Primary.Attributes {
			if strings.HasPrefix(key, "models.") && value == expected {
				return nil
			}
		}

		return fmt.Errorf("resource %s models did not contain %q", resourceName, expected)
	}
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
		t.Skip("OPENAI_API_KEY must be set for openai provider acceptance tests")
	}
}

func testAccRequireGoogleGenAIKey(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	if os.Getenv("ALGOLIA_APP_ID") == "" || os.Getenv("ALGOLIA_API_KEY") == "" {
		t.Skip("ALGOLIA_APP_ID and ALGOLIA_API_KEY must be set for acceptance tests")
	}

	if os.Getenv("GOOGLE_GENAI_API_KEY") == "" {
		t.Skip("GOOGLE_GENAI_API_KEY must be set for google_genai provider acceptance tests")
	}
}

func testAccAgentProviderResourceConfigOpenAI(name string) string {
	return fmt.Sprintf(`
resource "algolia_agent_provider" "test" {
  name          = %[1]q
  provider_name = "openai"

  openai {
    api_key = %[2]q
  }
}
`, name, os.Getenv("OPENAI_API_KEY"))
}

func testAccAgentProviderModelsDataSourceConfigOpenAI(name string) string {
	return fmt.Sprintf(`
resource "algolia_agent_provider" "test" {
  name          = %[1]q
  provider_name = "openai"

  openai {
    api_key = %[2]q
  }
}

data "algolia_agent_provider_models" "test" {
  provider_id = algolia_agent_provider.test.id
}
`, name, os.Getenv("OPENAI_API_KEY"))
}

func testAccAgentProviderDataSourceConfigOpenAI(name string) string {
	return fmt.Sprintf(`
resource "algolia_agent_provider" "test" {
  name          = %[1]q
  provider_name = "openai"

  openai {
    api_key  = %[2]q
    base_url = "https://api.openai.com/v1"
  }
}

data "algolia_agent_provider" "test" {
  provider_id = algolia_agent_provider.test.id
}
`, name, os.Getenv("OPENAI_API_KEY"))
}

func testAccAgentProviderResourceConfigGoogleGenAI(name string) string {
	return fmt.Sprintf(`
resource "algolia_agent_provider" "test" {
  name          = %[1]q
  provider_name = "google_genai"

  google_genai {
    api_key = %[2]q
  }
}
`, name, os.Getenv("GOOGLE_GENAI_API_KEY"))
}
