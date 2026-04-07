package apikey_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
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

func TestAccAPIKeyResource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	description := fmt.Sprintf("tf-api-key-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	updatedDescription := description + "-updated"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAPIKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyResourceConfig(description, 100),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("algolia_api_key.test", "id"),
					resource.TestCheckResourceAttr("algolia_api_key.test", "description", description),
					resource.TestCheckResourceAttr("algolia_api_key.test", "max_hits_per_query", "100"),
					resource.TestCheckResourceAttrSet("algolia_api_key.test", "created_at"),
				),
			},
			{
				Config: testAccAPIKeyResourceConfig(updatedDescription, 200),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_api_key.test", "description", updatedDescription),
					resource.TestCheckResourceAttr("algolia_api_key.test", "max_hits_per_query", "200"),
				),
			},
			{
				ResourceName:            "algolia_api_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"expires_at"},
			},
			{
				Config: testAccAPIKeyResourceConfig(updatedDescription, 200),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_api_key.test", "description", updatedDescription),
				),
			},
		},
	})
}

func testAccCheckAPIKeyDestroy(s *terraform.State) error {
	client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_api_key" {
			continue
		}

		_, err := client.GetApiKey(client.NewApiGetApiKeyRequest(rs.Primary.ID))
		if err == nil {
			return fmt.Errorf("api key %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccRequireCredentials(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	if os.Getenv("ALGOLIA_APP_ID") == "" || os.Getenv("ALGOLIA_API_KEY") == "" {
		t.Skip("ALGOLIA_APP_ID and ALGOLIA_API_KEY must be set for acceptance tests")
	}
}

func testAccAPIKeyResourceConfig(description string, maxHits int) string {
	return fmt.Sprintf(`
resource "algolia_api_key" "test" {
  acl                         = ["search", "browse"]
  description                 = %[1]q
  expires_at                  = "2030-01-01T00:00:00Z"
  indexes                     = ["products_*"]
  referers                    = ["https://example.com/*"]
  max_hits_per_query          = %[2]d
  max_queries_per_ip_per_hour = 1000
}
`, description, maxHits)
}
