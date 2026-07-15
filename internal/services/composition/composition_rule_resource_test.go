package composition_test

import (
	"fmt"
	"os"
	"testing"

	compositionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/composition"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccCompositionRuleResource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-composition-rule-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	compositionID := fmt.Sprintf("tf-composition-rule-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	objectID := "promote-featured"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCompositionRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCompositionRuleResourceConfig(indexName, compositionID, objectID, `[{"filters":"brand:apple"}]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_composition_rule.test", "composition_id", compositionID),
					resource.TestCheckResourceAttr("algolia_composition_rule.test", "object_id", objectID),
					resource.TestCheckResourceAttr("algolia_composition_rule.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("algolia_composition_rule.test", "consequence"),
				),
			},
			{
				Config: testAccCompositionRuleResourceConfig(indexName, compositionID, objectID, `[{"filters":"brand:samsung"}]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_composition_rule.test", "composition_id", compositionID),
				),
			},
			{
				ResourceName:      "algolia_composition_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccCompositionRuleResource_generatedObjectID(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-composition-rule-gen-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	compositionID := fmt.Sprintf("tf-composition-rule-gen-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCompositionRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_composition" "test" {
  object_id = %[2]q
  name      = "Test composition"

  behavior = jsonencode({
    injection = {
      main = {
        source = {
          search = {
            index = algolia_index.test.name
          }
        }
      }
    }
  })
}

resource "algolia_composition_rule" "test" {
  composition_id = algolia_composition.test.object_id

  consequence = jsonencode({
    behavior = {
      injection = {
        main = {
          source = {
            search = {
              index = algolia_index.test.name
            }
          }
        }
      }
    }
  })
}
`, indexName, compositionID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("algolia_composition_rule.test", "object_id"),
				),
			},
		},
	})
}

func TestAccCompositionRuleDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	indexName := fmt.Sprintf("tf-composition-rule-ds-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	compositionID := fmt.Sprintf("tf-composition-rule-ds-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	objectID := "promote-featured"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCompositionRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCompositionRuleDataSourceConfig(indexName, compositionID, objectID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.algolia_composition_rule.test", "composition_id", compositionID),
					resource.TestCheckResourceAttr("data.algolia_composition_rule.test", "object_id", objectID),
					resource.TestCheckResourceAttrSet("data.algolia_composition_rule.test", "consequence"),
				),
			},
		},
	})
}

func testAccCheckCompositionRuleDestroy(s *terraform.State) error {
	client, err := compositionapi.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_composition_rule" {
			continue
		}

		compositionID := rs.Primary.Attributes["composition_id"]
		objectID := rs.Primary.Attributes["object_id"]

		if _, err := client.GetRule(client.NewApiGetRuleRequest(compositionID, objectID)); err == nil {
			return fmt.Errorf("composition rule %s/%s still exists", compositionID, objectID)
		}
	}

	return nil
}

func testAccCompositionRuleResourceConfig(indexName, compositionID, objectID, conditionsJSON string) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_composition" "test" {
  object_id = %[2]q
  name      = "Test composition"

  behavior = jsonencode({
    injection = {
      main = {
        source = {
          search = {
            index = algolia_index.test.name
          }
        }
      }
    }
  })
}

resource "algolia_composition_rule" "test" {
  composition_id = algolia_composition.test.object_id
  object_id       = %[3]q
  description     = "promote featured products"
  enabled         = true

  conditions = %[4]q

  consequence = jsonencode({
    behavior = {
      injection = {
        main = {
          source = {
            search = {
              index = algolia_index.test.name
            }
          }
        }
      }
    }
  })
}
`, indexName, compositionID, objectID, conditionsJSON)
}

func testAccCompositionRuleDataSourceConfig(indexName, compositionID, objectID string) string {
	return testAccCompositionRuleResourceConfig(indexName, compositionID, objectID, `[{"filters":"brand:apple"}]`) + `

data "algolia_composition_rule" "test" {
  composition_id = algolia_composition_rule.test.composition_id
  object_id       = algolia_composition_rule.test.object_id
}
`
}
