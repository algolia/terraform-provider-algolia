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
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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

func TestAccRuleResource_objectIDChangeForcesReplacement(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-test-rule-replace-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleResourceConfig(indexName, "brand-rule", "rule description", `{"query":"iphone"}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_rule.test", "object_id", "brand-rule"),
				),
			},
			{
				// Renaming the rule must destroy the original object instead of
				// writing to the new identity and leaving the old rule live.
				Config: testAccRuleResourceConfig(indexName, "brand-rule-renamed", "rule description", `{"query":"iphone"}`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("algolia_rule.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_rule.test", "object_id", "brand-rule-renamed"),
					testAccCheckRuleAbsent(indexName, "brand-rule"),
				),
			},
		},
	})
}

func TestAccRuleResource_indexNameChangeForcesReplacement(t *testing.T) {
	testAccRequireCredentials(t)

	firstIndex := fmt.Sprintf("tf-test-rule-idx-a-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	secondIndex := fmt.Sprintf("tf-test-rule-idx-b-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleResourceConfig(firstIndex, "brand-rule", "rule description", `{"query":"iphone"}`),
			},
			{
				Config: testAccRuleResourceConfig(secondIndex, "brand-rule", "rule description", `{"query":"iphone"}`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("algolia_rule.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_rule.test", "index_name", secondIndex),
				),
			},
		},
	})
}

// TestAccRuleResource_unmodelledFields covers the fields that used to be wiped
// on every apply because the provider never modelled them, plus the verbatim
// params_json round trip.
func TestAccRuleResource_unmodelledFields(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-test-rule-fields-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	objectID := "brand-rule"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleUnmodelledFieldsConfig(indexName, objectID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_rule.test", "tags.#", "2"),
					resource.TestCheckResourceAttr("algolia_rule.test", "tags.0", "tf-test-seasonal"),
					resource.TestCheckResourceAttr("algolia_rule.test", "tags.1", "tf-test-promo"),
					resource.TestCheckResourceAttr("algolia_rule.test", "consequence.0.filter_promotes", "true"),
					// The configured document is stored verbatim even though a
					// re-encode would sort `filters` ahead of `query`.
					resource.TestCheckResourceAttr("algolia_rule.test", "consequence.0.params_json", `{"query":"iphone","filters":"brand:apple"}`),
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

// TestAccRuleResource_nonCanonicalFormats covers the two configured formats
// Algolia does not store as written: a validity window in local time (the API
// keeps a Unix second) and a user_data document whose keys are not in sorted
// order (the API's response decodes into a map). Both are valid configuration for
// Optional, non-Computed attributes, so re-rendering either from the response
// fails the apply with "Provider produced inconsistent result after apply".
//
// There is deliberately no import step. `terraform import` has no configuration
// to preserve, so it stores Algolia's own representation - a UTC timestamp, and
// sorted keys - and ImportStateVerify would report that as a mismatch. For
// validity that is irreducible: a zone offset is not recoverable from a Unix
// second.
func TestAccRuleResource_nonCanonicalFormats(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-test-rule-formats-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	objectID := "brand-rule"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleNonCanonicalFormatsConfig(indexName, objectID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_rule.test", "consequence.0.user_data", `{"banner":"promo","alt":"x"}`),
					resource.TestCheckResourceAttr("algolia_rule.test", "validity.0.from", "2030-01-01T00:00:00+02:00"),
					resource.TestCheckResourceAttr("algolia_rule.test", "validity.0.until", "2030-01-02T00:00:00.500+02:00"),
				),
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

// testAccCheckRuleAbsent asserts that a rule the provider stopped tracking was
// actually deleted rather than orphaned on the index.
func testAccCheckRuleAbsent(indexName, objectID string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
		if err != nil {
			return err
		}

		if _, err := client.GetRule(client.NewApiGetRuleRequest(indexName, objectID)); err == nil {
			return fmt.Errorf("rule %s/%s still exists", indexName, objectID)
		}

		return nil
	}
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

// testAccRuleNonCanonicalFormatsConfig writes user_data with its keys in an
// order a re-encode would not produce ("alt" sorts before "banner"), and a
// validity window in a +02:00 offset with sub-second precision.
func testAccRuleNonCanonicalFormatsConfig(indexName, objectID string) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_rule" "test" {
  index_name = algolia_index.test.name
  object_id  = %[2]q

  conditions {
    pattern   = "shoes"
    anchoring = "contains"
  }

  consequence {
    params_json = "{\"query\":\"sneakers\"}"
    user_data   = "{\"banner\":\"promo\",\"alt\":\"x\"}"
  }

  validity {
    from  = "2030-01-01T00:00:00+02:00"
    until = "2030-01-02T00:00:00.500+02:00"
  }
}
`, indexName, objectID)
}

func testAccRuleUnmodelledFieldsConfig(indexName, objectID string) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_rule" "test" {
  index_name = algolia_index.test.name
  object_id  = %[2]q
  tags       = ["tf-test-seasonal", "tf-test-promo"]

  conditions {
    pattern   = "{facet:brand}"
    anchoring = "contains"
  }

  consequence {
    params_json     = %[3]q
    filter_promotes = true

    promote {
      object_ids = ["1", "2"]
      position   = 0
    }
  }
}
`, indexName, objectID, `{"query":"iphone","filters":"brand:apple"}`)
}

func testAccRuleDataSourceConfig(indexName, objectID string) string {
	return testAccRuleResourceConfig(indexName, objectID, "rule description", `{"query":"iphone"}`) + `

data "algolia_rule" "test" {
  index_name = algolia_rule.test.index_name
  object_id  = algolia_rule.test.object_id
}
`
}
