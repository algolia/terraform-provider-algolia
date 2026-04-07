package rule_test

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

func TestAccRuleResource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-rule-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	objectID := "brand-rule"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleResourceConfig(indexName, objectID, "rule description", `{"query":"iphone"}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_rule.test", "index_name", indexName),
					resource.TestCheckResourceAttr("algolia_rule.test", "object_id", objectID),
					resource.TestCheckResourceAttr("algolia_rule.test", "description", "rule description"),
				),
			},
			{
				Config: testAccRuleResourceConfig(indexName, objectID, "updated rule description", `{"query":"ipad"}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_rule.test", "description", "updated rule description"),
				),
			},
			{
				ResourceName:      "algolia_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckRuleDestroy(s *terraform.State) error {
	client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_rule" {
			continue
		}

		indexName := rs.Primary.Attributes["index_name"]
		objectID := rs.Primary.Attributes["object_id"]
		_, err := client.GetRule(client.NewApiGetRuleRequest(indexName, objectID))
		if err == nil {
			return fmt.Errorf("rule %s/%s still exists", indexName, objectID)
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

func testAccRuleResourceConfig(indexName, objectID, description, paramsJSON string) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_rule" "test" {
  index_name   = algolia_index.test.name
  object_id    = %[2]q
  description  = %[3]q
  enabled      = true

  conditions {
    pattern      = "{facet:brand}"
    anchoring    = "contains"
    alternatives = true
    context      = "mobile"
  }

  consequence {
    params_json = %[4]q

    promote {
      object_ids = ["1", "2"]
      position   = 0
    }

    hide      = ["3"]
    user_data = "{\"banner\":\"promo\"}"
  }

  validity {
    from  = "2030-01-01T00:00:00Z"
    until = "2030-01-02T00:00:00Z"
  }
}
`, indexName, objectID, description, paramsJSON)
}
