package querysuggestions_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	suggestions "github.com/algolia/algoliasearch-client-go/v4/algolia/query-suggestions"
	"github.com/algolia/terraform-provider-algolia/internal/analyticsregion"
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
				),
			},
			{
				Config: testAccQuerySuggestionsConfig(sourceIndexName, qsIndexName, "desktop"),
				Check:  resource.ComposeAggregateTestCheckFunc(),
			},
			{
				ResourceName:      "algolia_query_suggestions.test",
				ImportState:       true,
				ImportStateId:     qsIndexName,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccQuerySuggestionsResource_omittedOptionalCollections applies a
// configuration that leaves every Optional collection inside source_indices
// out, which the other configurations in this file all set explicitly. Those
// attributes are Optional and not Computed, so their planned value is the
// configuration verbatim (null): writing a known empty collection back after
// apply would make Terraform fail with "Provider produced inconsistent result
// after apply".
func TestAccQuerySuggestionsResource_omittedOptionalCollections(t *testing.T) {
	testAccRequireCredentials(t)

	sourceIndexName := fmt.Sprintf("tf-qs-min-source-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	qsIndexName := fmt.Sprintf("tf-qs-min-index-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckQuerySuggestionsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccQuerySuggestionsMinimalConfig(sourceIndexName, qsIndexName, 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "index_name", qsIndexName),
					resource.TestCheckNoResourceAttr("algolia_query_suggestions.test", "source_indices.0.generate.#"),
					resource.TestCheckNoResourceAttr("algolia_query_suggestions.test", "source_indices.0.analytics_tags.#"),
					resource.TestCheckNoResourceAttr("algolia_query_suggestions.test", "source_indices.0.external.#"),
					resource.TestCheckNoResourceAttr("algolia_query_suggestions.test", "languages.#"),
					resource.TestCheckNoResourceAttr("algolia_query_suggestions.test", "exclude.#"),
				),
			},
			{
				// An Update over the same omitted attributes has to stay
				// consistent too, not just the initial Create.
				Config: testAccQuerySuggestionsMinimalConfig(sourceIndexName, qsIndexName, 20),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "source_indices.0.min_hits", "20"),
					resource.TestCheckNoResourceAttr("algolia_query_suggestions.test", "source_indices.0.generate.#"),
				),
			},
		},
	})
}

// TestAccQuerySuggestionsResource_explicitlyEmptyOptionalCollections is the
// mirror of the test above: an explicit `[]` plans a known empty collection, so
// writing null back after apply would be just as inconsistent.
func TestAccQuerySuggestionsResource_explicitlyEmptyOptionalCollections(t *testing.T) {
	testAccRequireCredentials(t)

	sourceIndexName := fmt.Sprintf("tf-qs-empty-source-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	qsIndexName := fmt.Sprintf("tf-qs-empty-index-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckQuerySuggestionsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccQuerySuggestionsEmptyCollectionsConfig(sourceIndexName, qsIndexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "exclude.#", "0"),
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "languages.#", "0"),
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "source_indices.0.generate.#", "0"),
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "source_indices.0.analytics_tags.#", "0"),
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "source_indices.0.external.#", "0"),
				),
			},
		},
	})
}

// TestAccQuerySuggestionsResource_preservesFullReplacementFields checks that an
// Update which does not mention enable_personalization or
// allow_special_characters keeps the values a previous apply set, since
// UpdateConfig replaces the whole configuration.
func TestAccQuerySuggestionsResource_preservesFullReplacementFields(t *testing.T) {
	testAccRequireCredentials(t)

	sourceIndexName := fmt.Sprintf("tf-qs-full-source-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	qsIndexName := fmt.Sprintf("tf-qs-full-index-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckQuerySuggestionsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccQuerySuggestionsFullReplacementConfig(sourceIndexName, qsIndexName, 10, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "allow_special_characters", "true"),
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "enable_personalization", "true"),
				),
			},
			{
				// Same configuration with both fields removed: the values set
				// above must survive the full-replacement PUT.
				Config: testAccQuerySuggestionsFullReplacementConfig(sourceIndexName, qsIndexName, 20, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "source_indices.0.min_hits", "20"),
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "allow_special_characters", "true"),
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "enable_personalization", "true"),
				),
			},
		},
	})
}

func TestAccQuerySuggestionsResource_drift(t *testing.T) {
	testAccRequireCredentials(t)

	sourceIndexName := fmt.Sprintf("tf-qs-drift-source-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	qsIndexName := fmt.Sprintf("tf-qs-drift-index-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckQuerySuggestionsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccQuerySuggestionsConfig(sourceIndexName, qsIndexName, "mobile"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "exclude.0", "free"),
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "source_indices.0.analytics_tags.0", "mobile"),
				),
			},
			{
				PreConfig: testAccMutateQuerySuggestions(t, sourceIndexName, qsIndexName, "drifted", "tablet"),
				Config:    testAccQuerySuggestionsConfig(sourceIndexName, qsIndexName, "mobile"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "exclude.0", "free"),
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "source_indices.0.analytics_tags.0", "mobile"),
				),
			},
		},
	})
}

func TestAccQuerySuggestionsDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	sourceIndexName := fmt.Sprintf("tf-qs-ds-source-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	qsIndexName := fmt.Sprintf("tf-qs-ds-index-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckQuerySuggestionsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccQuerySuggestionsDataSourceConfig(sourceIndexName, qsIndexName, "mobile"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.algolia_query_suggestions.test", "index_name", qsIndexName),
					resource.TestCheckResourceAttr("data.algolia_query_suggestions.test", "source_indices.0.analytics_tags.0", "mobile"),
				),
			},
		},
	})
}

func testAccCheckQuerySuggestionsDestroy(s *terraform.State) error {
	client, err := analyticsregion.NewQuerySuggestionsClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"), os.Getenv(analyticsregion.EnvVar))
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

	if os.Getenv("ALGOLIA_APP_ID") == "" || os.Getenv("ALGOLIA_API_KEY") == "" || os.Getenv(analyticsregion.EnvVar) == "" {
		t.Skip("ALGOLIA_APP_ID, ALGOLIA_API_KEY, and ALGOLIA_ANALYTICS_REGION must be set for acceptance tests")
	}
}

func testAccMutateQuerySuggestions(t *testing.T, sourceIndexName, qsIndexName, exclude, analyticsTag string) func() {
	t.Helper()

	return func() {
		t.Helper()

		client, err := analyticsregion.NewQuerySuggestionsClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"), os.Getenv(analyticsregion.EnvVar))
		if err != nil {
			t.Fatalf("create Query Suggestions client: %v", err)
		}

		config := suggestions.NewConfiguration([]suggestions.SourceIndex{
			*suggestions.NewSourceIndex(
				sourceIndexName,
				suggestions.WithSourceIndexAnalyticsTags([]string{analyticsTag}),
				suggestions.WithSourceIndexMinHits(10),
				suggestions.WithSourceIndexMinLetters(3),
				suggestions.WithSourceIndexExternal([]string{"external_products"}),
				suggestions.WithSourceIndexFacets([]suggestions.Facet{
					*suggestions.NewFacet(
						suggestions.WithFacetAttribute("brand"),
						suggestions.WithFacetAmount(5),
					),
				}),
				suggestions.WithSourceIndexGenerate([][]string{{"brand", "category"}}),
			),
		})
		config.SetLanguages(suggestions.ArrayOfStringAsLanguages([]string{"en"}))
		config.SetExclude([]string{exclude})

		if _, err := client.UpdateConfig(client.NewApiUpdateConfigRequest(qsIndexName, config)); err != nil {
			t.Fatalf("mutate Query Suggestions config %s: %v", qsIndexName, err)
		}

		testAccWaitForQuerySuggestionsMutation(t, client, qsIndexName, exclude, analyticsTag)
	}
}

func testAccWaitForQuerySuggestionsMutation(t *testing.T, client *suggestions.APIClient, qsIndexName, exclude, analyticsTag string) {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	for {
		config, err := client.GetConfig(client.NewApiGetConfigRequest(qsIndexName))
		if err == nil && len(config.GetExclude()) > 0 && config.GetExclude()[0] == exclude {
			sourceIndices := config.GetSourceIndices()
			if len(sourceIndices) > 0 {
				tags := sourceIndices[0].GetAnalyticsTags()
				if len(tags) > 0 && tags[0] == analyticsTag {
					return
				}
			}
		}

		if time.Now().After(deadline) {
			t.Fatalf("Query Suggestions config %s mutation did not become visible", qsIndexName)
		}
		time.Sleep(500 * time.Millisecond)
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

// testAccQuerySuggestionsMinimalConfig omits every Optional collection, both
// on the resource and inside the source_indices block.
func testAccQuerySuggestionsMinimalConfig(sourceIndexName, qsIndexName string, minHits int) string {
	return fmt.Sprintf(`
resource "algolia_index" "source" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_query_suggestions" "test" {
  index_name = %[2]q

  source_indices {
    index_name = algolia_index.source.name
    min_hits   = %[3]d
  }
}
`, sourceIndexName, qsIndexName, minHits)
}

// testAccQuerySuggestionsEmptyCollectionsConfig sets every Optional collection
// to an explicit empty value.
func testAccQuerySuggestionsEmptyCollectionsConfig(sourceIndexName, qsIndexName string) string {
	return fmt.Sprintf(`
resource "algolia_index" "source" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_query_suggestions" "test" {
  index_name = %[2]q
  languages  = []
  exclude    = []

  source_indices {
    index_name     = algolia_index.source.name
    min_hits       = 10
    analytics_tags = []
    external       = []
    generate       = []
  }
}
`, sourceIndexName, qsIndexName)
}

// testAccQuerySuggestionsFullReplacementConfig sets enable_personalization and
// allow_special_characters only when withFullReplacementFields is true, so the
// second apply can check that removing them from configuration does not wipe
// them from the Algolia side.
func testAccQuerySuggestionsFullReplacementConfig(sourceIndexName, qsIndexName string, minHits int, withFullReplacementFields bool) string {
	fields := ""
	if withFullReplacementFields {
		fields = `
  enable_personalization   = true
  allow_special_characters = true
`
	}

	return fmt.Sprintf(`
resource "algolia_index" "source" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_query_suggestions" "test" {
  index_name = %[2]q
%[4]s
  source_indices {
    index_name = algolia_index.source.name
    min_hits   = %[3]d
  }
}
`, sourceIndexName, qsIndexName, minHits, fields)
}

func testAccQuerySuggestionsDataSourceConfig(sourceIndexName, qsIndexName, analyticsTag string) string {
	return testAccQuerySuggestionsConfig(sourceIndexName, qsIndexName, analyticsTag) + `

data "algolia_query_suggestions" "test" {
  index_name = algolia_query_suggestions.test.index_name
}
`
}
