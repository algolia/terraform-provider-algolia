package ingestion_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/algolia/terraform-provider-algolia/internal/analyticsregion"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccProtoV6ProviderFactories and testAccRequireCredentials are shared
// with authentication_resource_test.go (same ingestion_test package).

func TestAccIngestionDestinationResource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	name := fmt.Sprintf("tf-acc-destination-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	indexName := fmt.Sprintf("tf_acc_destination_%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIngestionDestinationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIngestionDestinationConfig(name, indexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_ingestion_destination.test", "name", name),
					resource.TestCheckResourceAttr("algolia_ingestion_destination.test", "type", "search"),
					resource.TestCheckResourceAttrSet("algolia_ingestion_destination.test", "destination_id"),
					resource.TestCheckResourceAttrSet("algolia_ingestion_destination.test", "created_at"),
					resource.TestCheckResourceAttrPair("algolia_ingestion_destination.test", "id", "algolia_ingestion_destination.test", "destination_id"),
				),
			},
			{
				// Renaming exercises UpdateDestination rather than a
				// replace, since only `type` is RequiresReplace.
				Config: testAccIngestionDestinationConfig(name+"-renamed", indexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_ingestion_destination.test", "name", name+"-renamed"),
				),
			},
			{
				ResourceName:      "algolia_ingestion_destination.test",
				ImportState:       true,
				ImportStateVerify: true,
				// deletion_protection is not represented in the Algolia API, so an import
				// cannot read it back and seeds the protected default instead. A fixture
				// that turns protection off therefore differs from the imported value by
				// design, which is the fail-safe working rather than a mismatch.
				ImportStateVerifyIgnore: []string{"deletion_protection"},
			},
		},
	})
}

func TestAccIngestionDestinationDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	name := fmt.Sprintf("tf-acc-destination-ds-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	indexName := fmt.Sprintf("tf_acc_destination_ds_%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIngestionDestinationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIngestionDestinationDataSourceConfig(name, indexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.algolia_ingestion_destination.test", "name", name),
					resource.TestCheckResourceAttr("data.algolia_ingestion_destination.test", "type", "search"),
					resource.TestCheckResourceAttrPair(
						"data.algolia_ingestion_destination.test", "destination_id",
						"algolia_ingestion_destination.test", "destination_id",
					),
				),
			},
		},
	})
}

func testAccCheckIngestionDestinationDestroy(s *terraform.State) error {
	client, err := analyticsregion.NewIngestionClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"), os.Getenv(analyticsregion.EnvVar))
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_ingestion_destination" {
			continue
		}

		destinationID := rs.Primary.Attributes["destination_id"]
		if _, err := client.GetDestination(client.NewApiGetDestinationRequest(destinationID)); err == nil {
			return fmt.Errorf("ingestion destination %s still exists", destinationID)
		}
	}

	return nil
}

func testAccIngestionDestinationConfig(name, indexName string) string {
	// A "search" destination requires an authentication of type "algolia", and
	// the API requires that authentication's appID to match the application
	// making the request - a placeholder is rejected with "appID must match the
	// appID used to do the request".
	return fmt.Sprintf(`
resource "algolia_ingestion_authentication" "destination" {
  deletion_protection = false
  name                = "%[1]s-auth"
  type                = "algolia"

  input               = jsonencode({
    appID  = %[3]q
    apiKey = %[4]q
  })
}

resource "algolia_ingestion_destination" "test" {
  deletion_protection = false
  name                = %[1]q
  type                = "search"
  authentication_id   = algolia_ingestion_authentication.destination.authentication_id

  input               = jsonencode({
    indexName = %[2]q
  })
}
`, name, indexName, os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
}

func testAccIngestionDestinationDataSourceConfig(name, indexName string) string {
	return testAccIngestionDestinationConfig(name, indexName) + `

data "algolia_ingestion_destination" "test" {
  destination_id = algolia_ingestion_destination.test.destination_id
}
`
}
