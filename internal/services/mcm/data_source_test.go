package mcm_test

import (
	"os"
	"testing"

	"github.com/algolia/terraform-provider-algolia/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The MCM (multi-cluster management) endpoints error out on applications
// that aren't on a multi-cluster plan, which is most applications. So on
// top of the usual TF_ACC/ALGOLIA_APP_ID/ALGOLIA_API_KEY gating, these
// acceptance tests are additionally gated behind ALGOLIA_RUN_MCM_ACC=1 -
// mirroring how the A/B Testing and Personalization acceptance tests are
// gated behind ALGOLIA_RUN_ABTESTING_ACC / ALGOLIA_RUN_PERSONALIZATION_ACC
// (see AGENTS.md).

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"algolia": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestAccClustersDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "algolia_clusters" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.algolia_clusters.test", "id"),
					resource.TestCheckResourceAttrSet("data.algolia_clusters.test", "clusters.#"),
				),
			},
		},
	})
}

func TestAccUserIdsDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "algolia_user_ids" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.algolia_user_ids.test", "id"),
					resource.TestCheckResourceAttrSet("data.algolia_user_ids.test", "user_ids.#"),
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

	if os.Getenv("ALGOLIA_RUN_MCM_ACC") != "1" {
		t.Skip("Set ALGOLIA_RUN_MCM_ACC=1 to run MCM acceptance tests; the MCM endpoints error out on applications that aren't on a multi-cluster plan")
	}

	if os.Getenv("ALGOLIA_APP_ID") == "" || os.Getenv("ALGOLIA_API_KEY") == "" {
		t.Skip("ALGOLIA_APP_ID and ALGOLIA_API_KEY must be set for acceptance tests")
	}
}
