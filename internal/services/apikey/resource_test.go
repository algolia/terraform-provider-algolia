package apikey_test

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

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"algolia": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestAccAPIKeyResource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	description := fmt.Sprintf("tf-api-key-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	updatedDescription := description + "-updated"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAPIKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyResourceConfig(description, 100),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("algolia_api_key.test", "id"),
					resource.TestCheckResourceAttr("algolia_api_key.test", "description", description),
					resource.TestCheckResourceAttr("algolia_api_key.test", "max_hits_per_query", "100"),
					resource.TestCheckResourceAttrSet("algolia_api_key.test", "created_at"),
				),
			},
			{
				Config: testAccAPIKeyResourceConfig(updatedDescription, 200),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_api_key.test", "description", updatedDescription),
					resource.TestCheckResourceAttr("algolia_api_key.test", "max_hits_per_query", "200"),
				),
			},
			{
				ResourceName:            "algolia_api_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"expires_at"},
			},
			{
				Config: testAccAPIKeyResourceConfig(updatedDescription, 200),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_api_key.test", "description", updatedDescription),
				),
			},
		},
	})
}

func TestAccAPIKeyResource_drift(t *testing.T) {
	testAccRequireCredentials(t)

	description := fmt.Sprintf("tf-api-key-drift-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	var key string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAPIKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyResourceConfig(description, 100),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCaptureAPIKeyID("algolia_api_key.test", &key),
					resource.TestCheckResourceAttr("algolia_api_key.test", "description", description),
					resource.TestCheckResourceAttr("algolia_api_key.test", "max_hits_per_query", "100"),
				),
			},
			{
				PreConfig: testAccMutateAPIKey(t, &key, "drifted-description", 25),
				Config:    testAccAPIKeyResourceConfig(description, 100),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_api_key.test", "description", description),
					resource.TestCheckResourceAttr("algolia_api_key.test", "max_hits_per_query", "100"),
				),
			},
		},
	})
}

// TestAccAPIKeyResource_queryParameters covers what actually cures the
// wipe-on-update blocker for this attribute: modelling it at all. Import has to
// hydrate the restriction, because state is what lets a later apply either
// preserve it or show its removal. Clearing it with an empty string also has to
// reach the key, since that is the one value the API treats as "drop the field".
func TestAccAPIKeyResource_queryParameters(t *testing.T) {
	testAccRequireCredentials(t)

	description := fmt.Sprintf("tf-acc-api-key-qp-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	restriction := "filters=tenant%3Aacme"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAPIKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyQueryParametersConfig(description, fmt.Sprintf("query_parameters = %q", restriction)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_api_key.test", "query_parameters", restriction),
					testAccCheckAPIKeyQueryParameters("algolia_api_key.test", restriction),
				),
			},
			{
				// Before the fix there was no attribute for import to fill, so
				// state had no record of the restriction at all.
				ResourceName:            "algolia_api_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"expires_at"},
			},
			{
				// Changing the restriction is an ordinary update. This stays a
				// filters restriction on purpose: the API rejects a
				// restrictSources value that does not contain the caller's own IP.
				Config: testAccAPIKeyQueryParametersConfig(description, `query_parameters = "filters=tenant%3Abeta"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_api_key.test", "query_parameters", "filters=tenant%3Abeta"),
					testAccCheckAPIKeyQueryParameters("algolia_api_key.test", "filters=tenant%3Abeta"),
				),
			},
			{
				Config: testAccAPIKeyQueryParametersConfig(description, `query_parameters = ""`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_api_key.test", "query_parameters", ""),
					testAccCheckAPIKeyQueryParameters("algolia_api_key.test", ""),
				),
			},
		},
	})
}

// TestAccAPIKeyResource_optionalRestrictionsPlanTheirRemoval is the guard on the
// wipe-on-update hazard as a whole. UpdateApiKey resets every attribute the
// request omits, and the provider does send a request that omits them - but all
// five are Optional and not Computed and are hydrated from the API on every Read
// and Import, so state records them and Terraform plans the removal rather than
// performing it unannounced. The PlanOnly step is the assertion that matters:
// dropping them from configuration produces a non-empty plan. The step after it
// confirms the plan was honest and the values really are gone.
func TestAccAPIKeyResource_optionalRestrictionsPlanTheirRemoval(t *testing.T) {
	testAccRequireCredentials(t)

	description := fmt.Sprintf("tf-acc-api-key-restrict-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAPIKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyRestrictionsConfig(description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_api_key.test", "indexes.0", "products_*"),
					resource.TestCheckResourceAttr("algolia_api_key.test", "referers.0", "https://example.com/*"),
					resource.TestCheckResourceAttr("algolia_api_key.test", "max_hits_per_query", "100"),
					resource.TestCheckResourceAttr("algolia_api_key.test", "max_queries_per_ip_per_hour", "1000"),
					resource.TestCheckResourceAttr("algolia_api_key.test", "query_parameters", "filters=tenant%3Aacme"),
				),
			},
			{
				ResourceName:            "algolia_api_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"expires_at"},
			},
			{
				Config:             testAccAPIKeyNoRestrictionsConfig(description),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccAPIKeyNoRestrictionsConfig(description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("algolia_api_key.test", "indexes.0"),
					resource.TestCheckNoResourceAttr("algolia_api_key.test", "referers.0"),
					resource.TestCheckNoResourceAttr("algolia_api_key.test", "max_hits_per_query"),
					resource.TestCheckNoResourceAttr("algolia_api_key.test", "max_queries_per_ip_per_hour"),
					resource.TestCheckNoResourceAttr("algolia_api_key.test", "query_parameters"),
					testAccCheckAPIKeyQueryParameters("algolia_api_key.test", ""),
				),
			},
		},
	})
}

func testAccCheckAPIKeyDestroy(s *terraform.State) error {
	client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_api_key" {
			continue
		}

		_, err := client.GetApiKey(client.NewApiGetApiKeyRequest(rs.Primary.ID))
		if err == nil {
			return fmt.Errorf("api key %s still exists", rs.Primary.ID)
		}
	}

	return nil
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

func testAccCaptureAPIKeyID(resourceName string, key *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}

		*key = rs.Primary.ID
		return nil
	}
}

func testAccMutateAPIKey(t *testing.T, key *string, description string, maxHits int32) func() {
	t.Helper()

	return func() {
		t.Helper()

		if *key == "" {
			t.Fatal("API key ID was not captured before drift mutation")
		}

		client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
		if err != nil {
			t.Fatalf("create Algolia client: %v", err)
		}

		current, err := client.GetApiKey(client.NewApiGetApiKeyRequest(*key))
		if err != nil {
			t.Fatalf("read API key %s before mutation: %v", *key, err)
		}

		mutated := search.NewApiKey(current.GetAcl())
		mutated.SetDescription(description)
		mutated.SetIndexes(current.GetIndexes())
		mutated.SetReferers(current.GetReferers())
		mutated.SetMaxHitsPerQuery(maxHits)
		if value, ok := current.GetMaxQueriesPerIPPerHourOk(); ok && value != nil {
			mutated.SetMaxQueriesPerIPPerHour(*value)
		}
		if value, ok := current.GetValidityOk(); ok && value != nil {
			mutated.SetValidity(*value)
		}

		if _, err := client.UpdateApiKey(client.NewApiUpdateApiKeyRequest(*key, mutated)); err != nil {
			t.Fatalf("mutate API key %s: %v", *key, err)
		}

		deadline := time.Now().Add(30 * time.Second)
		for {
			apiKey, err := client.GetApiKey(client.NewApiGetApiKeyRequest(*key))
			if err == nil && apiKey.GetDescription() == description && apiKey.GetMaxHitsPerQuery() == maxHits {
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("API key %s mutation did not become visible", *key)
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// testAccCheckAPIKeyQueryParameters asserts what the live key carries, not just
// what state records: the point of the fail-safe is the restriction still being
// enforced by Algolia.
func testAccCheckAPIKeyQueryParameters(resourceName, want string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}

		client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
		if err != nil {
			return err
		}

		apiKey, err := client.GetApiKey(client.NewApiGetApiKeyRequest(rs.Primary.ID))
		if err != nil {
			return fmt.Errorf("read API key described as %q: %w", rs.Primary.Attributes["description"], err)
		}

		if got := apiKey.GetQueryParameters(); got != want {
			return fmt.Errorf("live query parameters = %q, want %q", got, want)
		}

		return nil
	}
}

func testAccAPIKeyQueryParametersConfig(description, queryParameters string) string {
	return fmt.Sprintf(`
resource "algolia_api_key" "test" {
  acl         = ["search"]
  description = %[1]q
  %[2]s
}
`, description, queryParameters)
}

func testAccAPIKeyRestrictionsConfig(description string) string {
	return fmt.Sprintf(`
resource "algolia_api_key" "test" {
  acl                         = ["search"]
  description                 = %[1]q
  indexes                     = ["products_*"]
  referers                    = ["https://example.com/*"]
  max_hits_per_query          = 100
  max_queries_per_ip_per_hour = 1000
  query_parameters            = %[2]q
}
`, description, "filters=tenant%3Aacme")
}

func testAccAPIKeyNoRestrictionsConfig(description string) string {
	return fmt.Sprintf(`
resource "algolia_api_key" "test" {
  acl         = ["search"]
  description = %[1]q
}
`, description)
}

func testAccAPIKeyResourceConfig(description string, maxHits int) string {
	return fmt.Sprintf(`
resource "algolia_api_key" "test" {
  acl                         = ["search", "browse"]
  description                 = %[1]q
  expires_at                  = "2030-01-01T00:00:00Z"
  indexes                     = ["products_*"]
  referers                    = ["https://example.com/*"]
  max_hits_per_query          = %[2]d
  max_queries_per_ip_per_hour = 1000
}
`, description, maxHits)
}
