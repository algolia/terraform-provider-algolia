//go:build e2e

// Package e2e holds end-to-end tests that drive the provider against a real
// Algolia application. They are excluded from the normal unit-test build by
// the `e2e` build tag and only run via `make e2e` (which sets TF_ACC=1 and
// loads credentials from .env.e2e).
package e2e

import (
	"errors"
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

var providerFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"algolia": providerserver.NewProtocol6WithError(provider.New("e2e")()),
}

// requireE2E skips the test unless it is explicitly enabled (TF_ACC) and the
// fixed e2e application credentials are present.
func requireE2E(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("e2e tests are skipped unless TF_ACC is set; run `make e2e`")
	}
	if os.Getenv("ALGOLIA_APP_ID") == "" || os.Getenv("ALGOLIA_API_KEY") == "" {
		t.Skip("ALGOLIA_APP_ID and ALGOLIA_API_KEY must be set for e2e tests")
	}
}

func e2eSearchClient(t *testing.T) *search.APIClient {
	t.Helper()

	client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		t.Fatalf("creating search client: %v", err)
	}
	return client
}

// TestE2EIndexLifecycle drives a full create/update/destroy of an
// algolia_index against the real application, asserting at each step that both
// Terraform state and the live Algolia API agree - proving the provider can
// actually operate the resource end to end, not just track it in state.
func TestE2EIndexLifecycle(t *testing.T) {
	requireE2E(t)

	indexName := acctest.RandomWithPrefix("tf-e2e-index")
	client := e2eSearchClient(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		CheckDestroy:             checkIndexAbsent(client, indexName),
		Steps: []resource.TestStep{
			{ // Create
				Config: e2eIndexConfig(indexName, 20),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_index.test", "name", indexName),
					resource.TestCheckResourceAttr("algolia_index.test", "deletion_protection", "false"),
					resource.TestCheckResourceAttr("algolia_index.test", "attributes.searchable_attributes.0", "name"),
					resource.TestCheckResourceAttr("algolia_index.test", "pagination.hits_per_page", "20"),
					checkIndexHitsPerPage(client, indexName, 20),
				),
			},
			{ // Update
				Config: e2eIndexConfig(indexName, 50),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_index.test", "pagination.hits_per_page", "50"),
					checkIndexHitsPerPage(client, indexName, 50),
				),
			},
		},
	})
}

func e2eIndexConfig(name string, hitsPerPage int) string {
	return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false

  attributes {
    searchable_attributes = ["name", "description"]
  }

  pagination {
    hits_per_page = %[2]d
  }
}
`, name, hitsPerPage)
}

// checkIndexHitsPerPage asserts the live Algolia API reports the expected
// hitsPerPage, proving the settings actually reached Algolia rather than only
// living in Terraform state.
func checkIndexHitsPerPage(client *search.APIClient, indexName string, want int32) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		settings, err := client.GetSettings(client.NewApiGetSettingsRequest(indexName))
		if err != nil {
			return fmt.Errorf("reading settings for %s from the API: %w", indexName, err)
		}
		if settings.HitsPerPage == nil {
			return fmt.Errorf("index %s: hitsPerPage is nil in the API response", indexName)
		}
		if *settings.HitsPerPage != want {
			return fmt.Errorf("index %s: API hitsPerPage = %d, want %d", indexName, *settings.HitsPerPage, want)
		}
		return nil
	}
}

// checkIndexAbsent verifies the index no longer exists after destroy.
func checkIndexAbsent(client *search.APIClient, indexName string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		_, err := client.GetSettings(client.NewApiGetSettingsRequest(indexName))
		if err == nil {
			return fmt.Errorf("index %s still exists after destroy", indexName)
		}

		var apiErr *search.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return nil
		}
		return fmt.Errorf("unexpected error checking that index %s was destroyed: %w", indexName, err)
	}
}
