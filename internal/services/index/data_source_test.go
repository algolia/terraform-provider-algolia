package index_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccIndexDataSource_basic(t *testing.T) {
	indexName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIndexDataSourceConfig_basic(indexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.algolia_index.test", "name", indexName),
				),
			},
		},
	})
}

func testAccIndexDataSourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false
}

data "algolia_index" "test" {
  name = algolia_index.test.name
}
`, name)
}
