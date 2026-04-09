package index_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/terraform-provider-algolia/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var testAccVirtualIndexProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"algolia": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestAccVirtualIndexResource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	primaryIndexName := fmt.Sprintf("tf-virtual-primary-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	replicaName := fmt.Sprintf("tf-virtual-replica-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccVirtualIndexProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVirtualIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualIndexResourceConfig(primaryIndexName, replicaName, 80),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "name", replicaName),
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "primary_index_name", primaryIndexName),
				),
			},
			{
				Config: testAccVirtualIndexResourceConfig(primaryIndexName, replicaName, 60),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "primary_index_name", primaryIndexName),
				),
			},
			{
				ResourceName:                         "algolia_virtual_index.test",
				ImportState:                          true,
				ImportStateId:                        replicaName,
				ImportStateVerifyIdentifierAttribute: "name",
			},
			{
				Config: testAccVirtualIndexResourceConfig(primaryIndexName, replicaName, 60),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "name", replicaName),
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "primary_index_name", primaryIndexName),
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "ranking.relevancy_strictness", "60"),
				),
			},
		},
	})
}

func TestAccVirtualIndexResource_drift(t *testing.T) {
	testAccRequireCredentials(t)

	primaryIndexName := fmt.Sprintf("tf-virtual-drift-primary-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	replicaName := fmt.Sprintf("tf-virtual-drift-replica-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccVirtualIndexProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVirtualIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualIndexResourceConfig(primaryIndexName, replicaName, 80),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "ranking.relevancy_strictness", "80"),
				),
			},
			{
				PreConfig: testAccMutateVirtualIndex(t, replicaName, 25),
				Config:    testAccVirtualIndexResourceConfig(primaryIndexName, replicaName, 80),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "ranking.relevancy_strictness", "80"),
				),
			},
			{
				PreConfig: testAccUnlinkVirtualReplica(t, primaryIndexName, replicaName),
				Config:    testAccVirtualIndexResourceConfig(primaryIndexName, replicaName, 80),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "name", replicaName),
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "primary_index_name", primaryIndexName),
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "ranking.relevancy_strictness", "80"),
				),
			},
		},
	})
}

func testAccRequireCredentials(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	if os.Getenv("ALGOLIA_APP_ID") == "" || os.Getenv("ALGOLIA_API_KEY") == "" {
		t.Skip("ALGOLIA_APP_ID and ALGOLIA_API_KEY must be set for acceptance tests")
	}
}

func TestAccVirtualIndexDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	primaryIndexName := fmt.Sprintf("tf-virtual-ds-primary-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	replicaName := fmt.Sprintf("tf-virtual-ds-replica-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccVirtualIndexProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualIndexDataSourceConfig(primaryIndexName, replicaName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.algolia_virtual_index.test", "name", replicaName),
					resource.TestCheckResourceAttr("data.algolia_virtual_index.test", "primary_index_name", primaryIndexName),
				),
			},
		},
	})
}

func testAccCheckVirtualIndexDestroy(s *terraform.State) error {
	client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_virtual_index" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		_, err := client.GetSettings(client.NewApiGetSettingsRequest(name))
		if err == nil {
			return fmt.Errorf("virtual index %s still exists", name)
		}
	}

	return nil
}

func testAccMutateVirtualIndex(t *testing.T, replicaName string, strictness int32) func() {
	t.Helper()

	return func() {
		t.Helper()

		client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
		if err != nil {
			t.Fatalf("create Algolia client: %v", err)
		}

		setResp, err := client.SetSettings(client.NewApiSetSettingsRequest(replicaName, search.NewIndexSettings(
			search.WithIndexSettingsRelevancyStrictness(strictness),
			search.WithIndexSettingsCustomRanking([]string{"desc(popularity)"}),
		)))
		if err != nil {
			t.Fatalf("mutate virtual index %s: %v", replicaName, err)
		}

		testAccWaitForVirtualIndexMutation(t, client, replicaName, setResp.TaskID, strictness)
	}
}

func testAccUnlinkVirtualReplica(t *testing.T, primaryIndexName, replicaName string) func() {
	t.Helper()

	return func() {
		t.Helper()

		client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
		if err != nil {
			t.Fatalf("create Algolia client: %v", err)
		}

		settings, err := client.GetSettings(client.NewApiGetSettingsRequest(primaryIndexName))
		if err != nil {
			t.Fatalf("read primary index %s settings: %v", primaryIndexName, err)
		}

		virtualReplicaName := "virtual(" + replicaName + ")"
		filteredReplicas := make([]string, 0, len(settings.GetReplicas()))
		for _, replica := range settings.GetReplicas() {
			if replica != virtualReplicaName {
				filteredReplicas = append(filteredReplicas, replica)
			}
		}

		setResp, err := client.SetSettings(client.NewApiSetSettingsRequest(primaryIndexName, search.NewIndexSettings(
			search.WithIndexSettingsReplicas(filteredReplicas),
		)))
		if err != nil {
			t.Fatalf("unlink virtual replica %s from %s: %v", replicaName, primaryIndexName, err)
		}

		testAccWaitForVirtualReplicaUnlink(t, client, primaryIndexName, replicaName, setResp.TaskID)
	}
}

func testAccWaitForVirtualIndexMutation(t *testing.T, client *search.APIClient, replicaName string, taskID int64, strictness int32) {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	for {
		task, err := client.GetTask(client.NewApiGetTaskRequest(replicaName, taskID))
		if err == nil && task.Status == search.TASK_STATUS_PUBLISHED {
			settings, err := client.GetSettings(client.NewApiGetSettingsRequest(replicaName))
			if err == nil && settings.GetRelevancyStrictness() == strictness {
				return
			}
		}

		if time.Now().After(deadline) {
			t.Fatalf("virtual index %s mutation did not become visible", replicaName)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func testAccWaitForVirtualReplicaUnlink(t *testing.T, client *search.APIClient, primaryIndexName, replicaName string, taskID int64) {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	for {
		task, err := client.GetTask(client.NewApiGetTaskRequest(primaryIndexName, taskID))
		if err == nil && task.Status == search.TASK_STATUS_PUBLISHED {
			settings, err := client.GetSettings(client.NewApiGetSettingsRequest(replicaName))
			if err == nil && settings.GetPrimary() == "" {
				return
			}
		}

		if time.Now().After(deadline) {
			t.Fatalf("virtual index %s unlink from %s did not become visible", replicaName, primaryIndexName)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func testAccVirtualIndexResourceConfig(primaryIndexName, replicaName string, strictness int) string {
	return fmt.Sprintf(`
resource "algolia_index" "primary" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_virtual_index" "test" {
  name                = %[2]q
  primary_index_name  = algolia_index.primary.name
  deletion_protection = false

  ranking {
    relevancy_strictness = %[3]d
    custom_ranking       = ["desc(popularity)"]
  }
}
`, primaryIndexName, replicaName, strictness)
}

func testAccVirtualIndexDataSourceConfig(primaryIndexName, replicaName string) string {
	return fmt.Sprintf(`
resource "algolia_index" "primary" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_virtual_index" "test" {
  name                = %[2]q
  primary_index_name  = algolia_index.primary.name
  deletion_protection = false

  ranking {
    relevancy_strictness = 80
  }
}

data "algolia_virtual_index" "test" {
  name = algolia_virtual_index.test.name
}
`, primaryIndexName, replicaName)
}
