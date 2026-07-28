package ingestion_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/algolia/terraform-provider-algolia/internal/analyticsregion"
	"github.com/algolia/terraform-provider-algolia/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// The Ingestion API is region-routed (see internal/analyticsregion), so
// these acceptance tests additionally require ALGOLIA_ANALYTICS_REGION to
// be set, on top of the usual TF_ACC/ALGOLIA_APP_ID/ALGOLIA_API_KEY.

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"algolia": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestAccIngestionAuthenticationResource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	name := fmt.Sprintf("tf-acc-auth-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIngestionAuthenticationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIngestionAuthenticationConfig(name, os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_ingestion_authentication.test", "name", name),
					resource.TestCheckResourceAttr("algolia_ingestion_authentication.test", "type", "algolia"),
					resource.TestCheckResourceAttrSet("algolia_ingestion_authentication.test", "authentication_id"),
					resource.TestCheckResourceAttrSet("algolia_ingestion_authentication.test", "created_at"),
					resource.TestCheckResourceAttrPair("algolia_ingestion_authentication.test", "id", "algolia_ingestion_authentication.test", "authentication_id"),
				),
			},
			{
				// Renaming (and rotating the dummy credentials) exercises
				// UpdateAuthentication rather than a replace, since only
				// `type` and `platform` are RequiresReplace.
				Config: testAccIngestionAuthenticationConfig(name+"-renamed", os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_ingestion_authentication.test", "name", name+"-renamed"),
				),
			},
			{
				ResourceName:            "algolia_ingestion_authentication.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"input"},
			},
		},
	})
}

func TestAccIngestionAuthenticationDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	name := fmt.Sprintf("tf-acc-auth-ds-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIngestionAuthenticationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIngestionAuthenticationDataSourceConfig(name, os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.algolia_ingestion_authentication.test", "name", name),
					resource.TestCheckResourceAttr("data.algolia_ingestion_authentication.test", "type", "algolia"),
					resource.TestCheckResourceAttrPair(
						"data.algolia_ingestion_authentication.test", "authentication_id",
						"algolia_ingestion_authentication.test", "authentication_id",
					),
				),
			},
		},
	})
}

func testAccCheckIngestionAuthenticationDestroy(s *terraform.State) error {
	client, err := analyticsregion.NewIngestionClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"), os.Getenv(analyticsregion.EnvVar))
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_ingestion_authentication" {
			continue
		}

		authenticationID := rs.Primary.Attributes["authentication_id"]
		if _, err := client.GetAuthentication(client.NewApiGetAuthenticationRequest(authenticationID)); err == nil {
			return fmt.Errorf("ingestion authentication %s still exists", authenticationID)
		}
	}

	return nil
}

func testAccRequireCredentials(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	if os.Getenv("ALGOLIA_APP_ID") == "" || os.Getenv("ALGOLIA_API_KEY") == "" || os.Getenv(analyticsregion.EnvVar) == "" {
		t.Skip("ALGOLIA_APP_ID, ALGOLIA_API_KEY, and ALGOLIA_ANALYTICS_REGION must be set for acceptance tests")
	}
}

func testAccIngestionAuthenticationConfig(name, appID, apiKey string) string {
	return fmt.Sprintf(`
resource "algolia_ingestion_authentication" "test" {
  name = %[1]q
  type = "algolia"

  input = jsonencode({
    appID  = %[2]q
    apiKey = %[3]q
  })
}
`, name, appID, apiKey)
}

func testAccIngestionAuthenticationDataSourceConfig(name, appID, apiKey string) string {
	return testAccIngestionAuthenticationConfig(name, appID, apiKey) + `

data "algolia_ingestion_authentication" "test" {
  authentication_id = algolia_ingestion_authentication.test.authentication_id
}
`
}
