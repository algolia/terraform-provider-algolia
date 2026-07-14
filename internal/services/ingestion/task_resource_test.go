package ingestion_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/algolia/terraform-provider-algolia/internal/analyticsregion"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
				// Toggling enabled and adding a cron schedule exercises
				// UpdateTask rather than a replace, since only source_id
				// and action are RequiresReplace.
				Config: testAccIngestionTaskConfig(sourceName, destinationName, indexName, false, "0 0 * * *"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_ingestion_task.test", "enabled", "false"),
					resource.TestCheckResourceAttr("algolia_ingestion_task.test", "cron", "0 0 * * *"),
				),
			},
			{
				ResourceName:      "algolia_ingestion_task.test",
				ImportState:       true,
				ImportStateVerify: true,
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
					resource.TestCheckResourceAttr("data.algolia_ingestion_task.test", "action", "replace"),
					resource.TestCheckResourceAttrPair(
						"data.algolia_ingestion_task.test", "task_id",
						"algolia_ingestion_task.test", "task_id",
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

resource "algolia_ingestion_destination" "test" {
  name = %[2]q
  type = "search"

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
`, sourceName, destinationName, indexName, enabled, cronAttr)
}

func testAccIngestionTaskDataSourceConfig(sourceName, destinationName, indexName string) string {
	return testAccIngestionTaskConfig(sourceName, destinationName, indexName, true, "") + `

data "algolia_ingestion_task" "test" {
  task_id = algolia_ingestion_task.test.task_id
}
`
}
