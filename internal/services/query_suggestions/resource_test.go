package query_suggestions_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/algolia/terraform-provider-algolia/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"algolia": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func testAccRegion() string {
	if r := os.Getenv("ALGOLIA_REGION"); r != "" {
		return r
	}
	return "us"
}

func TestAccQuerySuggestionsConfigResource_basic(t *testing.T) {
	indexName := fmt.Sprintf("tf-test-qs-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	sourceIndex := fmt.Sprintf("tf-test-src-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	region := testAccRegion()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccQSConfigResourceConfig_basic(indexName, sourceIndex, region),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_query_suggestions_config.test", "index_name", indexName),
					resource.TestCheckResourceAttr("algolia_query_suggestions_config.test", "region", region),
					resource.TestCheckResourceAttr("algolia_query_suggestions_config.test", "deletion_protection", "false"),
					resource.TestCheckResourceAttr("algolia_query_suggestions_config.test", "source_index.#", "1"),
					resource.TestCheckResourceAttr("algolia_query_suggestions_config.test", "source_index.0.index_name", sourceIndex),
				),
			},
		},
	})
}

func TestAccQuerySuggestionsConfigResource_update(t *testing.T) {
	indexName := fmt.Sprintf("tf-test-qs-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	sourceIndex := fmt.Sprintf("tf-test-src-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	region := testAccRegion()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccQSConfigResourceConfig_basic(indexName, sourceIndex, region),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_query_suggestions_config.test", "enable_personalization", "false"),
				),
			},
			{
				Config: testAccQSConfigResourceConfig_updated(indexName, sourceIndex, region),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_query_suggestions_config.test", "index_name", indexName),
					resource.TestCheckResourceAttr("algolia_query_suggestions_config.test", "enable_personalization", "true"),
					resource.TestCheckResourceAttr("algolia_query_suggestions_config.test", "exclude.#", "1"),
				),
			},
		},
	})
}

func TestAccQuerySuggestionsConfigResource_deletionProtection(t *testing.T) {
	indexName := fmt.Sprintf("tf-test-qs-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	sourceIndex := fmt.Sprintf("tf-test-src-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	region := testAccRegion()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccQSConfigResourceConfig_deletionProtection(indexName, sourceIndex, region),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_query_suggestions_config.test", "deletion_protection", "true"),
				),
			},
			{
				Config:      testAccQSConfigResourceConfig_deletionProtection(indexName, sourceIndex, region),
				Destroy:     true,
				ExpectError: regexp.MustCompile("Deletion Protection Enabled"),
			},
			{
				// Disable deletion protection so the test framework can clean up.
				Config: testAccQSConfigResourceConfig_basic(indexName, sourceIndex, region),
			},
		},
	})
}

func TestAccQuerySuggestionsConfigResource_import(t *testing.T) {
	indexName := fmt.Sprintf("tf-test-qs-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	sourceIndex := fmt.Sprintf("tf-test-src-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	region := testAccRegion()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccQSConfigResourceConfig_basic(indexName, sourceIndex, region),
			},
			{
				ResourceName:            "algolia_query_suggestions_config.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"deletion_protection", "region"},
			},
		},
	})
}

// ---- config helpers ----

func testAccQSConfigResourceConfig_basic(indexName, sourceIndex, region string) string {
	return fmt.Sprintf(`
resource "algolia_query_suggestions_config" "test" {
  index_name          = %[1]q
  region              = %[3]q
  deletion_protection = false

  source_index {
    index_name = %[2]q
  }

  enable_personalization   = false
  allow_special_characters = false
}
`, indexName, sourceIndex, region)
}

func testAccQSConfigResourceConfig_updated(indexName, sourceIndex, region string) string {
	return fmt.Sprintf(`
resource "algolia_query_suggestions_config" "test" {
  index_name          = %[1]q
  region              = %[3]q
  deletion_protection = false

  source_index {
    index_name = %[2]q
    min_hits   = 3
  }

  exclude                  = ["bad_query"]
  enable_personalization   = true
  allow_special_characters = false
}
`, indexName, sourceIndex, region)
}

func testAccQSConfigResourceConfig_deletionProtection(indexName, sourceIndex, region string) string {
	return fmt.Sprintf(`
resource "algolia_query_suggestions_config" "test" {
  index_name          = %[1]q
  region              = %[3]q
  deletion_protection = true

  source_index {
    index_name = %[2]q
  }
}
`, indexName, sourceIndex, region)
}
