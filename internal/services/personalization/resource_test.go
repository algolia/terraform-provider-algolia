package personalization_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	api "github.com/algolia/algoliasearch-client-go/v4/algolia/personalization"
	"github.com/algolia/terraform-provider-algolia/internal/analyticsregion"
	"github.com/algolia/terraform-provider-algolia/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"algolia": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestAccPersonalizationStrategyResource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPersonalizationStrategyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPersonalizationStrategyConfig(80, 50, 30),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_personalization_strategy.test", "id", "default"),
					resource.TestCheckResourceAttr("algolia_personalization_strategy.test", "personalization_impact", "80"),
				),
			},
			{
				Config: testAccPersonalizationStrategyConfig(60, 40, 20),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_personalization_strategy.test", "personalization_impact", "60"),
				),
			},
			{
				ResourceName:      "algolia_personalization_strategy.test",
				ImportState:       true,
				ImportStateId:     "default",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPersonalizationStrategyResource_drift(t *testing.T) {
	testAccRequireCredentials(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPersonalizationStrategyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPersonalizationStrategyUnsortedConfig(80, 50, 30),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_personalization_strategy.test", "personalization_impact", "80"),
					// Index 0 is the first block in configuration order, not the
					// alphabetically first rule.
					resource.TestCheckResourceAttr("algolia_personalization_strategy.test", "events_scoring.0.event_name", "Zebra Viewed"),
					resource.TestCheckResourceAttr("algolia_personalization_strategy.test", "events_scoring.1.event_name", "Product Clicked"),
					resource.TestCheckResourceAttr("algolia_personalization_strategy.test", "facets_scoring.0.facet_name", "category"),
					resource.TestCheckResourceAttr("algolia_personalization_strategy.test", "facets_scoring.1.facet_name", "brand"),
				),
			},
			{
				PreConfig: testAccMutatePersonalizationStrategy(t, 15, 10, 5),
				Config:    testAccPersonalizationStrategyUnsortedConfig(80, 50, 30),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_personalization_strategy.test", "personalization_impact", "80"),
					// Index 0 is the first block in configuration order, not the
					// alphabetically first rule.
					resource.TestCheckResourceAttr("algolia_personalization_strategy.test", "events_scoring.0.event_name", "Zebra Viewed"),
					resource.TestCheckResourceAttr("algolia_personalization_strategy.test", "events_scoring.1.event_name", "Product Clicked"),
					resource.TestCheckResourceAttr("algolia_personalization_strategy.test", "facets_scoring.0.facet_name", "category"),
					resource.TestCheckResourceAttr("algolia_personalization_strategy.test", "facets_scoring.1.facet_name", "brand"),
				),
			},
		},
	})
}

func TestAccPersonalizationStrategyDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPersonalizationStrategyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPersonalizationStrategyDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.algolia_personalization_strategy.test", "id", "default"),
					resource.TestCheckResourceAttr("data.algolia_personalization_strategy.test", "personalization_impact", "80"),
				),
			},
		},
	})
}

func testAccCheckPersonalizationStrategyDestroy(_ *terraform.State) error {
	client, err := analyticsregion.NewPersonalizationClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"), os.Getenv(analyticsregion.EnvVar))
	if err != nil {
		return err
	}

	resp, err := client.GetPersonalizationStrategy()
	if err != nil {
		return err
	}

	if resp.GetPersonalizationImpact() != 0 {
		return fmt.Errorf("expected personalization strategy impact to be reset to 0 on destroy, got %d", resp.GetPersonalizationImpact())
	}
	for _, event := range resp.GetEventsScoring() {
		if event.GetScore() != 0 {
			return fmt.Errorf("expected events_scoring scores to be reset to 0 on destroy, got %#v", resp.GetEventsScoring())
		}
	}
	for _, facet := range resp.GetFacetsScoring() {
		if facet.GetScore() != 0 {
			return fmt.Errorf("expected facets_scoring scores to be reset to 0 on destroy, got %#v", resp.GetFacetsScoring())
		}
	}

	return nil
}

func testAccRequireCredentials(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	if os.Getenv("ALGOLIA_RUN_PERSONALIZATION_ACC") != "1" {
		t.Skip("Set ALGOLIA_RUN_PERSONALIZATION_ACC=1 to run Personalization acceptance tests; strategy saves are quota-limited per day")
	}

	if os.Getenv("ALGOLIA_APP_ID") == "" || os.Getenv("ALGOLIA_API_KEY") == "" || os.Getenv(analyticsregion.EnvVar) == "" {
		t.Skip("ALGOLIA_APP_ID, ALGOLIA_API_KEY, and ALGOLIA_ANALYTICS_REGION must be set for acceptance tests")
	}
}

func testAccMutatePersonalizationStrategy(t *testing.T, impact, eventScore, facetScore int) func() {
	t.Helper()

	return func() {
		t.Helper()

		client, err := analyticsregion.NewPersonalizationClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"), os.Getenv(analyticsregion.EnvVar))
		if err != nil {
			t.Fatalf("create personalization client: %v", err)
		}

		req := api.NewEmptyPersonalizationStrategyParams()
		req.SetPersonalizationImpact(int32(impact))
		req.SetEventsScoring([]api.EventsScoring{
			*api.NewEventsScoring(int32(eventScore), "Drifted Click", api.EVENT_TYPE_CLICK),
		})
		req.SetFacetsScoring([]api.FacetsScoring{
			*api.NewFacetsScoring(int32(facetScore), "brand"),
		})

		if _, err := client.SetPersonalizationStrategy(client.NewApiSetPersonalizationStrategyRequest(req)); err != nil {
			t.Fatalf("mutate personalization strategy: %v", err)
		}

		testAccWaitForPersonalizationStrategy(t, client, int32(impact), "Drifted Click")
	}
}

func testAccWaitForPersonalizationStrategy(t *testing.T, client *api.APIClient, impact int32, eventName string) {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	for {
		resp, err := client.GetPersonalizationStrategy()
		if err == nil && resp.GetPersonalizationImpact() == impact {
			events := resp.GetEventsScoring()
			if len(events) > 0 && events[0].GetEventName() == eventName {
				return
			}
		}

		if time.Now().After(deadline) {
			t.Fatalf("personalization strategy mutation did not become visible")
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// testAccPersonalizationStrategyConfig holds a single rule per block, so state
// order is unambiguous and ImportStateVerify can compare it: an import has no
// configuration to take block order from, so it falls back to a deterministic
// sorted order (see orderEventsScoring), which for one rule per block is the
// configured order as well.
func testAccPersonalizationStrategyConfig(impact, eventScore, facetScore int) string {
	return fmt.Sprintf(`
resource "algolia_personalization_strategy" "test" {
  personalization_impact = %[1]d

  events_scoring {
    event_name = "Product Clicked"
    event_type = "click"
    score      = %[2]d
  }

  facets_scoring {
    facet_name = "category"
    score      = %[3]d
  }
}
`, impact, eventScore, facetScore)
}

// testAccPersonalizationStrategyUnsortedConfig deliberately lists the scoring
// blocks in an order that is NOT the sorted one - "Zebra Viewed" (view) sorts
// after "Product Clicked" (click), and "category" after "brand". events_scoring
// and facets_scoring are list blocks, so block order is part of the value
// Terraform compares between plan and apply: a provider that imposes its own
// order here fails every apply with "Provider produced inconsistent result after
// apply".
func testAccPersonalizationStrategyUnsortedConfig(impact, eventScore, facetScore int) string {
	return fmt.Sprintf(`
resource "algolia_personalization_strategy" "test" {
  personalization_impact = %[1]d

  events_scoring {
    event_name = "Zebra Viewed"
    event_type = "view"
    score      = %[2]d
  }

  events_scoring {
    event_name = "Product Clicked"
    event_type = "click"
    score      = %[2]d
  }

  facets_scoring {
    facet_name = "category"
    score      = %[3]d
  }

  facets_scoring {
    facet_name = "brand"
    score      = %[3]d
  }
}
`, impact, eventScore, facetScore)
}

func testAccPersonalizationStrategyDataSourceConfig() string {
	return testAccPersonalizationStrategyConfig(80, 50, 30) + `

data "algolia_personalization_strategy" "test" {
  depends_on = [algolia_personalization_strategy.test]
}
`
}
