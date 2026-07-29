package synonym_test

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

func TestAccSynonymResource_objectIDChangeForcesReplacement(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-test-synonym-replace-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSynonymDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSynonymRegularConfig(indexName, "brand-synonym"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_synonym.test", "object_id", "brand-synonym"),
				),
			},
			{
				// Renaming the synonym must destroy the original object instead
				// of writing to the new identity and leaving the old one live.
				Config: testAccSynonymRegularConfig(indexName, "brand-synonym-renamed"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("algolia_synonym.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_synonym.test", "object_id", "brand-synonym-renamed"),
					testAccCheckSynonymAbsent(indexName, "brand-synonym"),
				),
			},
		},
	})
}

func TestAccSynonymResource_indexNameChangeForcesReplacement(t *testing.T) {
	testAccRequireCredentials(t)

	firstIndex := fmt.Sprintf("tf-test-synonym-idx-a-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	secondIndex := fmt.Sprintf("tf-test-synonym-idx-b-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSynonymDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSynonymRegularConfig(firstIndex, "brand-synonym"),
			},
			{
				Config: testAccSynonymRegularConfig(secondIndex, "brand-synonym"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("algolia_synonym.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_synonym.test", "index_name", secondIndex),
				),
			},
		},
	})
}

// TestAccSynonymResource_explicitEmptyCollections covers collections that are
// configured as `[]`: the API never returns them, so the read has to keep them
// known-empty instead of null or the apply is rejected as inconsistent.
func TestAccSynonymResource_explicitEmptyCollections(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-test-synonym-empty-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	objectID := "brand-synonym"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSynonymDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSynonymEmptyCollectionsConfig(indexName, objectID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_synonym.test", "synonyms.#", "2"),
					resource.TestCheckResourceAttr("algolia_synonym.test", "corrections.#", "0"),
					resource.TestCheckResourceAttr("algolia_synonym.test", "replacements.#", "0"),
				),
			},
		},
	})
}

func TestAccSynonymResource_drift(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-synonym-drift-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	objectID := "brand-synonym"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSynonymDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSynonymRegularConfig(indexName, objectID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_synonym.test", "type", "synonym"),
					resource.TestCheckResourceAttr("algolia_synonym.test", "synonyms.#", "2"),
				),
			},
			{
				PreConfig: testAccMutateSynonym(t, indexName, objectID, []string{"android phone", "google phone"}),
				Config:    testAccSynonymRegularConfig(indexName, objectID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_synonym.test", "type", "synonym"),
					resource.TestCheckResourceAttr("algolia_synonym.test", "synonyms.#", "2"),
				),
			},
		},
	})
}

func TestAccSynonymDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-synonym-ds-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	objectID := "brand-synonym"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSynonymDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSynonymDataSourceConfig(indexName, objectID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.algolia_synonym.test", "index_name", indexName),
					resource.TestCheckResourceAttr("data.algolia_synonym.test", "object_id", objectID),
					resource.TestCheckResourceAttr("data.algolia_synonym.test", "type", "synonym"),
					resource.TestCheckResourceAttr("data.algolia_synonym.test", "synonyms.#", "2"),
				),
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

// testAccCheckSynonymAbsent asserts that a synonym the provider stopped
// tracking was actually deleted rather than orphaned on the index.
func testAccCheckSynonymAbsent(indexName, objectID string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
		if err != nil {
			return err
		}

		if _, err := client.GetSynonym(client.NewApiGetSynonymRequest(indexName, objectID)); err == nil {
			return fmt.Errorf("synonym %s/%s still exists", indexName, objectID)
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

func testAccMutateSynonym(t *testing.T, indexName, objectID string, synonyms []string) func() {
	t.Helper()

	return func() {
		t.Helper()

		client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
		if err != nil {
			t.Fatalf("create Algolia client: %v", err)
		}

		current, err := client.GetSynonym(client.NewApiGetSynonymRequest(indexName, objectID))
		if err != nil {
			t.Fatalf("read synonym %s/%s before mutation: %v", indexName, objectID, err)
		}

		current.SetSynonyms(synonyms)
		saveResp, err := client.SaveSynonym(client.NewApiSaveSynonymRequest(indexName, objectID, current))
		if err != nil {
			t.Fatalf("mutate synonym %s/%s: %v", indexName, objectID, err)
		}

		testAccWaitForSynonymMutation(t, client, indexName, objectID, saveResp.TaskID, synonyms)
	}
}

func testAccWaitForSynonymMutation(t *testing.T, client *search.APIClient, indexName, objectID string, taskID int64, expected []string) {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	for {
		task, err := client.GetTask(client.NewApiGetTaskRequest(indexName, taskID))
		if err == nil && task.Status == search.TASK_STATUS_PUBLISHED {
			synonym, err := client.GetSynonym(client.NewApiGetSynonymRequest(indexName, objectID))
			if err == nil {
				got := synonym.GetSynonyms()
				if len(got) == len(expected) {
					match := true
					for i := range got {
						if got[i] != expected[i] {
							match = false
							break
						}
					}
					if match {
						return
					}
				}
			}
		}

		if time.Now().After(deadline) {
			t.Fatalf("synonym %s/%s mutation did not become visible", indexName, objectID)
		}
		time.Sleep(500 * time.Millisecond)
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

func testAccSynonymEmptyCollectionsConfig(indexName, objectID string) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_synonym" "test" {
  index_name   = algolia_index.test.name
  object_id    = %[2]q
  type         = "synonym"
  synonyms     = ["iphone", "ios phone"]
  corrections  = []
  replacements = []
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

func testAccSynonymDataSourceConfig(indexName, objectID string) string {
	return testAccSynonymRegularConfig(indexName, objectID) + `

data "algolia_synonym" "test" {
  index_name = algolia_synonym.test.index_name
  object_id  = algolia_synonym.test.object_id
}
`
}
