package dictionary_test

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

func TestAccDictionaryEntryResource_stopword(t *testing.T) {
	testAccRequireCredentials(t)

	objectID := fmt.Sprintf("tf-dict-stopword-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	word := fmt.Sprintf("tfstopword%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDictionaryEntryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDictionaryEntryStopwordConfig(objectID, word, "enabled"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_dictionary_entry.test", "dictionary", "stopwords"),
					resource.TestCheckResourceAttr("algolia_dictionary_entry.test", "object_id", objectID),
					resource.TestCheckResourceAttr("algolia_dictionary_entry.test", "language", "en"),
					resource.TestCheckResourceAttr("algolia_dictionary_entry.test", "word", word),
					resource.TestCheckResourceAttr("algolia_dictionary_entry.test", "state", "enabled"),
				),
			},
			{
				Config: testAccDictionaryEntryStopwordConfig(objectID, word, "disabled"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_dictionary_entry.test", "state", "disabled"),
				),
			},
			{
				ResourceName:      "algolia_dictionary_entry.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDictionaryEntryResource_plural(t *testing.T) {
	testAccRequireCredentials(t)

	objectID := fmt.Sprintf("tf-dict-plural-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDictionaryEntryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDictionaryEntryPluralConfig(objectID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_dictionary_entry.test", "dictionary", "plurals"),
					resource.TestCheckResourceAttr("algolia_dictionary_entry.test", "language", "fr"),
					resource.TestCheckResourceAttr("algolia_dictionary_entry.test", "words.#", "2"),
				),
			},
			{
				ResourceName:      "algolia_dictionary_entry.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDictionaryEntryResource_compound(t *testing.T) {
	testAccRequireCredentials(t)

	objectID := fmt.Sprintf("tf-dict-compound-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDictionaryEntryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDictionaryEntryCompoundConfig(objectID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_dictionary_entry.test", "dictionary", "compounds"),
					resource.TestCheckResourceAttr("algolia_dictionary_entry.test", "language", "de"),
					resource.TestCheckResourceAttr("algolia_dictionary_entry.test", "word", "kopfschmerzen"),
					resource.TestCheckResourceAttr("algolia_dictionary_entry.test", "decomposition.#", "2"),
				),
			},
			{
				ResourceName:      "algolia_dictionary_entry.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDictionaryEntryResource_generatedObjectID(t *testing.T) {
	testAccRequireCredentials(t)

	word := fmt.Sprintf("tfstopwordgen%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDictionaryEntryDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "algolia_dictionary_entry" "test" {
  dictionary = "stopwords"
  language   = "en"
  word       = %[1]q
}
`, word),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("algolia_dictionary_entry.test", "object_id"),
					resource.TestCheckResourceAttr("algolia_dictionary_entry.test", "word", word),
				),
			},
		},
	})
}

func TestAccDictionaryEntryDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	objectID := fmt.Sprintf("tf-dict-ds-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	word := fmt.Sprintf("tfstopwordds%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDictionaryEntryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDictionaryEntryDataSourceConfig(objectID, word),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.algolia_dictionary_entry.test", "dictionary", "stopwords"),
					resource.TestCheckResourceAttr("data.algolia_dictionary_entry.test", "object_id", objectID),
					resource.TestCheckResourceAttr("data.algolia_dictionary_entry.test", "word", word),
				),
			},
		},
	})
}

func testAccCheckDictionaryEntryDestroy(s *terraform.State) error {
	client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_dictionary_entry" {
			continue
		}

		dictionaryType := search.DictionaryType(rs.Primary.Attributes["dictionary"])
		objectID := rs.Primary.Attributes["object_id"]

		found, err := testAccFindDictionaryEntry(client, dictionaryType, objectID)
		if err != nil {
			return err
		}
		if found {
			return fmt.Errorf("dictionary entry %s/%s still exists", dictionaryType, objectID)
		}
	}

	return nil
}

func testAccFindDictionaryEntry(client *search.APIClient, dictionaryType search.DictionaryType, objectID string) (bool, error) {
	page := int32(0)
	for {
		params := search.NewSearchDictionaryEntriesParams(
			"",
			search.WithSearchDictionaryEntriesParamsPage(page),
			search.WithSearchDictionaryEntriesParamsHitsPerPage(1000),
		)

		resp, err := client.SearchDictionaryEntries(client.NewApiSearchDictionaryEntriesRequest(dictionaryType, params))
		if err != nil {
			return false, err
		}

		for _, hit := range resp.Hits {
			if hit.GetObjectID() == objectID {
				return true, nil
			}
		}

		page++
		if len(resp.Hits) == 0 || page >= resp.NbPages {
			return false, nil
		}
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

func testAccDictionaryEntryStopwordConfig(objectID, word, state string) string {
	return fmt.Sprintf(`
resource "algolia_dictionary_entry" "test" {
  dictionary = "stopwords"
  object_id  = %[1]q
  language   = "en"
  word       = %[2]q
  state      = %[3]q
}
`, objectID, word, state)
}

func testAccDictionaryEntryPluralConfig(objectID string) string {
	return fmt.Sprintf(`
resource "algolia_dictionary_entry" "test" {
  dictionary = "plurals"
  object_id  = %[1]q
  language   = "fr"
  words      = ["cheval", "chevaux"]
}
`, objectID)
}

func testAccDictionaryEntryCompoundConfig(objectID string) string {
	return fmt.Sprintf(`
resource "algolia_dictionary_entry" "test" {
  dictionary    = "compounds"
  object_id     = %[1]q
  language      = "de"
  word          = "kopfschmerzen"
  decomposition = ["kopf", "schmerzen"]
}
`, objectID)
}

func testAccDictionaryEntryDataSourceConfig(objectID, word string) string {
	return testAccDictionaryEntryStopwordConfig(objectID, word, "enabled") + `

data "algolia_dictionary_entry" "test" {
  dictionary = algolia_dictionary_entry.test.dictionary
  object_id  = algolia_dictionary_entry.test.object_id
}
`
}
