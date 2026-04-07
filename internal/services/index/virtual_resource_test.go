package index_test

import (
	"fmt"
	"os"
	"testing"

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
				ResourceName:      "algolia_virtual_index.test",
				ImportState:       true,
				ImportStateVerify: true,
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
