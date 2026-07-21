package recommend_test

import (
	"fmt"
	"os"
	"testing"

	recommendapi "github.com/algolia/algoliasearch-client-go/v4/algolia/recommend"
	"github.com/algolia/terraform-provider-algolia/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// The Recommend API is, unlike Query Suggestions/Personalization/Ingestion/
// A-B Testing, not region-routed - these acceptance tests only require the
// usual TF_ACC/ALGOLIA_APP_ID/ALGOLIA_API_KEY (no ALGOLIA_ANALYTICS_REGION).

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"algolia": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestAccRecommendRuleResource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-recommend-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	objectID := "hide-rule"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRecommendRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRecommendRuleResourceConfig(indexName, objectID, `{"hide":[{"objectID":"42"}]}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_recommend_rule.test", "index_name", indexName),
					resource.TestCheckResourceAttr("algolia_recommend_rule.test", "model", "related-products"),
					resource.TestCheckResourceAttr("algolia_recommend_rule.test", "object_id", objectID),
					resource.TestCheckResourceAttr("algolia_recommend_rule.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("algolia_recommend_rule.test", "consequence"),
				),
			},
			{
				Config: testAccRecommendRuleResourceConfig(indexName, objectID, `{"hide":[{"objectID":"42"},{"objectID":"43"}]}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_recommend_rule.test", "index_name", indexName),
				),
			},
			{
				ResourceName:      "algolia_recommend_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRecommendRuleResource_generatedObjectID(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-recommend-gen-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRecommendRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_recommend_rule" "test" {
  index_name  = algolia_index.test.name
  model       = "bought-together"
  consequence = jsonencode({ hide = [{ objectID = "42" }] })
}
`, indexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("algolia_recommend_rule.test", "object_id"),
				),
			},
		},
	})
}

func TestAccRecommendRuleDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-recommend-ds-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	objectID := "hide-rule"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRecommendRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRecommendRuleDataSourceConfig(indexName, objectID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.algolia_recommend_rule.test", "index_name", indexName),
					resource.TestCheckResourceAttr("data.algolia_recommend_rule.test", "model", "related-products"),
					resource.TestCheckResourceAttr("data.algolia_recommend_rule.test", "object_id", objectID),
					resource.TestCheckResourceAttrSet("data.algolia_recommend_rule.test", "consequence"),
				),
			},
		},
	})
}

func testAccCheckRecommendRuleDestroy(s *terraform.State) error {
	client, err := recommendapi.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_recommend_rule" {
			continue
		}

		indexName := rs.Primary.Attributes["index_name"]
		model := recommendapi.RecommendModels(rs.Primary.Attributes["model"])
		objectID := rs.Primary.Attributes["object_id"]

		if _, err := client.GetRecommendRule(client.NewApiGetRecommendRuleRequest(indexName, model, objectID)); err == nil {
			return fmt.Errorf("recommend rule %s/%s/%s still exists", indexName, model, objectID)
		}
	}

	return nil
}

func testAccRequireCredentials(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	// Gated behind an explicit flag: the vendored algoliasearch-client-go v4
	// types RuleMetadata.lastUpdate as *string, but the API returns it as a
	// number, so the read after create fails to decode (upstream client bug).
	// See https://github.com/algolia/algoliasearch-client-go. Remove this gate
	// once the client is fixed.
	if os.Getenv("ALGOLIA_RUN_RECOMMEND_ACC") != "1" {
		t.Skip("Set ALGOLIA_RUN_RECOMMEND_ACC=1 to run Recommend rule acceptance tests; currently blocked by an upstream client decode bug (RuleMetadata.lastUpdate typed as *string vs numeric API response)")
	}

	if os.Getenv("ALGOLIA_APP_ID") == "" || os.Getenv("ALGOLIA_API_KEY") == "" {
		t.Skip("ALGOLIA_APP_ID and ALGOLIA_API_KEY must be set for acceptance tests")
	}
}

func testAccRecommendRuleResourceConfig(indexName, objectID, consequenceJSON string) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_recommend_rule" "test" {
  index_name  = algolia_index.test.name
  model       = "related-products"
  object_id   = %[2]q
  description = "hide discontinued items"
  enabled     = true

  consequence = %[3]q
}
`, indexName, objectID, consequenceJSON)
}

func testAccRecommendRuleDataSourceConfig(indexName, objectID string) string {
	return testAccRecommendRuleResourceConfig(indexName, objectID, `{"hide":[{"objectID":"42"}]}`) + `

data "algolia_recommend_rule" "test" {
  index_name = algolia_recommend_rule.test.index_name
  model      = algolia_recommend_rule.test.model
  object_id  = algolia_recommend_rule.test.object_id
}
`
}
