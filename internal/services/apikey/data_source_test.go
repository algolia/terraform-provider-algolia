package apikey_test

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccAPIKeyDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	description := fmt.Sprintf("tf-api-key-ds-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAPIKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyDataSourceConfig(description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.algolia_api_key.test", "id", "algolia_api_key.test", "id"),
					resource.TestCheckResourceAttr("data.algolia_api_key.test", "description", description),
					resource.TestCheckResourceAttr("data.algolia_api_key.test", "acl.#", "2"),
					resource.TestCheckResourceAttr("data.algolia_api_key.test", "max_hits_per_query", "100"),
					resource.TestCheckResourceAttrSet("data.algolia_api_key.test", "created_at"),
				),
			},
		},
	})
}

func testAccAPIKeyDataSourceConfig(description string) string {
	return testAccAPIKeyResourceConfig(description, 100) + `

data "algolia_api_key" "test" {
  key = algolia_api_key.test.id
}
`
}

func TestAccAPIKeyDataSource_notFound(t *testing.T) {
	testAccRequireCredentials(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      `data "algolia_api_key" "missing" { key = "tf-acc-does-not-exist-00000000" }`,
				ExpectError: regexp.MustCompile(`API key not found`),
			},
		},
	})
}

func TestAccAPIKeysDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	description := fmt.Sprintf("tf-api-keys-ds-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAPIKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeysDataSourceConfig(description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.algolia_api_keys.test", "id"),
					testAccCheckAPIKeysDataSourceContains("data.algolia_api_keys.test", "algolia_api_key.test"),
				),
			},
		},
	})
}

func testAccAPIKeysDataSourceConfig(description string) string {
	return testAccAPIKeyResourceConfig(description, 100) + `

data "algolia_api_keys" "test" {
  depends_on = [algolia_api_key.test]
}
`
}

// testAccCheckAPIKeysDataSourceContains verifies that the given resource's
// API key value appears somewhere in the algolia_api_keys listing. The
// listing has no filtering, and other keys may exist on the application
// under test, so this scans every returned entry rather than asserting an
// exact count.
func testAccCheckAPIKeysDataSourceContains(dataSourceName, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		wantKey := rs.Primary.ID

		ds, ok := s.RootModule().Resources[dataSourceName]
		if !ok {
			return fmt.Errorf("data source %s not found in state", dataSourceName)
		}

		count, err := strconv.Atoi(ds.Primary.Attributes["keys.#"])
		if err != nil {
			return fmt.Errorf("could not parse keys.#: %w", err)
		}

		for i := 0; i < count; i++ {
			if ds.Primary.Attributes[fmt.Sprintf("keys.%d.value", i)] == wantKey {
				return nil
			}
		}

		return fmt.Errorf("expected key %s to be present in algolia_api_keys listing (%d keys returned)", wantKey, count)
	}
}
