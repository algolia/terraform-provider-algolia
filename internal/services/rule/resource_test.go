package rule_test

import (
	"fmt"
	"os"
	"testing"
	"time"

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

func TestAccRuleResource_drift(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-rule-drift-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	objectID := "brand-rule"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleResourceConfig(indexName, objectID, "rule description", `{"query":"iphone"}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_rule.test", "description", "rule description"),
				),
			},
			{
				PreConfig: testAccMutateRule(t, indexName, objectID, "drifted rule description", "galaxy"),
				Config:    testAccRuleResourceConfig(indexName, objectID, "rule description", `{"query":"iphone"}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_rule.test", "description", "rule description"),
				),
			},
		},
	})
}

func TestAccRuleDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-rule-ds-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	objectID := "brand-rule"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleDataSourceConfig(indexName, objectID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.algolia_rule.test", "index_name", indexName),
					resource.TestCheckResourceAttr("data.algolia_rule.test", "object_id", objectID),
					resource.TestCheckResourceAttr("data.algolia_rule.test", "description", "rule description"),
					resource.TestCheckResourceAttr("data.algolia_rule.test", "consequence.0.promote.0.object_ids.#", "2"),
				),
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

func testAccMutateRule(t *testing.T, indexName, objectID, description, query string) func() {
	t.Helper()

	return func() {
		t.Helper()

		client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
		if err != nil {
			t.Fatalf("create Algolia client: %v", err)
		}

		current, err := client.GetRule(client.NewApiGetRuleRequest(indexName, objectID))
		if err != nil {
			t.Fatalf("read rule %s/%s before mutation: %v", indexName, objectID, err)
		}

		current.SetDescription(description)
		consequence := current.GetConsequence()
		params := consequence.GetParams()
		params.SetQuery(search.StringAsConsequenceQuery(query))
		consequence.SetParams(&params)
		current.SetConsequence(&consequence)

		saveResp, err := client.SaveRule(client.NewApiSaveRuleRequest(indexName, objectID, current))
		if err != nil {
			t.Fatalf("mutate rule %s/%s: %v", indexName, objectID, err)
		}

		testAccWaitForRuleMutation(t, client, indexName, objectID, saveResp.TaskID, description, query)
	}
}

func testAccWaitForRuleMutation(t *testing.T, client *search.APIClient, indexName, objectID string, taskID int64, description, query string) {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	for {
		task, err := client.GetTask(client.NewApiGetTaskRequest(indexName, taskID))
		if err == nil && task.Status == search.TASK_STATUS_PUBLISHED {
			rule, err := client.GetRule(client.NewApiGetRuleRequest(indexName, objectID))
			if err == nil && rule.GetDescription() == description {
				consequence := rule.GetConsequence()
				params, ok := consequence.GetParamsOk()
				if ok && params != nil {
					consequenceQuery, ok := params.GetQueryOk()
					if ok && consequenceQuery != nil && consequenceQuery.GetActualInstance() == query {
						return
					}
				}
			}
		}

		if time.Now().After(deadline) {
			t.Fatalf("rule %s/%s mutation did not become visible", indexName, objectID)
		}
		time.Sleep(500 * time.Millisecond)
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

func testAccRuleDataSourceConfig(indexName, objectID string) string {
	return testAccRuleResourceConfig(indexName, objectID, "rule description", `{"query":"iphone"}`) + `

data "algolia_rule" "test" {
  index_name = algolia_rule.test.index_name
  object_id  = algolia_rule.test.object_id
}
`
}
