package abtest_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/algolia/terraform-provider-algolia/internal/analyticsregion"
	"github.com/algolia/terraform-provider-algolia/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// The A/B Testing API is region-routed (see internal/analyticsregion), so
// these acceptance tests require ALGOLIA_ANALYTICS_REGION on top of the
// usual TF_ACC/ALGOLIA_APP_ID/ALGOLIA_API_KEY. Creating an A/B test also
// has cost/quota implications for the target application, so these tests
// are additionally gated behind ALGOLIA_RUN_ABTESTING_ACC=1 - mirroring
// how the Personalization acceptance tests are gated behind
// ALGOLIA_RUN_PERSONALIZATION_ACC (see AGENTS.md).

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"algolia": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestAccABTestResource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	name := fmt.Sprintf("tf-acc-abtest-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	endAt := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckABTestDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccABTestResourceConfig(name, endAt),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_ab_test.test", "name", name),
					resource.TestCheckResourceAttr("algolia_ab_test.test", "end_at", endAt),
					resource.TestCheckResourceAttrSet("algolia_ab_test.test", "ab_test_id"),
					resource.TestCheckResourceAttrSet("algolia_ab_test.test", "status"),
					resource.TestCheckResourceAttrPair("algolia_ab_test.test", "id", "algolia_ab_test.test", "ab_test_id"),
				),
			},
			{
				ResourceName:      "algolia_ab_test.test",
				ImportState:       true,
				ImportStateVerify: true,
				// `variants` is deliberately no longer ignored: import used to echo the
				// enriched read shape and could not match a configuration, and verifying
				// it is what proves it now emits the create shape.
				//
				// `metrics` still is, for a reason verified against the API rather than
				// assumed. It is rebuilt from the per-variant metric *results*, and those
				// only exist once a test has gathered data: a test created seconds ago
				// reports `"metrics": null` on every variant. This test necessarily
				// creates one, so there is nothing here to rebuild from. Importing a test
				// that has been running does recover it, which is the case that matters.
				ImportStateVerifyIgnore: []string{"metrics"},
			},
			{
				// A real change must still replace. suppressEquivalentJSON runs ahead of
				// RequiresReplace and aligns the planned value with state when the two
				// documents carry the same data, so the failure mode to guard against is
				// it swallowing an edit the operator meant. Changing a metric name is a
				// genuine difference, and the plan has to say so.
				Config: testAccABTestResourceConfigMetrics(name, endAt, `[{ name = "conversionRate" }]`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("algolia_ab_test.test", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.TestCheckResourceAttr("algolia_ab_test.test", "metrics", `[{"name":"conversionRate"}]`),
			},
			{
				// ...and reformatting the same document must not. Same data, different
				// whitespace and key order: the plan has to be empty.
				Config: testAccABTestResourceConfigMetrics(name, endAt, "[{\n    name = \"conversionRate\"\n  }]"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// testAccABTestResourceConfigMetrics is testAccABTestResourceConfig with the
// metrics document under the caller's control, so a step can change it and assert
// on the resulting plan.
func testAccABTestResourceConfigMetrics(name, endAt, metrics string) string {
	return fmt.Sprintf(`
resource "algolia_index" "control" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_index" "variant" {
  name                = "%[1]s-variant"
  deletion_protection = false
}

resource "algolia_ab_test" "test" {
  name   = %[1]q
  end_at = %[2]q

  variants = jsonencode([
    {
      index             = algolia_index.control.name
      trafficPercentage = 50
    },
    {
      index             = algolia_index.variant.name
      trafficPercentage = 50
    },
  ])

  metrics = jsonencode(%[3]s)
}
`, name, endAt, metrics)
}

func TestAccABTestDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	name := fmt.Sprintf("tf-acc-abtest-ds-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	endAt := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckABTestDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccABTestDataSourceConfig(name, endAt),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.algolia_ab_test.test", "name", name),
					resource.TestCheckResourceAttrSet("data.algolia_ab_test.test", "status"),
					resource.TestCheckResourceAttrSet("data.algolia_ab_test.test", "variants"),
					resource.TestCheckResourceAttrPair(
						"data.algolia_ab_test.test", "ab_test_id",
						"algolia_ab_test.test", "ab_test_id",
					),
				),
			},
		},
	})
}

func testAccCheckABTestDestroy(s *terraform.State) error {
	client, err := analyticsregion.NewABTestingClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"), os.Getenv(analyticsregion.EnvVar))
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_ab_test" {
			continue
		}

		abTestID, err := strconv.ParseInt(rs.Primary.Attributes["ab_test_id"], 10, 32)
		if err != nil {
			return err
		}

		if _, err := client.GetABTest(client.NewApiGetABTestRequest(int32(abTestID))); err == nil {
			return fmt.Errorf("A/B test %d still exists", abTestID)
		}
	}

	return nil
}

func testAccRequireCredentials(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	if os.Getenv("ALGOLIA_RUN_ABTESTING_ACC") != "1" {
		t.Skip("Set ALGOLIA_RUN_ABTESTING_ACC=1 to run A/B Testing acceptance tests; creating A/B tests has cost/quota implications")
	}

	if os.Getenv("ALGOLIA_APP_ID") == "" || os.Getenv("ALGOLIA_API_KEY") == "" || os.Getenv(analyticsregion.EnvVar) == "" {
		t.Skip("ALGOLIA_APP_ID, ALGOLIA_API_KEY, and ALGOLIA_ANALYTICS_REGION must be set for acceptance tests")
	}
}

func testAccABTestResourceConfig(name, endAt string) string {
	return fmt.Sprintf(`
resource "algolia_index" "control" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_index" "variant" {
  name                = "%[1]s-variant"
  deletion_protection = false
}

resource "algolia_ab_test" "test" {
  name   = %[1]q
  end_at = %[2]q

  variants = jsonencode([
    {
      index             = algolia_index.control.name
      trafficPercentage = 50
    },
    {
      index             = algolia_index.variant.name
      trafficPercentage = 50
    },
  ])

  metrics = jsonencode([
    { name = "addToCartRate" },
  ])
}
`, name, endAt)
}

func testAccABTestDataSourceConfig(name, endAt string) string {
	return testAccABTestResourceConfig(name, endAt) + `

data "algolia_ab_test" "test" {
  ab_test_id = algolia_ab_test.test.ab_test_id
}
`
}
