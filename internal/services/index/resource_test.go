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
				ImportStateVerifyIdentifierAttribute: "name",
				// ImportStateVerify is deliberately not used here. The applied state
				// leaves blocks absent from the configuration null, while import has no
				// configuration to consult and therefore populates every block from the
				// API. Both shapes plan clean (the block attributes are Optional+Computed),
				// but they are not attribute-wise identical, so a full comparison would
				// fail for reasons unrelated to correctness. TestAccIndexResource_
				// importFullSettings covers the round-trip with a config that sets every
				// block; the check below covers what matters here: import must actually
				// read settings rather than returning a shell containing only the name.
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 imported state, got %d", len(states))
					}
					if got := states[0].Attributes["pagination.hits_per_page"]; got == "" {
						return fmt.Errorf("imported pagination.hits_per_page is empty; import did not read index settings")
					}
					return nil
				},
			},
		},
	})
}

// TestAccIndexResource_importAppliesDeletionProtectionDefault guards against a
// data-loss regression: deletion_protection is not represented in the Algolia
// API, so if import leaves it null the Delete guard reads null (false) and
// destroys an index the configuration explicitly protected.
func TestAccIndexResource_importAppliesDeletionProtectionDefault(t *testing.T) {
	indexName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		// This test is about delete semantics, so assert the index really is gone
		// at the end rather than trusting the framework's own teardown.
		CheckDestroy: testAccCheckIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIndexResourceConfig_basic(indexName),
			},
			{
				ResourceName:                         "algolia_index.test",
				ImportState:                          true,
				ImportStateId:                        indexName,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 imported state, got %d", len(states))
					}
					if got := states[0].Attributes["deletion_protection"]; got != "true" {
						return fmt.Errorf("imported deletion_protection = %q, want \"true\"; a null/false value lets destroy delete a protected index", got)
					}
					return nil
				},
			},
			{
				// Disable protection so the test can clean up.
				Config: testAccIndexResourceConfig_basic(indexName),
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
				ResourceName:                         "algolia_index.test",
				ImportState:                          true,
				ImportStateId:                        indexName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				// Each of these three is genuinely unrecoverable on import, verified
				// rather than assumed. Keep this list minimal: anything added here stops
				// being checked, which is how the original import defect went unnoticed.
				//
				//   deletion_protection - provider-side guard with no API representation.
				//     Import seeds the safe default (true) while this config sets false;
				//     TestAccIndexResource_importAppliesDeletionProtectionDefault asserts
				//     that seeded value instead of ignoring it outright.
				//
				//   relevancy_strictness, allow_compression_of_integer_array - Algolia
				//     accepts both on SetSettings but omits them from GetSettings. Checked
				//     against the live API: after PUTting relevancyStrictness=90 and
				//     allowCompressionOfIntegerArray=false, neither key appears in the GET
				//     response. preservePlannedValues normally carries them over from
				//     prior state, but import has no prior state to carry them from.
				ImportStateVerifyIgnore: []string{
					"deletion_protection",
					"ranking.relevancy_strictness",
					"performance.allow_compression_of_integer_array",
				},
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
				ResourceName:                         "algolia_index.test",
				ImportState:                          true,
				ImportStateId:                        indexName,
				ImportStateVerifyIdentifierAttribute: "name",
				// This config sets only 2 of the 10 blocks, so imported state (all
				// blocks populated from the API) is legitimately richer than applied
				// state. Assert the JSON-encoded fields survived import instead.
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 imported state, got %d", len(states))
					}
					for _, attr := range []string{"advanced.user_data", "languages.custom_normalization"} {
						if states[0].Attributes[attr] == "" {
							return fmt.Errorf("imported %s is empty; import did not read JSON-encoded settings", attr)
						}
					}
					return nil
				},
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
				ResourceName:                         "algolia_index.test",
				ImportState:                          true,
				ImportStateId:                        indexName,
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
				ResourceName:                         "algolia_index.test",
				ImportState:                          true,
				ImportStateId:                        indexName,
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

// testAccDeleteIndexOutOfBand removes an index behind Terraform's back, the way
// `algolia indices delete` or a dashboard click would.
func testAccDeleteIndexOutOfBand(t *testing.T, indexName string) {
	t.Helper()

	client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		t.Fatalf("could not build Search client: %v", err)
	}

	delResp, err := client.DeleteIndex(client.NewApiDeleteIndexRequest(indexName))
	if err != nil {
		t.Fatalf("could not delete index %s out of band: %v", indexName, err)
	}
	if _, err := client.WaitForTask(indexName, delResp.TaskID); err != nil {
		t.Fatalf("could not confirm out-of-band deletion of index %s: %v", indexName, err)
	}
}

// TestAccIndexResource_recoversFromOutOfBandDeletion covers the case that used to
// wedge the resource: with the index gone, Read raised "Error reading index", so
// plan, apply and destroy all failed and only `terraform state rm` unblocked the
// operator. Read must instead drop the resource from state, leaving a plan that
// recreates it.
func TestAccIndexResource_recoversFromOutOfBandDeletion(t *testing.T) {
	indexName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIndexResourceConfig_basic(indexName),
			},
			{
				// Refreshing against a deleted index must succeed and empty the
				// state, which surfaces as a plan that wants to create it again.
				PreConfig:          func() { testAccDeleteIndexOutOfBand(t, indexName) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
			{
				// And the recreate has to go through rather than failing on a
				// leftover state entry.
				Config: testAccIndexResourceConfig_basic(indexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_index.test", "name", indexName),
					resource.TestCheckResourceAttr("algolia_index.test", "deletion_protection", "false"),
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
