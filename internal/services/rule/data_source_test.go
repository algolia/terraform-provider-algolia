package rule_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccRuleDataSource_promotedIDsAfterUpdate(t *testing.T) {
	testAccRequireCredentials(t)
	indexName := "tf-rule-ds-update-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	original := []string{"shoe-30", "shoe-12", "shoe-21"}
	updated := []string{"shoe-21", "shoe-40", "shoe-12"}

	config := func(ids []string) string {
		return testAccRulePromoteOrderConfig(indexName, "Promoted records", ids) + `
data "algolia_rule" "test" {
  index_name = algolia_rule.test.index_name
  object_id  = algolia_rule.test.object_id
  depends_on = [algolia_rule.test]
}

output "promoted_ids" {
  value = join(",", data.algolia_rule.test.consequence[0].promote[0].object_ids)
}

output "condition_pattern" {
  value = data.algolia_rule.test.conditions[0].pattern
}
`
	}
	check := func(ids []string) resource.TestCheckFunc {
		return resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckOutput("promoted_ids", fmt.Sprintf("%s,%s,%s", ids[0], ids[1], ids[2])),
			resource.TestCheckOutput("condition_pattern", "running shoes"),
			resource.TestCheckResourceAttr("data.algolia_rule.test", "consequence.0.promote.0.object_ids.1", ids[1]),
		)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRuleDestroy,
		Steps: []resource.TestStep{
			{Config: config(original), Check: check(original)},
			{
				Config: config(updated),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("algolia_rule.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: check(updated),
			},
		},
	})
}
