package ingestion_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/algolia/terraform-provider-algolia/internal/analyticsregion"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccProtoV6ProviderFactories and testAccRequireCredentials are shared
// with authentication_resource_test.go (same ingestion_test package).

func TestAccIngestionTransformationResource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	name := fmt.Sprintf("tf-acc-transformation-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIngestionTransformationDestroy,
		Steps: []resource.TestStep{
			{
				// Simplest possible transformation: a code-type
				// transformation with trivial source code, so no external
				// dependencies (authentication, source, destination) are
				// needed to exercise Create/Read/Update/Delete.
				Config: testAccIngestionTransformationConfig(name, "function transform(record) { return record; }"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_ingestion_transformation.test", "name", name),
					resource.TestCheckResourceAttr("algolia_ingestion_transformation.test", "type", "code"),
					resource.TestCheckResourceAttr("algolia_ingestion_transformation.test", "input", `{"code":"function transform(record) { return record; }"}`),
					resource.TestCheckResourceAttrSet("algolia_ingestion_transformation.test", "transformation_id"),
					resource.TestCheckResourceAttrSet("algolia_ingestion_transformation.test", "created_at"),
					resource.TestCheckResourceAttrPair("algolia_ingestion_transformation.test", "id", "algolia_ingestion_transformation.test", "transformation_id"),
				),
			},
			{
				// Changing the code (and renaming) exercises
				// UpdateTransformation.
				Config: testAccIngestionTransformationConfig(name+"-renamed", "function transform(record) { record.updated = true; return record; }"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_ingestion_transformation.test", "name", name+"-renamed"),
					resource.TestCheckResourceAttr("algolia_ingestion_transformation.test", "input", `{"code":"function transform(record) { record.updated = true; return record; }"}`),
				),
			},
			{
				ResourceName:      "algolia_ingestion_transformation.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccIngestionTransformationResource_switchToLegacyCode covers moving an
// existing input-based transformation to the legacy `code` attribute while
// omitting `type`. The API derives a type from `code`, and a provider that stored
// that derived value replayed it on the next update - sending a type alongside
// code, which the API rejects with "'input' is required if 'Type' is present".
// The third step is the proof it converges rather than merely applying once.
func TestAccIngestionTransformationResource_switchToLegacyCode(t *testing.T) {
	testAccRequireCredentials(t)

	name := fmt.Sprintf("tf-acc-transformation-switch-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	code := "function transform({ record }) { return record; }"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIngestionTransformationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIngestionTransformationConfig(name, code),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_ingestion_transformation.test", "type", "code"),
				),
			},
			{
				Config: testAccIngestionTransformationLegacyCodeConfig(name, code),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_ingestion_transformation.test", "code", code),
					// Null rather than the type the API derived: storing that is what
					// broke the update.
					resource.TestCheckNoResourceAttr("algolia_ingestion_transformation.test", "type"),
					resource.TestCheckNoResourceAttr("algolia_ingestion_transformation.test", "input"),
				),
			},
			{
				Config:   testAccIngestionTransformationLegacyCodeConfig(name, code),
				PlanOnly: true,
			},
		},
	})
}

func TestAccIngestionTransformationDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	name := fmt.Sprintf("tf-acc-transformation-ds-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIngestionTransformationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIngestionTransformationDataSourceConfig(name, "function transform(record) { return record; }"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.algolia_ingestion_transformation.test", "name", name),
					resource.TestCheckResourceAttr("data.algolia_ingestion_transformation.test", "type", "code"),
					resource.TestCheckResourceAttrPair(
						"data.algolia_ingestion_transformation.test", "transformation_id",
						"algolia_ingestion_transformation.test", "transformation_id",
					),
				),
			},
		},
	})
}

func testAccCheckIngestionTransformationDestroy(s *terraform.State) error {
	client, err := analyticsregion.NewIngestionClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"), os.Getenv(analyticsregion.EnvVar))
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_ingestion_transformation" {
			continue
		}

		transformationID := rs.Primary.Attributes["transformation_id"]
		if _, err := client.GetTransformation(client.NewApiGetTransformationRequest(transformationID)); err == nil {
			return fmt.Errorf("ingestion transformation %s still exists", transformationID)
		}
	}

	return nil
}

// testAccIngestionTransformationLegacyCodeConfig uses the deprecated top-level
// `code` attribute with no `type`, which is the only shape the API accepts for it.
func testAccIngestionTransformationLegacyCodeConfig(name, code string) string {
	return fmt.Sprintf(`
resource "algolia_ingestion_transformation" "test" {
  name = %[1]q
  code = %[2]q
}
`, name, code)
}

func testAccIngestionTransformationConfig(name, code string) string {
	// The API rejects `type` alongside a top-level `code` with no `input`
	// ("'input' is required if 'Type' is present"), so the code goes in `input`.
	// This is also the shape that exercises the TransformationInput union.
	return fmt.Sprintf(`
resource "algolia_ingestion_transformation" "test" {
  name = %[1]q
  type = "code"

  input = jsonencode({
    code = %[2]q
  })
}
`, name, code)
}

func testAccIngestionTransformationDataSourceConfig(name, code string) string {
	return testAccIngestionTransformationConfig(name, code) + `

data "algolia_ingestion_transformation" "test" {
  transformation_id = algolia_ingestion_transformation.test.transformation_id
}
`
}
