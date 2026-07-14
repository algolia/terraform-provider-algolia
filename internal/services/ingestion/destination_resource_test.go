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
	return fmt.Sprintf(`
resource "algolia_ingestion_destination" "test" {
  name = %[1]q
  type = "search"

  input = jsonencode({
    indexName = %[2]q
  })
}
`, name, indexName)
}

func testAccIngestionDestinationDataSourceConfig(name, indexName string) string {
	return testAccIngestionDestinationConfig(name, indexName) + `

data "algolia_ingestion_destination" "test" {
  destination_id = algolia_ingestion_destination.test.destination_id
}
`
}
