package index_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/terraform-provider-algolia/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var testAccVirtualIndexProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"algolia": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestAccVirtualIndexResource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	primaryIndexName := fmt.Sprintf("tf-virtual-primary-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	replicaName := fmt.Sprintf("tf-virtual-replica-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccVirtualIndexProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVirtualIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualIndexResourceConfig(primaryIndexName, replicaName, 80),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "name", replicaName),
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "primary_index_name", primaryIndexName),
				),
			},
			{
				Config: testAccVirtualIndexResourceConfig(primaryIndexName, replicaName, 60),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "primary_index_name", primaryIndexName),
				),
			},
			{
				ResourceName:                         "algolia_virtual_index.test",
				ImportState:                          true,
				ImportStateId:                        replicaName,
				ImportStateVerifyIdentifierAttribute: "name",
			},
			{
				Config: testAccVirtualIndexResourceConfig(primaryIndexName, replicaName, 60),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "name", replicaName),
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "primary_index_name", primaryIndexName),
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "ranking.relevancy_strictness", "60"),
				),
			},
		},
	})
}

func TestAccVirtualIndexResource_drift(t *testing.T) {
	testAccRequireCredentials(t)

	primaryIndexName := fmt.Sprintf("tf-virtual-drift-primary-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	replicaName := fmt.Sprintf("tf-virtual-drift-replica-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccVirtualIndexProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVirtualIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualIndexResourceConfig(primaryIndexName, replicaName, 80),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "ranking.relevancy_strictness", "80"),
				),
			},
			{
				PreConfig: testAccMutateVirtualIndex(t, replicaName, 25),
				Config:    testAccVirtualIndexResourceConfig(primaryIndexName, replicaName, 80),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "ranking.relevancy_strictness", "80"),
				),
			},
		},
	})
}

// TestAccVirtualIndexResource_unlinkedRecovery covers an index that still exists
// but has stopped being a virtual replica, which is what happens when the
// primary's replicas list is rewritten without its virtual(...) entry. Read used
// to raise an error for this, wedging plan, apply and destroy alike and leaving
// `terraform state rm` as the only way out. It should instead drop the resource
// and let the next apply re-link it.
func TestAccVirtualIndexResource_unlinkedRecovery(t *testing.T) {
	testAccRequireCredentials(t)

	primaryIndexName := fmt.Sprintf("tf-virtual-unlink-primary-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	replicaName := fmt.Sprintf("tf-virtual-unlink-replica-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccVirtualIndexProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVirtualIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualIndexResourceConfig(primaryIndexName, replicaName, 80),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "primary_index_name", primaryIndexName),
				),
			},
			{
				// Unlink the replica behind Terraform's back, then apply the same
				// configuration. The refresh must not error, and the plan that
				// follows must re-create the link rather than doing nothing.
				PreConfig: testAccUnlinkVirtualReplica(t, primaryIndexName),
				Config:    testAccVirtualIndexResourceConfig(primaryIndexName, replicaName, 80),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("algolia_virtual_index.test", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "primary_index_name", primaryIndexName),
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "ranking.relevancy_strictness", "80"),
					testAccCheckPrimaryHasVirtualReplicas(primaryIndexName, replicaName),
				),
			},
		},
	})
}

// TestAccVirtualIndexResource_multipleOnOnePrimary is the regression test for
// concurrent linking. Each of these three resources read-modify-writes the same
// primary's replicas list, and Terraform applies them in parallel, so without
// serialisation the later writes clobber the earlier ones and the configuration
// needs a second apply to converge. Both the replicas assertion and the
// post-apply empty-plan check would fail in that case.
func TestAccVirtualIndexResource_multipleOnOnePrimary(t *testing.T) {
	testAccRequireCredentials(t)

	suffix := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	primaryIndexName := fmt.Sprintf("tf-virtual-multi-primary-%s", suffix)
	replicaNames := []string{
		fmt.Sprintf("tf-virtual-multi-asc-%s", suffix),
		fmt.Sprintf("tf-virtual-multi-desc-%s", suffix),
		fmt.Sprintf("tf-virtual-multi-rank-%s", suffix),
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccVirtualIndexProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVirtualIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualIndexMultipleConfig(primaryIndexName, replicaNames),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPrimaryHasVirtualReplicas(primaryIndexName, replicaNames...),
				),
			},
		},
	})
}

// TestAccVirtualIndexResource_standardAndVirtualCoexist is the ownership rule end
// to end. The primary declares a standard replica in advanced.replicas while an
// algolia_virtual_index declares a virtual one, and both survive - which is only
// true because the two own disjoint parts of the same Algolia setting. Before that
// split, whichever applied last unlinked the other's replica.
func TestAccVirtualIndexResource_standardAndVirtualCoexist(t *testing.T) {
	testAccRequireCredentials(t)

	suffix := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	primaryIndexName := fmt.Sprintf("tf-virtual-coexist-primary-%s", suffix)
	standardName := fmt.Sprintf("tf-virtual-coexist-standard-%s", suffix)
	virtualName := fmt.Sprintf("tf-virtual-coexist-virtual-%s", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccVirtualIndexProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVirtualIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualIndexCoexistConfig(primaryIndexName, standardName, virtualName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPrimaryHasVirtualReplicas(primaryIndexName, virtualName),
					testAccCheckPrimaryHasStandardReplicas(primaryIndexName, standardName),
				),
			},
			{
				// A second apply must be a no-op. If the two resources disagreed
				// about the list, each would undo the other and this plan would never
				// be empty.
				Config:   testAccVirtualIndexCoexistConfig(primaryIndexName, standardName, virtualName),
				PlanOnly: true,
			},
		},
	})
}

// TestAccVirtualIndexResource_virtualEntryRejectedOnPrimary covers the other half:
// a virtual entry named in advanced.replicas is a category error, caught at plan
// time rather than becoming a silent tug of war over the setting.
func TestAccVirtualIndexResource_virtualEntryRejectedOnPrimary(t *testing.T) {
	testAccRequireCredentials(t)

	suffix := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	primaryIndexName := fmt.Sprintf("tf-virtual-rejected-primary-%s", suffix)
	replicaName := fmt.Sprintf("tf-virtual-rejected-replica-%s", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccVirtualIndexProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccVirtualIndexDeclaredByPrimaryConfig(primaryIndexName, replicaName),
				ExpectError: regexp.MustCompile(`Virtual replica declared on the wrong resource`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccVirtualIndexResource_undeclaredReplicasNotWritten covers the case where
// an algolia_index whose configuration says nothing about advanced.replicas
// nonetheless writes a replicas list, because the attribute is Optional+Computed
// and its plan value falls back to the last-refreshed state. That state goes
// stale the moment another resource links a replica, so an unrelated update to
// the primary would write the old list back and unlink the new replica.
//
// The ordering here is deliberate and load-bearing:
//
//   - "second" names its primary as a literal string, so no dependency edge ties
//     it to algolia_index.primary.
//   - algolia_index.primary declares depends_on = [second], putting its own
//     update after that link is created.
//   - "first" references algolia_index.primary.name, so it comes last.
//
// The refresh at the start of step 2 is what loads the primary's replicas into
// state as a known value (step 1 read it back before "first" was linked, so it
// was null then). That is precisely the real-world mechanism, and it is why this
// test fails without the fix: the primary writes ["virtual(first)"], omitting
// "second", which had just been linked.
func TestAccVirtualIndexResource_undeclaredReplicasNotWritten(t *testing.T) {
	testAccRequireCredentials(t)

	suffix := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	primaryIndexName := fmt.Sprintf("tf-virtual-undeclared-primary-%s", suffix)
	firstReplica := fmt.Sprintf("tf-virtual-undeclared-first-%s", suffix)
	secondReplica := fmt.Sprintf("tf-virtual-undeclared-second-%s", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccVirtualIndexProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVirtualIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualIndexUndeclaredReplicasConfig(primaryIndexName, firstReplica, "", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPrimaryHasVirtualReplicas(primaryIndexName, firstReplica),
				),
			},
			{
				// Adds the second replica and changes an unrelated setting inside the
				// primary's advanced block, forcing an Update that carries the
				// now-stale replicas list alongside it.
				Config: testAccVirtualIndexUndeclaredReplicasConfig(primaryIndexName, firstReplica, secondReplica, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_index.primary", "advanced.min_proximity", "2"),
					testAccCheckPrimaryHasVirtualReplicas(primaryIndexName, firstReplica, secondReplica),
				),
			},
		},
	})
}

// TestAccVirtualIndexResource_standardReplicaConversion covers a virtual replica
// that Algolia has converted into a standard one, which happens when the primary
// lists it under its plain name instead of the virtual(...) form. A standard
// replica holds its own copy of the primary's records, so this is the case where
// algolia_virtual_index would otherwise be quietly managing - and able to delete -
// a record-bearing index while reporting no drift at all.
func TestAccVirtualIndexResource_standardReplicaConversion(t *testing.T) {
	testAccRequireCredentials(t)

	suffix := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	primaryIndexName := fmt.Sprintf("tf-virtual-standard-primary-%s", suffix)
	replicaName := fmt.Sprintf("tf-virtual-standard-replica-%s", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccVirtualIndexProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVirtualIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualIndexResourceConfig(primaryIndexName, replicaName, 80),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPrimaryHasVirtualReplicas(primaryIndexName, replicaName),
				),
			},
			{
				// Convert it to a standard replica behind Terraform's back. Refreshing
				// must not fail: an error here would break plan, apply and destroy
				// together, and unlike the unlinked case no configuration edit could be
				// applied past it, because refresh runs first.
				PreConfig: testAccConvertToStandardReplica(t, primaryIndexName, replicaName),
				Config:    testAccVirtualIndexResourceConfig(primaryIndexName, replicaName, 80),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Still tracked, so Delete stays reachable and deletion_protection
					// still guards the records now sitting in it.
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "name", replicaName),
				),
			},
			{
				// Adopting it must fail while it is a standard replica. This is the step
				// that actually asserts the classification live: the refresh above stays
				// silent apart from a warning, which the test framework cannot assert on,
				// so without this step the whole test would still pass if the provider
				// went back to treating every replica as virtual.
				ResourceName:                         "algolia_virtual_index.test",
				ImportState:                          true,
				ImportStateId:                        replicaName,
				ImportStateVerifyIdentifierAttribute: "name",
				ExpectError:                          regexp.MustCompile(`standard replica`),
			},
			// The test ends with the index still a standard replica, so the
			// framework's own destroy exercises deleting one: Algolia refuses
			// deleteIndex while an index is still listed as a replica, and the
			// primary lists this one under its plain name. CheckDestroy confirms it
			// actually went away.
		},
	})
}

// TestAccVirtualIndexResource_standardReplicaRepair covers recovering from the
// conversion: the replica must go back to being a view over the primary's records.
//
// The repair belongs to the algolia_virtual_index resource, which relinks its own
// entry on every write. It is no longer expressed by naming the virtual(...) form
// in the primary's advanced.replicas - that list owns standard entries only and
// rejects the virtual form at plan time - so the trigger here is an ordinary edit
// to the virtual index itself.
//
// A conversion that nothing else edits therefore stays broken with a warning until
// the next write to the resource, because Read deliberately keeps the resource in
// state rather than planning a replacement that would delete the index. Closing
// that gap needs plan-time diagnostics (ModifyPlan), which the provider does not
// implement yet.
func TestAccVirtualIndexResource_standardReplicaRepair(t *testing.T) {
	testAccRequireCredentials(t)

	suffix := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	primaryIndexName := fmt.Sprintf("tf-virtual-repair-primary-%s", suffix)
	replicaName := fmt.Sprintf("tf-virtual-repair-replica-%s", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccVirtualIndexProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVirtualIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualIndexResourceConfig(primaryIndexName, replicaName, 80),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPrimaryHasVirtualReplicas(primaryIndexName, replicaName),
				),
			},
			{
				PreConfig: testAccConvertToStandardReplica(t, primaryIndexName, replicaName),
				// Same configuration apart from relevancy_strictness, so the only
				// reason an Update runs is that edit - and the relink it performs is
				// what this step asserts.
				Config: testAccVirtualIndexResourceConfig(primaryIndexName, replicaName, 60),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPrimaryHasVirtualReplicas(primaryIndexName, replicaName),
					resource.TestCheckResourceAttr("algolia_virtual_index.test", "ranking.relevancy_strictness", "60"),
				),
			},
		},
	})
}

// testAccConvertToStandardReplica rewrites the primary's replicas list to name the
// replica without the virtual() wrapper, which is what makes Algolia turn it into
// a standard replica and copy the primary's records into it.
func testAccConvertToStandardReplica(t *testing.T, primaryIndexName, replicaName string) func() {
	t.Helper()

	return func() {
		t.Helper()

		client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
		if err != nil {
			t.Fatalf("create Algolia client: %v", err)
		}

		setResp, err := client.SetSettings(client.NewApiSetSettingsRequest(primaryIndexName, search.NewIndexSettings(
			search.WithIndexSettingsReplicas([]string{replicaName}),
		)))
		if err != nil {
			t.Fatalf("convert replica %s to standard: %v", replicaName, err)
		}

		if _, err := client.WaitForTask(primaryIndexName, setResp.TaskID); err != nil {
			t.Fatalf("wait for conversion of %s: %v", replicaName, err)
		}
	}
}

// testAccUnlinkVirtualReplica clears the primary index's replicas list, which is
// how a virtual replica loses its primary in practice.
func testAccUnlinkVirtualReplica(t *testing.T, primaryIndexName string) func() {
	t.Helper()

	return func() {
		t.Helper()

		client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
		if err != nil {
			t.Fatalf("create Algolia client: %v", err)
		}

		setResp, err := client.SetSettings(client.NewApiSetSettingsRequest(primaryIndexName, search.NewIndexSettings(
			search.WithIndexSettingsReplicas([]string{}),
		)))
		if err != nil {
			t.Fatalf("unlink replicas of primary index %s: %v", primaryIndexName, err)
		}

		if _, err := client.WaitForTask(primaryIndexName, setResp.TaskID); err != nil {
			t.Fatalf("wait for unlink of primary index %s: %v", primaryIndexName, err)
		}
	}
}

// testAccCheckPrimaryHasVirtualReplicas asserts that the primary index lists
// every named replica in its virtual(...) form.
func testAccCheckPrimaryHasVirtualReplicas(primaryIndexName string, replicaNames ...string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
		if err != nil {
			return err
		}

		settings, err := client.GetSettings(client.NewApiGetSettingsRequest(primaryIndexName))
		if err != nil {
			return fmt.Errorf("read settings of primary index %s: %w", primaryIndexName, err)
		}

		present := make(map[string]struct{}, len(settings.Replicas))
		for _, entry := range settings.Replicas {
			present[entry] = struct{}{}
		}

		for _, replicaName := range replicaNames {
			wanted := "virtual(" + replicaName + ")"
			if _, ok := present[wanted]; !ok {
				return fmt.Errorf("primary index %s does not list %s; its replicas are %v",
					primaryIndexName, wanted, settings.Replicas)
			}
		}

		return nil
	}
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

func TestAccVirtualIndexDataSource_basic(t *testing.T) {
	testAccRequireCredentials(t)

	primaryIndexName := fmt.Sprintf("tf-virtual-ds-primary-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	replicaName := fmt.Sprintf("tf-virtual-ds-replica-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccVirtualIndexProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualIndexDataSourceConfig(primaryIndexName, replicaName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.algolia_virtual_index.test", "name", replicaName),
					resource.TestCheckResourceAttr("data.algolia_virtual_index.test", "primary_index_name", primaryIndexName),
				),
			},
		},
	})
}

func testAccCheckVirtualIndexDestroy(s *terraform.State) error {
	client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "algolia_virtual_index" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		_, err := client.GetSettings(client.NewApiGetSettingsRequest(name))
		if err == nil {
			return fmt.Errorf("virtual index %s still exists", name)
		}
	}

	return nil
}

func testAccMutateVirtualIndex(t *testing.T, replicaName string, strictness int32) func() {
	t.Helper()

	return func() {
		t.Helper()

		client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
		if err != nil {
			t.Fatalf("create Algolia client: %v", err)
		}

		setResp, err := client.SetSettings(client.NewApiSetSettingsRequest(replicaName, search.NewIndexSettings(
			search.WithIndexSettingsRelevancyStrictness(strictness),
			search.WithIndexSettingsCustomRanking([]string{"desc(popularity)"}),
		)))
		if err != nil {
			t.Fatalf("mutate virtual index %s: %v", replicaName, err)
		}

		testAccWaitForVirtualIndexMutation(t, client, replicaName, setResp.TaskID, strictness)
	}
}

func testAccWaitForVirtualIndexMutation(t *testing.T, client *search.APIClient, replicaName string, taskID int64, strictness int32) {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	for {
		task, err := client.GetTask(client.NewApiGetTaskRequest(replicaName, taskID))
		if err == nil && task.Status == search.TASK_STATUS_PUBLISHED {
			settings, err := client.GetSettings(client.NewApiGetSettingsRequest(replicaName))
			if err == nil && settings.GetRelevancyStrictness() == strictness {
				return
			}
		}

		if time.Now().After(deadline) {
			t.Fatalf("virtual index %s mutation did not become visible", replicaName)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func testAccVirtualIndexResourceConfig(primaryIndexName, replicaName string, strictness int) string {
	return fmt.Sprintf(`
resource "algolia_index" "primary" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_virtual_index" "test" {
  name                = %[2]q
  primary_index_name  = algolia_index.primary.name
  deletion_protection = false

  ranking {
    relevancy_strictness = %[3]d
    custom_ranking       = ["desc(popularity)"]
  }
}
`, primaryIndexName, replicaName, strictness)
}

func testAccVirtualIndexMultipleConfig(primaryIndexName string, replicaNames []string) string {
	config := fmt.Sprintf(`
resource "algolia_index" "primary" {
  name                = %[1]q
  deletion_protection = false
}
`, primaryIndexName)

	for i, replicaName := range replicaNames {
		config += fmt.Sprintf(`
resource "algolia_virtual_index" "replica_%[1]d" {
  name                = %[2]q
  primary_index_name  = algolia_index.primary.name
  deletion_protection = false

  ranking {
    custom_ranking = ["desc(popularity)"]
  }
}
`, i, replicaName)
	}

	return config
}

// testAccVirtualIndexUndeclaredReplicasConfig builds the ordering described on
// TestAccVirtualIndexResource_undeclaredReplicasNotWritten. Passing an empty
// secondReplica omits that resource, and with it the primary's depends_on.
func testAccVirtualIndexUndeclaredReplicasConfig(primaryIndexName, firstReplica, secondReplica string, minProximity int) string {
	dependsOn := ""
	second := ""
	if secondReplica != "" {
		dependsOn = "\n  depends_on = [algolia_virtual_index.second]\n"
		second = fmt.Sprintf(`
resource "algolia_virtual_index" "second" {
  name                = %[1]q
  primary_index_name  = %[2]q
  deletion_protection = false

  ranking {
    custom_ranking = ["desc(popularity)"]
  }
}
`, secondReplica, primaryIndexName)
	}

	// The advanced block must be present for this to reproduce: expandIndexSettings
	// skips the whole block when it is null, so an index with no advanced block at
	// all never writes replicas and the bug cannot appear. min_proximity is here
	// only to give the block a member and to change between steps, forcing Update.
	return fmt.Sprintf(`
resource "algolia_index" "primary" {
  name                = %[1]q
  deletion_protection = false
%[4]s
  advanced {
    min_proximity = %[5]d
  }
}

resource "algolia_virtual_index" "first" {
  name                = %[2]q
  primary_index_name  = algolia_index.primary.name
  deletion_protection = false

  ranking {
    custom_ranking = ["asc(price)"]
  }
}
%[3]s
`, primaryIndexName, firstReplica, second, dependsOn, minProximity)
}

// testAccVirtualIndexCoexistConfig declares the standard replica as its own
// algolia_index and references that resource from advanced.replicas rather than
// repeating the name as a literal.
//
// The reference is load-bearing, not style. Both writes create the same index -
// one directly, one as a side effect of linking it as a replica - and Terraform
// applies resources with no dependency between them concurrently. When the two
// land together, the plain index's own SetSettings task never reaches published,
// so its Create waits out the full 30-minute budget: Algolia appears to restart
// the index's task queue when it turns it into a replica, discarding the earlier
// task. Referencing the resource orders the two writes, which is how a
// configuration expresses this in the first place.
func testAccVirtualIndexCoexistConfig(primaryIndexName, standardName, virtualName string) string {
	return fmt.Sprintf(`
resource "algolia_index" "primary" {
  name                = %[1]q
  deletion_protection = false

  advanced {
    replicas = [algolia_index.standard_replica.name]
  }
}

resource "algolia_index" "standard_replica" {
  name                = %[2]q
  deletion_protection = false
}

resource "algolia_virtual_index" "test" {
  name                = %[3]q
  primary_index_name  = algolia_index.primary.name
  deletion_protection = false

  ranking {
    custom_ranking = ["asc(price)"]
  }
}
`, primaryIndexName, standardName, virtualName)
}

// testAccCheckPrimaryHasStandardReplicas asserts the primary lists each named
// replica under its plain name, i.e. not in the virtual(...) form.
func testAccCheckPrimaryHasStandardReplicas(primaryIndexName string, replicaNames ...string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client, err := search.NewClient(os.Getenv("ALGOLIA_APP_ID"), os.Getenv("ALGOLIA_API_KEY"))
		if err != nil {
			return err
		}

		settings, err := client.GetSettings(client.NewApiGetSettingsRequest(primaryIndexName))
		if err != nil {
			return fmt.Errorf("read settings of primary index %s: %w", primaryIndexName, err)
		}

		present := make(map[string]struct{}, len(settings.Replicas))
		for _, entry := range settings.Replicas {
			present[entry] = struct{}{}
		}

		for _, replicaName := range replicaNames {
			if _, ok := present[replicaName]; !ok {
				return fmt.Errorf("primary index %s does not list %s as a standard replica; its replicas are %v",
					primaryIndexName, replicaName, settings.Replicas)
			}
		}

		return nil
	}
}

func testAccVirtualIndexDeclaredByPrimaryConfig(primaryIndexName, replicaName string) string {
	return fmt.Sprintf(`
resource "algolia_index" "primary" {
  name                = %[1]q
  deletion_protection = false

  advanced {
    replicas = ["virtual(%[2]s)"]
  }
}

resource "algolia_virtual_index" "test" {
  name                = %[2]q
  primary_index_name  = algolia_index.primary.name
  deletion_protection = false

  ranking {
    custom_ranking = ["asc(price)"]
  }
}
`, primaryIndexName, replicaName)
}

func testAccVirtualIndexDataSourceConfig(primaryIndexName, replicaName string) string {
	return fmt.Sprintf(`
resource "algolia_index" "primary" {
  name                = %[1]q
  deletion_protection = false
}

resource "algolia_virtual_index" "test" {
  name                = %[2]q
  primary_index_name  = algolia_index.primary.name
  deletion_protection = false

  ranking {
    relevancy_strictness = 80
  }
}

data "algolia_virtual_index" "test" {
  name = algolia_virtual_index.test.name
}
`, primaryIndexName, replicaName)
}
