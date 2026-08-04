//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	testconfig "github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestE2ESearchExamples executes the checked-in complete Search examples, rather
// than a configuration copied into the test. A Terraform override gives each run
// a unique index name and turns deletion protection off without weakening the
// published example's safe defaults.
func TestE2ESearchExamples(t *testing.T) {
	requireE2E(t)

	tests := []struct {
		name              string
		directory         string
		indexResource     string
		indexLabel        string
		ruleResource      string
		ruleLabel         string
		ruleObjectID      string
		ruleDescription   string
		synonymResource   string
		synonymLabel      string
		synonymObjectID   string
		synonyms          []string
		apiKeyResource    string
		apiKeyLabel       string
		apiKeyDescription string
	}{
		{
			name:              "ecommerce-search",
			directory:         "ecommerce-search",
			indexResource:     "algolia_index.products",
			indexLabel:        "products",
			ruleResource:      "algolia_rule.boost_on_sale",
			ruleLabel:         "boost_on_sale",
			ruleObjectID:      "boost-on-sale",
			ruleDescription:   "Prioritise on-sale products for sale-intent queries",
			synonymResource:   "algolia_synonym.tv",
			synonymLabel:      "tv",
			synonymObjectID:   "tv-television",
			synonyms:          []string{"tv", "television", "telly"},
			apiKeyResource:    "algolia_api_key.frontend_search",
			apiKeyLabel:       "frontend_search",
			apiKeyDescription: "Public search-only key for storefront clients",
		},
		{
			name:              "media-search",
			directory:         "media-search",
			indexResource:     "algolia_index.titles",
			indexLabel:        "titles",
			ruleResource:      "algolia_rule.boost_originals",
			ruleLabel:         "boost_originals",
			ruleObjectID:      "boost-originals",
			ruleDescription:   "Prioritise platform originals for trending/browse queries",
			synonymResource:   "algolia_synonym.scifi",
			synonymLabel:      "scifi",
			synonymObjectID:   "scifi-science-fiction",
			synonyms:          []string{"scifi", "sci-fi", "science fiction"},
			apiKeyResource:    "algolia_api_key.app_search",
			apiKeyLabel:       "app_search",
			apiKeyDescription: "Public search-only key for streaming app clients",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := e2eSearchClient(t)
			indexName := acctest.RandomWithPrefix("tf-e2e-example")
			updatedDescription := tt.ruleDescription + ", updated"
			updatedSynonyms := append(append([]string(nil), tt.synonyms...), "updated-term")
			var apiKeyID string

			baseDir := copyExampleWithOverride(t, tt.directory, searchExampleOverride(tt.indexLabel, indexName,
				tt.ruleLabel, tt.ruleDescription, tt.synonymLabel, tt.synonyms, tt.apiKeyLabel, 24, 50))
			updatedDir := copyExampleWithOverride(t, tt.directory, searchExampleOverride(tt.indexLabel, indexName,
				tt.ruleLabel, updatedDescription, tt.synonymLabel, updatedSynonyms, tt.apiKeyLabel, 48, 75))

			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: providerFactories,
				CheckDestroy: resource.ComposeAggregateTestCheckFunc(
					checkIndexAbsent(client, indexName),
					checkAPIKeyAbsent(client, &apiKeyID),
				),
				Steps: []resource.TestStep{
					{
						ConfigDirectory: testconfig.StaticDirectory(baseDir),
						ConfigVariables: exampleCredentials(),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(tt.indexResource, "name", indexName),
							resource.TestCheckResourceAttr(tt.indexResource, "deletion_protection", "false"),
							captureResourceID(tt.apiKeyResource, &apiKeyID),
							checkIndexHitsPerPage(client, indexName, 24),
							checkRuleDescription(client, indexName, tt.ruleObjectID, tt.ruleDescription),
							checkSynonyms(client, indexName, tt.synonymObjectID, tt.synonyms...),
							checkAPIKey(client, &apiKeyID, tt.apiKeyDescription, 50),
						),
					},
					{
						PreConfig:       driftSearchExample(t, client, indexName, tt.ruleObjectID, tt.synonymObjectID, &apiKeyID),
						ConfigDirectory: testconfig.StaticDirectory(baseDir),
						ConfigVariables: exampleCredentials(),
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(tt.indexResource, plancheck.ResourceActionUpdate),
							plancheck.ExpectResourceAction(tt.ruleResource, plancheck.ResourceActionCreate),
							plancheck.ExpectResourceAction(tt.synonymResource, plancheck.ResourceActionUpdate),
							plancheck.ExpectResourceAction(tt.apiKeyResource, plancheck.ResourceActionUpdate),
						}},
						Check: resource.ComposeAggregateTestCheckFunc(
							checkIndexHitsPerPage(client, indexName, 24),
							checkRuleDescription(client, indexName, tt.ruleObjectID, tt.ruleDescription),
							checkSynonyms(client, indexName, tt.synonymObjectID, tt.synonyms...),
							checkAPIKey(client, &apiKeyID, tt.apiKeyDescription, 50),
						),
					},
					{
						ConfigDirectory: testconfig.StaticDirectory(updatedDir),
						ConfigVariables: exampleCredentials(),
						Check: resource.ComposeAggregateTestCheckFunc(
							checkIndexHitsPerPage(client, indexName, 48),
							checkRuleDescription(client, indexName, tt.ruleObjectID, updatedDescription),
							checkSynonyms(client, indexName, tt.synonymObjectID, updatedSynonyms...),
							checkAPIKey(client, &apiKeyID, tt.apiKeyDescription, 75),
						),
					},
				},
			})
		})
	}
}

func exampleCredentials() testconfig.Variables {
	return testconfig.Variables{
		"algolia_app_id":  testconfig.StringVariable(os.Getenv("ALGOLIA_APP_ID")),
		"algolia_api_key": testconfig.StringVariable(os.Getenv("ALGOLIA_API_KEY")),
	}
}

func copyExampleWithOverride(t *testing.T, name, override string) string {
	t.Helper()

	sourceDir := filepath.Join(repoRoot(t), "examples", name)
	destinationDir := t.TempDir()
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		t.Fatalf("read example %s: %v", name, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".tf" || entry.Name() == "versions.tf" {
			continue
		}
		contents, err := os.ReadFile(filepath.Join(sourceDir, entry.Name()))
		if err != nil {
			t.Fatalf("read example file %s: %v", entry.Name(), err)
		}
		// The test framework serves the in-tree provider from this Go process. It
		// rejects a provider block in ConfigDirectory when a provider factory is
		// supplied, so remove only that block from the temporary copy. versions.tf
		// is skipped above because its release constraint would require a registry
		// install instead of the in-process provider. The static example validator
		// separately checks both checked-in files themselves.
		contents = bytes.Replace(contents, []byte(`provider "algolia" {
  app_id  = var.algolia_app_id
  api_key = var.algolia_api_key
}
`), nil, 1)
		contents = bytes.Replace(contents, []byte(`provider "algolia" {
  app_id  = var.algolia_app_id
  api_key = var.algolia_api_key

  # The Ingestion API is region-routed: analytics_region must match the region
  # your application is hosted in (or set ALGOLIA_ANALYTICS_REGION).
  analytics_region = var.analytics_region
}
`), nil, 1)

		if err := os.WriteFile(filepath.Join(destinationDir, entry.Name()), contents, 0o600); err != nil {
			t.Fatalf("copy example file %s: %v", entry.Name(), err)
		}
	}
	if err := os.WriteFile(filepath.Join(destinationDir, "example_override.tf"), []byte(override), 0o600); err != nil {
		t.Fatalf("write example override: %v", err)
	}
	return destinationDir
}

func searchExampleOverride(indexLabel, indexName, ruleLabel, ruleDescription, synonymLabel string,
	synonyms []string, apiKeyLabel string, hitsPerPage, maxHits int,
) string {
	synonymsJSON, err := json.Marshal(synonyms)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf(`
resource "algolia_index" %q {
  name                = %q
  deletion_protection = false

  pagination {
    hits_per_page = %d
  }
}

resource "algolia_rule" %q {
  description = %q
}

resource "algolia_synonym" %q {
  synonyms = %s
}

resource "algolia_api_key" %q {
  max_hits_per_query = %d
}
`, indexLabel, indexName, hitsPerPage, ruleLabel, ruleDescription, synonymLabel, synonymsJSON, apiKeyLabel, maxHits)
}

func captureResourceID(resourceName string, id *string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		*id = resourceState.Primary.ID
		return nil
	}
}

func driftSearchExample(t *testing.T, client *search.APIClient, indexName, ruleID, synonymID string, apiKeyID *string) func() {
	t.Helper()

	return func() {
		t.Helper()

		settings, err := client.SetSettings(client.NewApiSetSettingsRequest(indexName,
			search.NewIndexSettings(search.WithIndexSettingsHitsPerPage(999))))
		if err != nil {
			t.Fatalf("drift settings of %s: %v", indexName, err)
		}
		if _, err := client.WaitForTask(indexName, settings.TaskID); err != nil {
			t.Fatalf("wait for settings drift on %s: %v", indexName, err)
		}

		deletedRule, err := client.DeleteRule(client.NewApiDeleteRuleRequest(indexName, ruleID))
		if err != nil {
			t.Fatalf("delete rule %s out of band: %v", ruleID, err)
		}
		if _, err := client.WaitForTask(indexName, deletedRule.TaskID); err != nil {
			t.Fatalf("wait for rule deletion on %s: %v", indexName, err)
		}

		driftedSynonym, err := client.SaveSynonym(client.NewApiSaveSynonymRequest(indexName, synonymID,
			search.NewSynonymHit(synonymID, search.SYNONYM_TYPE_SYNONYM,
				search.WithSynonymHitSynonyms([]string{"drifted", "values"}))))
		if err != nil {
			t.Fatalf("rewrite synonym %s out of band: %v", synonymID, err)
		}
		if _, err := client.WaitForTask(indexName, driftedSynonym.TaskID); err != nil {
			t.Fatalf("wait for synonym drift on %s: %v", indexName, err)
		}

		mutateAPIKey(t, client, apiKeyID, "drifted example key", 999)
	}
}

func mutateAPIKey(t *testing.T, client *search.APIClient, keyID *string, description string, maxHits int32) {
	t.Helper()
	if *keyID == "" {
		t.Fatal("API key ID was not captured before drift mutation")
	}

	current, err := client.GetApiKey(client.NewApiGetApiKeyRequest(*keyID))
	if err != nil {
		t.Fatalf("read API key before mutation: %v", err)
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
	if _, err := client.UpdateApiKey(client.NewApiUpdateApiKeyRequest(*keyID, mutated)); err != nil {
		t.Fatalf("mutate API key: %v", err)
	}
	waitForAPIKey(t, client, keyID, description, maxHits)
}

func waitForAPIKey(t *testing.T, client *search.APIClient, keyID *string, description string, maxHits int32) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		key, err := client.GetApiKey(client.NewApiGetApiKeyRequest(*keyID))
		if err == nil && key.GetDescription() == description && key.GetMaxHitsPerQuery() == maxHits {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("API key mutation did not become visible: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func checkAPIKey(client *search.APIClient, keyID *string, description string, maxHits int32) resource.TestCheckFunc {
	return func(*terraform.State) error {
		key, err := client.GetApiKey(client.NewApiGetApiKeyRequest(*keyID))
		if err != nil {
			return fmt.Errorf("read API key: %w", err)
		}
		if key.GetDescription() != description || key.GetMaxHitsPerQuery() != maxHits {
			return fmt.Errorf("API key description/max hits = %q/%d, want %q/%d",
				key.GetDescription(), key.GetMaxHitsPerQuery(), description, maxHits)
		}
		return nil
	}
}

func checkAPIKeyAbsent(client *search.APIClient, keyID *string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if *keyID == "" {
			return nil
		}
		_, err := client.GetApiKey(client.NewApiGetApiKeyRequest(*keyID))
		if err == nil {
			return errors.New("API key still exists after destroy")
		}
		var apiErr *search.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return nil
		}
		return fmt.Errorf("unexpected error checking API key destruction: %w", err)
	}
}
