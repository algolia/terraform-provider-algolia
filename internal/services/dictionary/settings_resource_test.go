package dictionary_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccProtoV6ProviderFactories and testAccRequireCredentials are shared
// with resource_test.go (algolia_dictionary_entry) in this same
// dictionary_test package.

// Dictionary settings are application-level global state (there is exactly
// one configuration per app), so these tests snapshot the settings in place
// before running and restore them afterwards via t.Cleanup, in addition to
// relying on the resource's own Delete (which resets to defaults).

func TestAccDictionarySettingsResource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		t.Fatalf("create search client: %v", err)
	}

	original, err := client.GetDictionarySettings()
	if err != nil {
		t.Fatalf("read original dictionary settings: %v", err)
	}
	t.Cleanup(func() {
		testAccRestoreDictionarySettings(t, client, original.GetDisableStandardEntries())
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDictionarySettingsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDictionarySettingsConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("algolia_dictionary_settings.test", "id"),
					resource.TestCheckResourceAttr("algolia_dictionary_settings.test", "disable_standard_entries.stopwords.en", "true"),
				),
			},
			{
				ResourceName:      "algolia_dictionary_settings.test",
				ImportState:       true,
				ImportStateId:     "placeholder",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDictionarySettingsDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		t.Fatalf("create search client: %v", err)
	}

	original, err := client.GetDictionarySettings()
	if err != nil {
		t.Fatalf("read original dictionary settings: %v", err)
	}
	t.Cleanup(func() {
		testAccRestoreDictionarySettings(t, client, original.GetDisableStandardEntries())
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDictionarySettingsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDictionarySettingsDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.algolia_dictionary_settings.test", "disable_standard_entries.stopwords.en", "true"),
				),
			},
		},
	})
}

func testAccCheckDictionarySettingsDestroy(_ *terraform.State) error {
	client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		return err
	}

	resp, err := client.GetDictionarySettings()
	if err != nil {
		return err
	}

	entries := resp.GetDisableStandardEntries()
	if len(entries.GetStopwords()) != 0 {
		return fmt.Errorf("expected dictionary settings to be reset to defaults (nothing disabled) on destroy, got %#v", entries)
	}

	return nil
}

// testAccRestoreDictionarySettings sets dictionary settings back to the
// given snapshot and waits for the app task to complete, so acceptance runs
// do not leave global dictionary settings mutated for the application.
func testAccRestoreDictionarySettings(t *testing.T, client *search.APIClient, original search.StandardEntries) {
	t.Helper()

	params := search.NewDictionarySettingsParams(original)
	updateResp, err := client.SetDictionarySettings(client.NewApiSetDictionarySettingsRequest(params))
	if err != nil {
		t.Errorf("restore dictionary settings: %v", err)
		return
	}

	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		taskResp, err := client.GetAppTask(client.NewApiGetAppTaskRequest(updateResp.TaskID))
		if err == nil && taskResp.Status == search.TASK_STATUS_PUBLISHED {
			return
		}
		time.Sleep(time.Second)
	}
	t.Errorf("restoring dictionary settings did not complete within 2 minutes")
}

func testAccDictionarySettingsConfig() string {
	return `
resource "algolia_dictionary_settings" "test" {
  disable_standard_entries = {
    stopwords = {
      en = true
    }
  }
}
`
}

func testAccDictionarySettingsDataSourceConfig() string {
	return testAccDictionarySettingsConfig() + `

data "algolia_dictionary_settings" "test" {
  depends_on = [algolia_dictionary_settings.test]
}
`
}
