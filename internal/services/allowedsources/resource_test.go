package allowedsources_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/terraform-provider-algolia/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"algolia": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// Allowed sources are application-level global state (there is exactly one
// allowlist per app, replaced wholesale on every apply), so these tests
// snapshot the sources in place before running and restore them afterwards
// via t.Cleanup, in addition to relying on the resource's own Delete (which
// clears the allowlist).
//
// The test configuration always includes "0.0.0.0/0" (allow all IPv4) as an
// allowed source rather than a specific narrow IP. This is deliberate: this
// resource replaces the ENTIRE allowlist on every apply, so using a narrow
// test IP could lock the very machine running these tests (and its
// follow-up GetSources/restore calls) out of the Algolia API. Allowing all
// addresses still exercises the full CRUD/import lifecycle without that
// risk.

func TestAccAllowedSourcesResource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		t.Fatalf("create search client: %v", err)
	}

	original, err := client.GetSources()
	if err != nil {
		t.Fatalf("read original allowed sources: %v", err)
	}
	t.Cleanup(func() {
		testAccRestoreAllowedSources(t, client, original)
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAllowedSourcesDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAllowedSourcesConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("algolia_allowed_sources.test", "id"),
					resource.TestCheckResourceAttr("algolia_allowed_sources.test", "source.#", "1"),
				),
			},
			{
				ResourceName:      "algolia_allowed_sources.test",
				ImportState:       true,
				ImportStateId:     "placeholder",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAllowedSourcesDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		t.Fatalf("create search client: %v", err)
	}

	original, err := client.GetSources()
	if err != nil {
		t.Fatalf("read original allowed sources: %v", err)
	}
	t.Cleanup(func() {
		testAccRestoreAllowedSources(t, client, original)
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAllowedSourcesDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAllowedSourcesDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.algolia_allowed_sources.test", "source.#", "1"),
				),
			},
		},
	})
}

func testAccCheckAllowedSourcesDestroy(_ *terraform.State) error {
	client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		return err
	}

	sources, err := client.GetSources()
	if err != nil {
		return err
	}

	if len(sources) != 0 {
		return fmt.Errorf("expected allowed sources to be cleared on destroy, got %#v", sources)
	}

	return nil
}

// testAccRestoreAllowedSources sets the allowed sources allowlist back to
// the given snapshot, so acceptance runs do not leave the application's
// global IP allowlist mutated. ReplaceSources rejects an empty slice
// client-side, so an empty snapshot is restored by deleting whatever is
// currently configured instead.
func testAccRestoreAllowedSources(t *testing.T, client *search.APIClient, original []search.Source) {
	t.Helper()

	if len(original) == 0 {
		current, err := client.GetSources()
		if err != nil {
			t.Errorf("read allowed sources during restore: %v", err)
			return
		}
		for _, source := range current {
			if _, err := client.DeleteSource(client.NewApiDeleteSourceRequest(source.GetSource())); err != nil {
				t.Errorf("restore allowed sources (delete %q): %v", source.GetSource(), err)
			}
		}
		return
	}

	if _, err := client.ReplaceSources(client.NewApiReplaceSourcesRequest(original)); err != nil {
		t.Errorf("restore allowed sources: %v", err)
	}
}

func testAccRequireCredentials(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	if os.Getenv("ALGOLIA_RUN_ALLOWEDSOURCES_ACC") != "1" {
		t.Skip("Set ALGOLIA_RUN_ALLOWEDSOURCES_ACC=1 to run allowed sources acceptance tests; the endpoint requires the Vault feature, which is not enabled on most applications (returns HTTP 402 otherwise)")
	}

	if os.Getenv("ALGOLIA_APP_ID") == "" || os.Getenv("ALGOLIA_API_KEY") == "" {
		t.Skip("ALGOLIA_APP_ID and ALGOLIA_API_KEY must be set for acceptance tests")
	}
}

func testAccAllowedSourcesConfig() string {
	return `
resource "algolia_allowed_sources" "test" {
  # 0.0.0.0/0 allows all IPv4 addresses. Acceptance tests use this instead of
  # a specific IP so the test runner's own address is never excluded from
  # the allowlist (see the lockout warning on this resource) - otherwise the
  # very next API call made by this test, or by its cleanup, could fail.
  source = [
    {
      source      = "0.0.0.0/0"
      description = "terraform-provider-algolia acceptance test - allow all"
    },
  ]
}
`
}

func testAccAllowedSourcesDataSourceConfig() string {
	return testAccAllowedSourcesConfig() + `

data "algolia_allowed_sources" "test" {
  depends_on = [algolia_allowed_sources.test]
}
`
}
