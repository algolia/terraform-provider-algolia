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

func TestAccIngestionSourceResource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	name := fmt.Sprintf("tf-acc-source-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIngestionSourceDestroy,
		Steps: []resource.TestStep{
			{
				// "push" needs no `input` at all, keeping this test free of
				// any dependency on an external platform/credential.
				Config: testAccIngestionSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_ingestion_source.test", "name", name),
					resource.TestCheckResourceAttr("algolia_ingestion_source.test", "type", "push"),
					resource.TestCheckResourceAttrSet("algolia_ingestion_source.test", "source_id"),
					resource.TestCheckResourceAttrSet("algolia_ingestion_source.test", "created_at"),
					resource.TestCheckResourceAttrPair("algolia_ingestion_source.test", "id", "algolia_ingestion_source.test", "source_id"),
				),
			},
			{
				// Renaming exercises UpdateSource rather than a replace,
				// since only `type` is RequiresReplace.
				Config: testAccIngestionSourceConfig(name + "-renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_ingestion_source.test", "name", name+"-renamed"),
				),
			},
			{
				ResourceName:      "algolia_ingestion_source.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccIngestionSourceDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	name := fmt.Sprintf("tf-acc-source-ds-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIngestionSourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIngestionSourceDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.algolia_ingestion_source.test", "name", name),
					resource.TestCheckResourceAttr("data.algolia_ingestion_source.test", "type", "push"),
					resource.TestCheckResourceAttrPair(
						"data.algolia_ingestion_source.test", "source_id",
						"algolia_ingestion_source.test", "source_id",
					),
				),
			},
		},
	})
}

func testAccCheckIngestionSourceDestroy(s *terraform.State) error {
	client, err := analyticsregion.NewIngestionClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"), os.Getenv(analyticsregion.EnvVar))
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_ingestion_source" {
			continue
		}

		sourceID := rs.Primary.Attributes["source_id"]
		if _, err := client.GetSource(client.NewApiGetSourceRequest(sourceID)); err == nil {
			return fmt.Errorf("ingestion source %s still exists", sourceID)
		}
	}

	return nil
}

func testAccIngestionSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "algolia_ingestion_source" "test" {
  name = %[1]q
  type = "push"
}
`, name)
}

func testAccIngestionSourceDataSourceConfig(name string) string {
	return testAccIngestionSourceConfig(name) + `

data "algolia_ingestion_source" "test" {
  source_id = algolia_ingestion_source.test.source_id
}
`
}
