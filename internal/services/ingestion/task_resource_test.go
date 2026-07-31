package ingestion_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/algolia/terraform-provider-algolia/internal/analyticsregion"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccProtoV6ProviderFactories and testAccRequireCredentials are shared
// with authentication_resource_test.go (same ingestion_test package).
//
// A task references a source and a destination by ID, so this test's
// config declares a "push" source and a "search" destination alongside
// the task (the same minimal, dependency-free shapes used by
// source_resource_test.go/destination_resource_test.go). Terraform's own
// destroy step at the end of resource.Test tears down all three; the
// CheckDestroy below additionally verifies none of them survive.

func TestAccIngestionTaskResource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	sourceName := fmt.Sprintf("tf-acc-task-source-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	destinationName := fmt.Sprintf("tf-acc-task-destination-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	indexName := fmt.Sprintf("tf_acc_task_%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIngestionTaskDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIngestionTaskConfig(sourceName, destinationName, indexName, true, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_ingestion_task.test", "action", "replace"),
					resource.TestCheckResourceAttr("algolia_ingestion_task.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("algolia_ingestion_task.test", "task_id"),
					resource.TestCheckResourceAttrSet("algolia_ingestion_task.test", "created_at"),
					resource.TestCheckResourceAttrPair("algolia_ingestion_task.test", "id", "algolia_ingestion_task.test", "task_id"),
					resource.TestCheckResourceAttrPair("algolia_ingestion_task.test", "source_id", "algolia_ingestion_source.test", "source_id"),
					resource.TestCheckResourceAttrPair("algolia_ingestion_task.test", "destination_id", "algolia_ingestion_destination.test", "destination_id"),
				),
			},
			{
				// Toggling enabled exercises UpdateTask rather than a replace,
				// since only source_id and action are RequiresReplace. No cron
				// here: the source is type "push", and the API rejects a
				// schedule on one ("a source of type 'push' isn't able to
				// schedule tasks").
				Config: testAccIngestionTaskConfig(sourceName, destinationName, indexName, false, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_ingestion_task.test", "enabled", "false"),
				),
			},
			{
				ResourceName:      "algolia_ingestion_task.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The Ingestion API returns "action": null on read - verified
				// directly against /2/tasks, including for tasks this provider
				// never touched - so it is write-only in practice and cannot be
				// recovered on import. Read preserves the prior value, but an
				// import has no prior to preserve.
				ImportStateVerifyIgnore: []string{"action"},
			},
		},
	})
}

// TestAccIngestionTaskResource_cronRemovalForcesReplace covers removing a
// schedule. The Ingestion API can set a cron and change it, but has no way to
// clear one - an empty expression is rejected as invalid and an explicit null is
// ignored - so the only route to an on-demand task is creating one without a
// cron. The third step is the one that matters: it proves the configuration
// converges, which is exactly what removing a cron did not do before.
func TestAccIngestionTaskResource_cronRemovalForcesReplace(t *testing.T) {
	testAccRequireCredentials(t)

	suffix := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	sourceName := fmt.Sprintf("tf-acc-cron-source-%s", suffix)
	destinationName := fmt.Sprintf("tf-acc-cron-destination-%s", suffix)
	indexName := fmt.Sprintf("tf_acc_cron_%s", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIngestionTaskDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIngestionTaskCronConfig(sourceName, destinationName, indexName, "0 0 * * *"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_ingestion_task.test", "cron", "0 0 * * *"),
					resource.TestCheckResourceAttrSet("algolia_ingestion_task.test", "next_run"),
				),
			},
			{
				// Changing the schedule is a plain update: the task keeps its ID.
				Config: testAccIngestionTaskCronConfig(sourceName, destinationName, indexName, "30 4 * * 1"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("algolia_ingestion_task.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_ingestion_task.test", "cron", "30 4 * * 1"),
				),
			},
			{
				// Removing it entirely cannot be expressed as an update, so the
				// task has to be replaced.
				Config: testAccIngestionTaskCronConfig(sourceName, destinationName, indexName, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("algolia_ingestion_task.test", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("algolia_ingestion_task.test", "cron"),
				),
			},
			{
				// The convergence check. Without the replacement above, the task
				// would still carry its schedule remotely and this plan would
				// propose the same removal again.
				Config:   testAccIngestionTaskCronConfig(sourceName, destinationName, indexName, ""),
				PlanOnly: true,
			},
		},
	})
}

func TestAccIngestionTaskDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	sourceName := fmt.Sprintf("tf-acc-task-ds-source-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	destinationName := fmt.Sprintf("tf-acc-task-ds-destination-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	indexName := fmt.Sprintf("tf_acc_task_ds_%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIngestionTaskDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIngestionTaskDataSourceConfig(sourceName, destinationName, indexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Not `action`: the Ingestion API returns it as null on read,
					// so a data source cannot surface it. Assert the fields the
					// API does return instead.
					resource.TestCheckResourceAttr("data.algolia_ingestion_task.test", "enabled", "true"),
					resource.TestCheckResourceAttrPair(
						"data.algolia_ingestion_task.test", "task_id",
						"algolia_ingestion_task.test", "task_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.algolia_ingestion_task.test", "source_id",
						"algolia_ingestion_source.test", "source_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.algolia_ingestion_task.test", "destination_id",
						"algolia_ingestion_destination.test", "destination_id",
					),
				),
			},
		},
	})
}

func testAccCheckIngestionTaskDestroy(s *terraform.State) error {
	client, err := analyticsregion.NewIngestionClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"), os.Getenv(analyticsregion.EnvVar))
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "algolia_ingestion_task":
			taskID := rs.Primary.Attributes["task_id"]
			if _, err := client.GetTask(client.NewApiGetTaskRequest(taskID)); err == nil {
				return fmt.Errorf("ingestion task %s still exists", taskID)
			}
		case "algolia_ingestion_source":
			sourceID := rs.Primary.Attributes["source_id"]
			if _, err := client.GetSource(client.NewApiGetSourceRequest(sourceID)); err == nil {
				return fmt.Errorf("ingestion source %s still exists", sourceID)
			}
		case "algolia_ingestion_destination":
			destinationID := rs.Primary.Attributes["destination_id"]
			if _, err := client.GetDestination(client.NewApiGetDestinationRequest(destinationID)); err == nil {
				return fmt.Errorf("ingestion destination %s still exists", destinationID)
			}
		}
	}

	return nil
}

func testAccIngestionTaskConfig(sourceName, destinationName, indexName string, enabled bool, cron string) string {
	cronAttr := ""
	if cron != "" {
		cronAttr = fmt.Sprintf("cron = %q", cron)
	}

	return fmt.Sprintf(`
resource "algolia_ingestion_source" "test" {
  name = %[1]q
  type = "push"
}

resource "algolia_ingestion_authentication" "test" {
  name = "%[2]s-auth"
  type = "algolia"

  input = jsonencode({
    appID  = %[6]q
    apiKey = %[7]q
  })
}

resource "algolia_ingestion_destination" "test" {
  name              = %[2]q
  type              = "search"
  authentication_id = algolia_ingestion_authentication.test.authentication_id

  input = jsonencode({
    indexName = %[3]q
  })
}

resource "algolia_ingestion_task" "test" {
  source_id      = algolia_ingestion_source.test.source_id
  destination_id = algolia_ingestion_destination.test.destination_id
  action         = "replace"
  enabled        = %[4]t
  %[5]s
}
`, sourceName, destinationName, indexName, enabled, cronAttr, os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
}

// testAccIngestionTaskCronConfig uses a "csv" source rather than the "push" one
// the other task tests use: the API refuses to schedule a task on a push source,
// so a cron test needs a pull-based one. The URL is never fetched - nothing here
// runs the task - but a "search" destination does need an "algolia"
// authentication whose appID matches the requesting application.
func testAccIngestionTaskCronConfig(sourceName, destinationName, indexName, cron string) string {
	cronAttr := ""
	if cron != "" {
		cronAttr = fmt.Sprintf("cron = %q", cron)
	}

	return fmt.Sprintf(`
resource "algolia_ingestion_source" "test" {
  name = %[1]q
  type = "csv"

  input = jsonencode({
    url            = "https://example.com/products.csv"
    uniqueIDColumn = "id"
  })
}

resource "algolia_ingestion_authentication" "test" {
  name = "%[2]s-auth"
  type = "algolia"

  input = jsonencode({
    appID  = %[5]q
    apiKey = %[6]q
  })
}

resource "algolia_ingestion_destination" "test" {
  name              = %[2]q
  type              = "search"
  authentication_id = algolia_ingestion_authentication.test.authentication_id

  input = jsonencode({
    indexName = %[3]q
  })
}

resource "algolia_ingestion_task" "test" {
  source_id      = algolia_ingestion_source.test.source_id
  destination_id = algolia_ingestion_destination.test.destination_id
  action         = "replace"
  %[4]s
}
`, sourceName, destinationName, indexName, cronAttr, os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
}

func testAccIngestionTaskDataSourceConfig(sourceName, destinationName, indexName string) string {
	return testAccIngestionTaskConfig(sourceName, destinationName, indexName, true, "") + `

data "algolia_ingestion_task" "test" {
  task_id = algolia_ingestion_task.test.task_id
}
`
}
