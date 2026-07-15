package composition_test

import (
	"fmt"
	"os"
	"testing"

	compositionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/composition"
	"github.com/algolia/terraform-provider-algolia/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// The Composition API is, like Recommend, not region-routed - these
// acceptance tests only require the usual TF_ACC/ALGOLIA_APP_ID/
// ALGOLIA_API_KEY (no ALGOLIA_ANALYTICS_REGION).

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"algolia": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestAccCompositionResource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-composition-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	objectID := fmt.Sprintf("tf-composition-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCompositionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCompositionResourceConfig(indexName, objectID, "Test composition"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_composition.test", "object_id", objectID),
					resource.TestCheckResourceAttr("algolia_composition.test", "name", "Test composition"),
					resource.TestCheckResourceAttrSet("algolia_composition.test", "behavior"),
				),
			},
			{
				Config: testAccCompositionResourceConfig(indexName, objectID, "Updated composition"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_composition.test", "name", "Updated composition"),
				),
			},
			{
				ResourceName:      "algolia_composition.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccCompositionDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-composition-ds-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	objectID := fmt.Sprintf("tf-composition-ds-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCompositionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCompositionDataSourceConfig(indexName, objectID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.algolia_composition.test", "object_id", objectID),
					resource.TestCheckResourceAttr("data.algolia_composition.test", "name", "Test composition"),
					resource.TestCheckResourceAttrSet("data.algolia_composition.test", "behavior"),
				),
			},
		},
	})
}

func testAccCheckCompositionDestroy(s *terraform.State) error {
	client, err := compositionapi.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_composition" {
			continue
		}

		objectID := rs.Primary.Attributes["object_id"]

		if _, err := client.GetComposition(client.NewApiGetCompositionRequest(objectID)); err == nil {
			return fmt.Errorf("composition %s still exists", objectID)
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

func testAccCompositionResourceConfig(indexName, objectID, name string) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_composition" "test" {
  object_id = %[2]q
  name      = %[3]q

  behavior = jsonencode({
    injection = {
      main = {
        source = {
          search = {
            index = algolia_index.test.name
          }
        }
      }
    }
  })
}
`, indexName, objectID, name)
}

func testAccCompositionDataSourceConfig(indexName, objectID string) string {
	return testAccCompositionResourceConfig(indexName, objectID, "Test composition") + `

data "algolia_composition" "test" {
  object_id = algolia_composition.test.object_id
}
`
}
