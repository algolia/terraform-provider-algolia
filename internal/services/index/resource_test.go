package index_test

import (
	"fmt"
	"os"
	"regexp"
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

func testAccCheckIndexDestroy(s *terraform.State) error {
	client, _ := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_index" {
			continue
		}
		_, err := client.GetSettings(client.NewApiGetSettingsRequest(rs.Primary.ID))
		if err == nil {
			return fmt.Errorf("index %s still exists", rs.Primary.ID)
		}
	}
	return nil
}

func TestAccIndexResource_basic(t *testing.T) {
	indexName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIndexResourceConfig_basic(indexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_index.test", "name", indexName),
					resource.TestCheckResourceAttr("algolia_index.test", "deletion_protection", "false"),
				),
			},
		},
	})
}

func TestAccIndexResource_fullSettings(t *testing.T) {
	indexName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIndexResourceConfig_fullSettings(indexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_index.test", "name", indexName),
					resource.TestCheckResourceAttr("algolia_index.test", "deletion_protection", "false"),
					// Attributes block
					resource.TestCheckResourceAttr("algolia_index.test", "attributes.searchable_attributes.0", "title"),
					resource.TestCheckResourceAttr("algolia_index.test", "attributes.searchable_attributes.1", "description"),
					resource.TestCheckResourceAttr("algolia_index.test", "attributes.attributes_to_retrieve.0", "title"),
					resource.TestCheckResourceAttr("algolia_index.test", "attributes.attributes_to_retrieve.1", "description"),
					// Ranking block
					resource.TestCheckResourceAttr("algolia_index.test", "ranking.custom_ranking.0", "desc(popularity)"),
					resource.TestCheckResourceAttr("algolia_index.test", "ranking.relevancy_strictness", "90"),
					// Faceting block
					resource.TestCheckResourceAttr("algolia_index.test", "faceting.attributes_for_faceting.0", "category"),
					resource.TestCheckResourceAttr("algolia_index.test", "faceting.max_values_per_facet", "50"),
					resource.TestCheckResourceAttr("algolia_index.test", "faceting.sort_facet_values_by", "count"),
					// Highlighting block
					resource.TestCheckResourceAttr("algolia_index.test", "highlighting.highlight_pre_tag", "<em>"),
					resource.TestCheckResourceAttr("algolia_index.test", "highlighting.highlight_post_tag", "</em>"),
					// Pagination block
					resource.TestCheckResourceAttr("algolia_index.test", "pagination.hits_per_page", "25"),
					resource.TestCheckResourceAttr("algolia_index.test", "pagination.pagination_limited_to", "500"),
					// Typos block
					resource.TestCheckResourceAttr("algolia_index.test", "typos.typo_tolerance", "true"),
					resource.TestCheckResourceAttr("algolia_index.test", "typos.min_word_size_for_1_typo", "4"),
					resource.TestCheckResourceAttr("algolia_index.test", "typos.min_word_size_for_2_typos", "8"),
					// Languages block
					resource.TestCheckResourceAttr("algolia_index.test", "languages.query_languages.0", "en"),
					resource.TestCheckResourceAttr("algolia_index.test", "languages.remove_words_if_no_results", "lastWords"),
					// Query strategy block
					resource.TestCheckResourceAttr("algolia_index.test", "query_strategy.query_type", "prefixLast"),
					resource.TestCheckResourceAttr("algolia_index.test", "query_strategy.exact_on_single_word_query", "attribute"),
					// Performance block
					resource.TestCheckResourceAttr("algolia_index.test", "performance.allow_compression_of_integer_array", "false"),
					// Advanced block
					resource.TestCheckResourceAttr("algolia_index.test", "advanced.distinct", "1"),
					resource.TestCheckResourceAttr("algolia_index.test", "advanced.min_proximity", "1"),
					resource.TestCheckResourceAttr("algolia_index.test", "advanced.enable_rules", "true"),
				),
			},
		},
	})
}

func TestAccIndexResource_update(t *testing.T) {
	indexName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIndexResourceConfig_update_step1(indexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_index.test", "name", indexName),
					resource.TestCheckResourceAttr("algolia_index.test", "pagination.hits_per_page", "20"),
				),
			},
			{
				Config: testAccIndexResourceConfig_update_step2(indexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_index.test", "name", indexName),
					resource.TestCheckResourceAttr("algolia_index.test", "pagination.hits_per_page", "50"),
					resource.TestCheckResourceAttr("algolia_index.test", "attributes.searchable_attributes.0", "title"),
					resource.TestCheckResourceAttr("algolia_index.test", "attributes.searchable_attributes.1", "body"),
				),
			},
		},
	})
}

func TestAccIndexResource_deletionProtection(t *testing.T) {
	indexName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIndexResourceConfig_deletionProtection(indexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_index.test", "name", indexName),
					resource.TestCheckResourceAttr("algolia_index.test", "deletion_protection", "true"),
				),
			},
			{
				Config:      testAccIndexResourceConfig_deletionProtection(indexName),
				Destroy:     true,
				ExpectError: regexp.MustCompile("Deletion Protection Enabled"),
			},
			{
				// Disable deletion protection so the test can clean up.
				Config: testAccIndexResourceConfig_basic(indexName),
			},
		},
	})
}

func TestAccIndexResource_import(t *testing.T) {
	indexName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIndexResourceConfig_basic(indexName),
			},
			{
				ResourceName:                         "algolia_index.test",
				ImportState:                          true,
				ImportStateId:                        indexName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerifyIgnore:              []string{"deletion_protection"},
			},
		},
	})
}

func TestAccIndexResource_importFullSettings(t *testing.T) {
	indexName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIndexResourceConfig_fullSettings(indexName),
			},
			{
				ResourceName:  "algolia_index.test",
				ImportState:   true,
				ImportStateId: indexName,
				ImportStateVerifyIdentifierAttribute: "name",
			},
			{
				Config: testAccIndexResourceConfig_fullSettings(indexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_index.test", "attributes.searchable_attributes.0", "title"),
					resource.TestCheckResourceAttr("algolia_index.test", "ranking.custom_ranking.0", "desc(popularity)"),
					resource.TestCheckResourceAttr("algolia_index.test", "faceting.attributes_for_faceting.0", "category"),
					resource.TestCheckResourceAttr("algolia_index.test", "highlighting.highlight_pre_tag", "<em>"),
					resource.TestCheckResourceAttr("algolia_index.test", "pagination.hits_per_page", "25"),
					resource.TestCheckResourceAttr("algolia_index.test", "typos.typo_tolerance", "true"),
					resource.TestCheckResourceAttr("algolia_index.test", "languages.query_languages.0", "en"),
					resource.TestCheckResourceAttr("algolia_index.test", "query_strategy.query_type", "prefixLast"),
					resource.TestCheckResourceAttr("algolia_index.test", "performance.allow_compression_of_integer_array", "false"),
					resource.TestCheckResourceAttr("algolia_index.test", "advanced.distinct", "1"),
				),
			},
		},
	})
}

func TestAccIndexResource_importJsonFields(t *testing.T) {
	indexName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIndexResourceConfig_jsonFields(indexName),
			},
			{
				ResourceName:  "algolia_index.test",
				ImportState:   true,
				ImportStateId: indexName,
				ImportStateVerifyIdentifierAttribute: "name",
			},
			{
				Config: testAccIndexResourceConfig_jsonFields(indexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("algolia_index.test", "languages.decompounded_attributes"),
					resource.TestCheckResourceAttrSet("algolia_index.test", "languages.custom_normalization"),
					resource.TestCheckResourceAttrSet("algolia_index.test", "advanced.user_data"),
				),
			},
		},
	})
}

func TestAccIndexResource_importUnionTypes(t *testing.T) {
	indexName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIndexResourceConfig_unionTypes(indexName),
			},
			{
				ResourceName:  "algolia_index.test",
				ImportState:   true,
				ImportStateId: indexName,
				ImportStateVerifyIdentifierAttribute: "name",
			},
			{
				Config: testAccIndexResourceConfig_unionTypes(indexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_index.test", "typos.typo_tolerance", "min"),
					resource.TestCheckResourceAttr("algolia_index.test", "advanced.distinct", "2"),
					resource.TestCheckResourceAttr("algolia_index.test", "languages.ignore_plurals_languages.#", "2"),
					resource.TestCheckResourceAttr("algolia_index.test", "languages.remove_stop_words_languages.#", "2"),
				),
			},
		},
	})
}

func TestAccIndexResource_importPartialBlocks(t *testing.T) {
	indexName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIndexResourceConfig_update_step1(indexName),
			},
			{
				ResourceName:  "algolia_index.test",
				ImportState:   true,
				ImportStateId: indexName,
				ImportStateVerifyIdentifierAttribute: "name",
			},
			{
				Config: testAccIndexResourceConfig_update_step1(indexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_index.test", "pagination.hits_per_page", "20"),
					resource.TestCheckResourceAttr("algolia_index.test", "attributes.searchable_attributes.0", "title"),
					resource.TestCheckNoResourceAttr("algolia_index.test", "ranking.%"),
					resource.TestCheckNoResourceAttr("algolia_index.test", "faceting.%"),
					resource.TestCheckNoResourceAttr("algolia_index.test", "highlighting.%"),
					resource.TestCheckNoResourceAttr("algolia_index.test", "typos.%"),
					resource.TestCheckNoResourceAttr("algolia_index.test", "languages.%"),
					resource.TestCheckNoResourceAttr("algolia_index.test", "query_strategy.%"),
					resource.TestCheckNoResourceAttr("algolia_index.test", "performance.%"),
					resource.TestCheckNoResourceAttr("algolia_index.test", "advanced.%"),
				),
			},
		},
	})
}

func TestAccIndexResource_importNonexistent(t *testing.T) {
	indexName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIndexResourceConfig_basic(indexName),
			},
			{
				ResourceName:  "algolia_index.test",
				ImportState:   true,
				ImportStateId: "tf-test-nonexistent-index-9999999",
				ExpectError:   regexp.MustCompile(`(?i)error reading index`),
			},
		},
	})
}

func testAccIndexResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false
}
`, name)
}

func testAccIndexResourceConfig_fullSettings(name string) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false

  attributes {
    searchable_attributes    = ["title", "description"]
    attributes_to_retrieve   = ["title", "description"]
    unretrievable_attributes = ["internal_id"]
    attribute_for_distinct   = "url"
  }

  ranking {
    custom_ranking       = ["desc(popularity)"]
    relevancy_strictness = 90
  }

  faceting {
    attributes_for_faceting = ["category"]
    max_values_per_facet    = 50
    sort_facet_values_by    = "count"
  }

  highlighting {
    highlight_pre_tag                     = "<em>"
    highlight_post_tag                    = "</em>"
    snippet_ellipsis_text                 = "..."
    restrict_highlight_and_snippet_arrays = false
  }

  pagination {
    hits_per_page         = 25
    pagination_limited_to = 500
  }

  typos {
    typo_tolerance            = "true"
    min_word_size_for_1_typo  = 4
    min_word_size_for_2_typos = 8
    allow_typos_on_numeric_tokens = true
  }

  languages {
    query_languages          = ["en"]
    remove_words_if_no_results = "lastWords"
    decompound_query         = true
  }

  query_strategy {
    query_type                 = "prefixLast"
    exact_on_single_word_query = "attribute"
  }

  performance {
    allow_compression_of_integer_array = false
  }

  advanced {
    distinct       = 1
    min_proximity  = 1
    enable_rules   = true
  }
}
`, name)
}

func testAccIndexResourceConfig_update_step1(name string) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false

  pagination {
    hits_per_page = 20
  }

  attributes {
    searchable_attributes = ["title"]
  }
}
`, name)
}

func testAccIndexResourceConfig_update_step2(name string) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false

  pagination {
    hits_per_page = 50
  }

  attributes {
    searchable_attributes = ["title", "body"]
  }
}
`, name)
}

func testAccIndexResourceConfig_deletionProtection(name string) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = true
}
`, name)
}

func testAccIndexResourceConfig_jsonFields(name string) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false

  languages {
    decompounded_attributes = jsonencode({ de = ["name", "description"] })
    custom_normalization    = jsonencode({ default = { "ä" = "ae", "ö" = "oe" } })
  }

  advanced {
    user_data = jsonencode({ version = 2, environment = "test", tags = ["a", "b", "c"] })
  }
}
`, name)
}

func testAccIndexResourceConfig_unionTypes(name string) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false

  attributes {
    attribute_for_distinct = "url"
  }

  typos {
    typo_tolerance = "min"
  }

  languages {
    ignore_plurals_languages    = ["en", "fr"]
    remove_stop_words_languages = ["en", "fr"]
  }

  advanced {
    distinct = 2
  }
}
`, name)
}
