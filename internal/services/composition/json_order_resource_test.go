package composition_test

import (
	"fmt"
	"os"
	"slices"
	"testing"

	compositionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/composition"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccCompositionResource_feedOrder(t *testing.T) {
	testAccRequireCredentials(t)
	name := "tf-feed-order-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	config := func(first, second string) string {
		return fmt.Sprintf(`
resource "algolia_index" "test" {
  name                = %[1]q
  deletion_protection = false
}
resource "algolia_composition" "test" {
  object_id = %[1]q
  name      = "Feed order test"
  behavior = jsonencode({
    multifeed = {
      feeds = {
        featured = { injection = { main = { source = { search = { index = algolia_index.test.name } } } } }
        organic  = { injection = { main = { source = { search = { index = algolia_index.test.name } } } } }
      }
      feedsOrder = [%[2]q, %[3]q]
    }
  })
}
`, name, first, second)
	}
	check := func(want []string) resource.TestCheckFunc {
		return func(_ *terraform.State) error {
			client, err := compositionapi.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
			if err != nil {
				return err
			}
			got, err := client.GetComposition(client.NewApiGetCompositionRequest(name))
			if err != nil {
				return err
			}
			behavior := got.Behavior.CompositionMultifeedBehavior
			if behavior == nil || !slices.Equal(behavior.Multifeed.FeedsOrder, want) {
				return fmt.Errorf("API composition does not have feed order %v", want)
			}
			return nil
		}
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCompositionDestroy,
		Steps: []resource.TestStep{
			{Config: config("featured", "organic"), Check: check([]string{"featured", "organic"})},
			{Config: config("organic", "featured"), Check: check([]string{"organic", "featured"})},
		},
	})
}
