package index_test

import (
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccIndexResource_renderingContentOrder(t *testing.T) {
	name := "tf-json-order-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	config := func(first, second string) string {
		return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %q
  deletion_protection = false
  languages {
    decompounded_attributes = jsonencode({ de = ["name", "description"] })
  }
  advanced {
    rendering_content = jsonencode({ facetOrdering = { facets = { order = [%q, %q] } } })
  }
}
`, name, first, second)
	}
	check := func(want []string) resource.TestCheckFunc {
		return func(_ *terraform.State) error {
			client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
			if err != nil {
				return err
			}
			settings, err := client.GetSettings(client.NewApiGetSettingsRequest(name))
			if err != nil {
				return err
			}
			if settings.RenderingContent == nil || settings.RenderingContent.FacetOrdering == nil || settings.RenderingContent.FacetOrdering.Facets == nil {
				return fmt.Errorf("API returned no facet ordering for %s", name)
			}
			if got := settings.RenderingContent.FacetOrdering.Facets.Order; !slices.Equal(got, want) {
				return fmt.Errorf("API facet order = %v, want %v", got, want)
			}
			return nil
		}
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: config("brand", "category"),
				Check:  check([]string{"brand", "category"}),
			},
			{
				Config: config("category", "brand"),
				Check:  check([]string{"category", "brand"}),
			},
			{
				PreConfig: func() {
					client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
					if err != nil {
						t.Fatal(err)
					}
					result, err := client.SetSettings(client.NewApiSetSettingsRequest(name, &search.IndexSettings{
						RenderingContent: &search.RenderingContent{
							FacetOrdering: &search.FacetOrdering{
								Facets: &search.Facets{Order: []string{"brand", "category"}},
							},
						},
					}))
					if err != nil {
						t.Fatal(err)
					}
					if _, err := client.WaitForTask(name, result.TaskID); err != nil {
						t.Fatal(err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config("category", "brand"),
				Check:  check([]string{"category", "brand"}),
			},
		},
	})
}
