package rule_test

import (
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccRuleResource_promoteOrder(t *testing.T) {
	testAccRequireCredentials(t)
	indexName := "tf-rule-order-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	original := []string{"shoe-30", "shoe-12", "shoe-21"}
	reordered := []string{"shoe-21", "shoe-30", "shoe-12"}

	check := func(ids []string) resource.TestCheckFunc {
		checks := []resource.TestCheckFunc{
			func(*terraform.State) error {
				client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
				if err != nil {
					return err
				}
				rule, err := client.GetRule(client.NewApiGetRuleRequest(indexName, "pins"))
				if err != nil {
					return err
				}
				promote := rule.Consequence.Promote
				if len(promote) != 1 || promote[0].PromoteObjectIDs == nil {
					return fmt.Errorf("expected one multi-record promote group, got %#v", promote)
				}
				group := promote[0].PromoteObjectIDs
				if !slices.Equal(group.ObjectIDs, ids) || group.Position != 0 {
					return fmt.Errorf("API promote group = %v at %d, want %v at 0", group.ObjectIDs, group.Position, ids)
				}
				return nil
			},
		}
		for i, id := range ids {
			checks = append(checks, resource.TestCheckResourceAttr("algolia_rule.test", fmt.Sprintf("consequence.0.promote.0.object_ids.%d", i), id))
		}
		return resource.ComposeAggregateTestCheckFunc(checks...)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRulePromoteOrderConfig(indexName, "Original description", original),
				Check:  check(original),
			},
			{
				ResourceName:      "algolia_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccRulePromoteOrderConfig(indexName, "Unrelated update", original),
				Check:  check(original),
			},
			{
				Config: testAccRulePromoteOrderConfig(indexName, "Unrelated update", reordered),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("algolia_rule.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: check(reordered),
			},
		},
	})
}

func testAccRulePromoteOrderConfig(indexName, description string, ids []string) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %q
  deletion_protection = false
}

resource "algolia_rule" "test" {
  index_name  = algolia_index.test.name
  object_id   = "pins"
  description = %q

  conditions {
    pattern   = "running shoes"
    anchoring = "is"
  }

  consequence {
    promote {
      object_ids = [%q, %q, %q]
      position   = 0
    }
  }
}
`, indexName, description, ids[0], ids[1], ids[2])
}
