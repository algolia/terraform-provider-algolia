package querysuggestions_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccQuerySuggestionsResource_allLanguages exercises the boolean arm of the
// API's `languages` union, which the provider models as `all_languages`. Both
// arms are Optional and not Computed, so each apply also checks that the arm
// left out of the configuration stays absent from state.
func TestAccQuerySuggestionsResource_allLanguages(t *testing.T) {
	testAccRequireCredentials(t)

	sourceIndexName := fmt.Sprintf("tf-qs-lang-source-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	qsIndexName := fmt.Sprintf("tf-qs-lang-index-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckQuerySuggestionsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccQuerySuggestionsLanguagesConfig(sourceIndexName, qsIndexName, "all_languages = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "all_languages", "true"),
					resource.TestCheckNoResourceAttr("algolia_query_suggestions.test", "languages.#"),
				),
			},
			{
				ResourceName:      "algolia_query_suggestions.test",
				ImportState:       true,
				ImportStateId:     qsIndexName,
				ImportStateVerify: true,
			},
			{
				// Switching arms: the boolean has to leave state as the list
				// arrives, since the API can only hold one of the two.
				Config: testAccQuerySuggestionsLanguagesConfig(sourceIndexName, qsIndexName, `languages = ["en"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "languages.0", "en"),
					resource.TestCheckNoResourceAttr("algolia_query_suggestions.test", "all_languages"),
				),
			},
			{
				// And back, so the reverse transition is covered too.
				Config: testAccQuerySuggestionsLanguagesConfig(sourceIndexName, qsIndexName, "all_languages = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_query_suggestions.test", "all_languages", "true"),
					resource.TestCheckNoResourceAttr("algolia_query_suggestions.test", "languages.#"),
				),
			},
		},
	})
}

// TestAccQuerySuggestionsResource_languagesArmsAreMutuallyExclusive checks that
// configuring both arms fails at plan time instead of reaching the API, which
// would silently keep only one of them.
func TestAccQuerySuggestionsResource_languagesArmsAreMutuallyExclusive(t *testing.T) {
	testAccRequireCredentials(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccQuerySuggestionsBothLanguageArmsConfig(),
				// No index or configuration is created: validation rejects the
				// configuration before the plan is walked.
				ExpectError: regexp.MustCompile("Invalid Attribute Combination"),
			},
		},
	})
}

// testAccQuerySuggestionsLanguagesConfig renders a configuration whose only
// variable part is how the `languages` union is expressed.
func testAccQuerySuggestionsLanguagesConfig(sourceIndexName, qsIndexName, languages string) string {
	return fmt.Sprintf(`
resource "algolia_index" "source" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_query_suggestions" "test" {
  index_name = %[2]q
  %[3]s

  source_indices {
    index_name = algolia_index.source.name
    min_hits   = 10
  }
}
`, sourceIndexName, qsIndexName, languages)
}

func testAccQuerySuggestionsBothLanguageArmsConfig() string {
	return `
resource "algolia_query_suggestions" "test" {
  index_name    = "tf-qs-lang-conflict"
  languages     = ["en"]
  all_languages = true

  source_indices {
    index_name = "tf-qs-lang-conflict-source"
  }
}
`
}
