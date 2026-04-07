package querysuggestions_test

import (
	"fmt"
	"os"
	"testing"

	suggestions "github.com/algolia/algoliasearch-client-go/v4/algolia/query-suggestions"
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

func TestAccQuerySuggestionsResource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	sourceIndexName := fmt.Sprintf("tf-qs-source-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	qsIndexName := fmt.Sprintf("tf-qs-index-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckQuerySuggestionsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccQuerySuggestionsConfig(sourceIndexName, qsIndexName, "mobile"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "index_name", qsIndexName),
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "region", "us"),
				),
			},
			{
				Config: testAccQuerySuggestionsConfig(sourceIndexName, qsIndexName, "desktop"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "region", "us"),
				),
			},
			{
				ResourceName:      "algolia_query_suggestions.test",
				ImportState:       true,
				ImportStateId:     "us/" + qsIndexName,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckQuerySuggestionsDestroy(s *terraform.State) error {
	client, err := suggestions.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"), suggestions.US)
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_query_suggestions" {
			continue
		}

		indexName := rs.Primary.Attributes["index_name"]
		_, err := client.GetConfig(client.NewApiGetConfigRequest(indexName))
		if err == nil {
			return fmt.Errorf("query suggestions config %s still exists", indexName)
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

func testAccQuerySuggestionsConfig(sourceIndexName, qsIndexName, analyticsTag string) string {
	return fmt.Sprintf(`
resource "algolia_index" "source" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_query_suggestions" "test" {
  index_name = %[2]q
  region     = "us"
  languages  = ["en"]
  exclude    = ["free"]

  source_indices {
    index_name     = algolia_index.source.name
    analytics_tags = [%[3]q]
    min_hits       = 10
    min_letters    = 3
    external       = ["external_products"]

    facets {
      attribute = "brand"
      amount    = 5
    }

    generate = [
      ["brand", "category"]
    ]
  }
}
`, sourceIndexName, qsIndexName, analyticsTag)
}
