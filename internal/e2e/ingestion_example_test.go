//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"
	"github.com/algolia/terraform-provider-algolia/internal/analyticsregion"
	testconfig "github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestE2EIngestionExample executes the checked-in ingestion pipeline while a
// temporary override gives every object a unique name, disables its scheduled
// task, and permits cleanup. It proves the example itself composes, without
// fetching the illustrative feed or replacing records in an index.
func TestE2EIngestionExample(t *testing.T) {
	requireE2E(t)
	if os.Getenv(analyticsregion.EnvVar) == "" {
		t.Skip("ALGOLIA_ANALYTICS_REGION must be set for the Ingestion example")
	}
	assertExampleResourceAddresses(t, "ingestion-pipeline", []string{
		"algolia_ingestion_source.products_csv",
		"algolia_ingestion_transformation.stamp_indexed_at",
		"algolia_ingestion_authentication.destination",
		"algolia_ingestion_destination.products_index",
		"algolia_ingestion_task.nightly_sync",
	})

	client, err := analyticsregion.NewIngestionClient(
		os.Getenv("ALGOLIA_APP_ID"),
		os.Getenv("ALGOLIA_API_KEY"),
		os.Getenv(analyticsregion.EnvVar),
	)
	if err != nil {
		t.Fatalf("create Ingestion client: %v", err)
	}

	suffix := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	names := ingestionExampleNames{
		source:         "tf-e2e-example-source-" + suffix,
		transformation: "tf-e2e-example-transformation-" + suffix,
		authentication: "tf-e2e-example-authentication-" + suffix,
		destination:    "tf-e2e-example-destination-" + suffix,
		index:          "tf_e2e_example_ingestion_" + suffix,
	}
	ids := &ingestionExampleIDs{}
	baseDir := copyExampleWithOverride(t, "ingestion-pipeline", ingestionExampleOverride(names, "0 2 * * *"))
	updatedNames := names
	updatedNames.source += "-updated"
	updatedNames.transformation += "-updated"
	updatedNames.authentication += "-updated"
	updatedNames.destination += "-updated"
	updatedDir := copyExampleWithOverride(t, "ingestion-pipeline", ingestionExampleOverride(updatedNames, "0 3 * * *"))

	variables := exampleCredentials()
	variables["analytics_region"] = testconfig.StringVariable(os.Getenv(analyticsregion.EnvVar))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		CheckDestroy:             checkIngestionExampleAbsent(client, ids),
		Steps: []resource.TestStep{
			{
				ConfigDirectory: testconfig.StaticDirectory(baseDir),
				ConfigVariables: variables,
				Check: resource.ComposeAggregateTestCheckFunc(
					captureResourceID("algolia_ingestion_source.products_csv", &ids.source),
					captureResourceID("algolia_ingestion_transformation.stamp_indexed_at", &ids.transformation),
					captureResourceID("algolia_ingestion_authentication.destination", &ids.authentication),
					captureResourceID("algolia_ingestion_destination.products_index", &ids.destination),
					captureResourceID("algolia_ingestion_task.nightly_sync", &ids.task),
					resource.TestCheckResourceAttr("algolia_ingestion_task.nightly_sync", "enabled", "false"),
					checkIngestionTask(client, ids, false, "0 2 * * *"),
				),
			},
			{
				PreConfig:       driftIngestionExample(t, client, ids),
				ConfigDirectory: testconfig.StaticDirectory(baseDir),
				ConfigVariables: variables,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("algolia_ingestion_task.nightly_sync", plancheck.ResourceActionUpdate),
				}},
				Check: checkIngestionTask(client, ids, false, "0 2 * * *"),
			},
			{
				ConfigDirectory: testconfig.StaticDirectory(updatedDir),
				ConfigVariables: variables,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_ingestion_source.products_csv", "name", updatedNames.source),
					resource.TestCheckResourceAttr("algolia_ingestion_transformation.stamp_indexed_at", "name", updatedNames.transformation),
					resource.TestCheckResourceAttr("algolia_ingestion_authentication.destination", "name", updatedNames.authentication),
					resource.TestCheckResourceAttr("algolia_ingestion_destination.products_index", "name", updatedNames.destination),
					checkIngestionTask(client, ids, false, "0 3 * * *"),
				),
			},
		},
	})
}

type ingestionExampleNames struct {
	source         string
	transformation string
	authentication string
	destination    string
	index          string
}

type ingestionExampleIDs struct {
	source         string
	transformation string
	authentication string
	destination    string
	task           string
}

func ingestionExampleOverride(names ingestionExampleNames, cron string) string {
	return fmt.Sprintf(`
resource "algolia_ingestion_source" "products_csv" {
  name                = %q
  deletion_protection = false
}

resource "algolia_ingestion_transformation" "stamp_indexed_at" {
  name                = %q
  deletion_protection = false
}

resource "algolia_ingestion_authentication" "destination" {
  name                = %q
  deletion_protection = false
}

resource "algolia_ingestion_destination" "products_index" {
  name                = %q
  deletion_protection = false

  input = jsonencode({
    indexName = %q
  })
}

resource "algolia_ingestion_task" "nightly_sync" {
  cron                = %q
  enabled             = false
  deletion_protection = false
}
`, names.source, names.transformation, names.authentication, names.destination, names.index, cron)
}

func driftIngestionExample(t *testing.T, client *ingestion.APIClient, ids *ingestionExampleIDs) func() {
	t.Helper()
	return func() {
		t.Helper()
		if ids.task == "" {
			t.Fatal("Ingestion task ID was not captured before drift mutation")
		}
		if _, err := client.EnableTask(client.NewApiEnableTaskRequest(ids.task)); err != nil {
			t.Fatalf("enable Ingestion task out of band: %v", err)
		}
		waitForIngestionTask(t, client, ids.task, true, "0 2 * * *")
	}
}

func checkIngestionTask(client *ingestion.APIClient, ids *ingestionExampleIDs, enabled bool, cron string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if ids.task == "" {
			return fmt.Errorf("Ingestion task ID is empty")
		}
		task, err := client.GetTask(client.NewApiGetTaskRequest(ids.task))
		if err != nil {
			return fmt.Errorf("read Ingestion task: %w", err)
		}
		if task.GetEnabled() != enabled || task.GetCron() != cron {
			return fmt.Errorf("Ingestion task enabled/cron = %t/%q, want %t/%q", task.GetEnabled(), task.GetCron(), enabled, cron)
		}
		return nil
	}
}

func waitForIngestionTask(t *testing.T, client *ingestion.APIClient, taskID string, enabled bool, cron string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		task, err := client.GetTask(client.NewApiGetTaskRequest(taskID))
		if err == nil && task.GetEnabled() == enabled && task.GetCron() == cron {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("Ingestion task mutation did not become visible: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func checkIngestionExampleAbsent(client *ingestion.APIClient, ids *ingestionExampleIDs) resource.TestCheckFunc {
	return func(*terraform.State) error {
		checks := []struct {
			kind string
			id   string
			get  func() error
		}{
			{"task", ids.task, func() error { _, err := client.GetTask(client.NewApiGetTaskRequest(ids.task)); return err }},
			{"destination", ids.destination, func() error {
				_, err := client.GetDestination(client.NewApiGetDestinationRequest(ids.destination))
				return err
			}},
			{"authentication", ids.authentication, func() error {
				_, err := client.GetAuthentication(client.NewApiGetAuthenticationRequest(ids.authentication))
				return err
			}},
			{"transformation", ids.transformation, func() error {
				_, err := client.GetTransformation(client.NewApiGetTransformationRequest(ids.transformation))
				return err
			}},
			{"source", ids.source, func() error { _, err := client.GetSource(client.NewApiGetSourceRequest(ids.source)); return err }},
		}
		for _, check := range checks {
			if check.id == "" {
				return fmt.Errorf("Ingestion %s ID was not captured before destroy", check.kind)
			}
			err := check.get()
			if err == nil {
				return fmt.Errorf("Ingestion %s %s still exists after destroy", check.kind, check.id)
			}
			if !algoliaerr.IsNotFound(err) {
				return fmt.Errorf("unexpected error checking Ingestion %s %s destruction: %w", check.kind, check.id, err)
			}
		}
		return nil
	}
}
