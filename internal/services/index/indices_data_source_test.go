package index_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccIndicesDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-test-indices-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIndicesDataSourceConfig(indexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.algolia_indices.test", "id"),
					testAccCheckIndicesDataSourceContains("data.algolia_indices.test", indexName),
				),
			},
		},
	})
}

func testAccIndicesDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false
}

data "algolia_indices" "test" {
  depends_on = [algolia_index.test]
}
`, name)
}

// testAccCheckIndicesDataSourceContains verifies that the given index name
// appears somewhere in the algolia_indices listing. The listing has no
// filtering, and other indices may exist on the application under test, so
// this scans every returned entry rather than asserting an exact count.
func testAccCheckIndicesDataSourceContains(dataSourceName, wantName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ds, ok := s.RootModule().Resources[dataSourceName]
		if !ok {
			return fmt.Errorf("data source %s not found in state", dataSourceName)
		}

		count, err := strconv.Atoi(ds.Primary.Attributes["indices.#"])
		if err != nil {
			return fmt.Errorf("could not parse indices.#: %w", err)
		}

		for i := 0; i < count; i++ {
			if ds.Primary.Attributes[fmt.Sprintf("indices.%d.name", i)] == wantName {
				return nil
			}
		}

		return fmt.Errorf("expected index %s to be present in algolia_indices listing (%d indices returned)", wantName, count)
	}
}
