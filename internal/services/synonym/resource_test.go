package synonym_test

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

func TestAccSynonymResource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-synonym-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	objectID := "brand-synonym"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSynonymDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSynonymRegularConfig(indexName, objectID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_synonym.test", "index_name", indexName),
					resource.TestCheckResourceAttr("algolia_synonym.test", "object_id", objectID),
					resource.TestCheckResourceAttr("algolia_synonym.test", "type", "synonym"),
				),
			},
			{
				Config: testAccSynonymOneWayConfig(indexName, objectID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_synonym.test", "type", "oneWaySynonym"),
					resource.TestCheckResourceAttr("algolia_synonym.test", "input", "iphone"),
				),
			},
			{
				ResourceName:      "algolia_synonym.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckSynonymDestroy(s *terraform.State) error {
	client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_synonym" {
			continue
		}

		indexName := rs.Primary.Attributes["index_name"]
		objectID := rs.Primary.Attributes["object_id"]
		_, err := client.GetSynonym(client.NewApiGetSynonymRequest(indexName, objectID))
		if err == nil {
			return fmt.Errorf("synonym %s/%s still exists", indexName, objectID)
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

func testAccSynonymRegularConfig(indexName, objectID string) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_synonym" "test" {
  index_name = algolia_index.test.name
  object_id  = %[2]q
  type       = "synonym"
  synonyms   = ["iphone", "ios phone"]
}
`, indexName, objectID)
}

func testAccSynonymOneWayConfig(indexName, objectID string) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_synonym" "test" {
  index_name = algolia_index.test.name
  object_id  = %[2]q
  type       = "oneWaySynonym"
  input      = "iphone"
  synonyms   = ["ios phone", "apple phone"]
}
`, indexName, objectID)
}
