//go:build e2e

package e2e

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestE2EReplicaClusterLifecycle drives one configuration through every stage a
// real user puts a provider through - create, drift, reconcile, update, destroy -
// against the live API, with a cluster of resources that share state rather than a
// single isolated resource.
//
// The per-resource acceptance suites cover each stage in isolation. What only a
// whole-cluster run can show is that the stages compose: that a primary index, a
// standard replica, a virtual replica, a rule and a synonym all survive an apply
// that touches several of them at once, and that reconciliation puts back exactly
// what drifted and nothing else.
//
// The primary's replicas list is the interesting part. It has two owners split by
// entry kind - advanced.replicas owns the standard names, each
// algolia_virtual_index owns its own virtual(...) entry - so every write to it is a
// merge, and every read of it is a filter. Drifting it in both directions at once
// (one entry removed, one added) is what proves the split holds.
func TestE2EReplicaClusterLifecycle(t *testing.T) {
	requireE2E(t)

	client := e2eSearchClient(t)
	names := newClusterNames()

	// The rogue replica is created by drifting the primary's replicas list, so
	// Terraform never manages it: reconciliation unlinks it but leaves the index
	// alone, which is the correct behaviour for an index the configuration does not
	// declare. Nothing else would ever remove it.
	t.Cleanup(func() { deleteIndexIfPresent(t, client, names.rogue) })

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		CheckDestroy:             checkClusterAbsent(client, names),
		Steps: []resource.TestStep{
			{ // Create.
				Config: clusterConfig(names, 20, false, "promote apple on shoes"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// State holds the standard entry only, while the API holds both
					// kinds. Both halves of the ownership split, in one assertion pair.
					resource.TestCheckResourceAttr("algolia_index.primary", "advanced.replicas.#", "1"),
					resource.TestCheckResourceAttr("algolia_index.primary", "advanced.replicas.0", names.standard),
					checkPrimaryReplicas(client, names.primary, names.standard, virtualEntry(names.virtual)),
					checkIndexHitsPerPage(client, names.primary, 20),
					checkRuleDescription(client, names.primary, ruleObjectID, "promote apple on shoes"),
					checkSynonyms(client, names.primary, synonymObjectID, "iphone", "ios phone"),
				),
			},
			{ // Drift, then reconcile. Same configuration as the step above.
				PreConfig: driftCluster(t, client, names),
				Config:    clusterConfig(names, 20, false, "promote apple on shoes"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						// Asserting the plan is what proves the drift was noticed. Without
						// these, a provider that silently adopted the drifted values would
						// still pass the checks below.
						plancheck.ExpectResourceAction("algolia_index.primary", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("algolia_synonym.phone", plancheck.ResourceActionUpdate),
						// Both were removed outright behind Terraform's back: the rule was
						// deleted, and unlinking the virtual replica is what stops Algolia
						// treating it as one, so Read drops each from state.
						plancheck.ExpectResourceAction("algolia_rule.promo", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("algolia_virtual_index.virtual", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// The rogue entry is gone and the virtual one is back, which can only
					// happen if the primary's write preserved an entry another resource
					// added during the same apply.
					checkPrimaryReplicas(client, names.primary, names.standard, virtualEntry(names.virtual)),
					checkIndexHitsPerPage(client, names.primary, 20),
					checkRuleDescription(client, names.primary, ruleObjectID, "promote apple on shoes"),
					checkSynonyms(client, names.primary, synonymObjectID, "iphone", "ios phone"),
				),
			},
			{ // Update: a second standard replica, a new page size, an edited rule.
				Config: clusterConfig(names, 50, true, "promote apple, updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("algolia_index.primary", "advanced.replicas.#", "2"),
					// Rewriting the standard list wholesale must still leave the virtual
					// entry linked.
					checkPrimaryReplicas(client, names.primary,
						names.standard, names.secondStandard, virtualEntry(names.virtual)),
					checkIndexHitsPerPage(client, names.primary, 50),
					checkRuleDescription(client, names.primary, ruleObjectID, "promote apple, updated"),
				),
			},
		},
	})
}

// TestE2EReplicaCreatedByTwoResources covers the configuration that has no
// dependency between a primary and a replica managed as its own algolia_index,
// because the replica is named as a literal string rather than referenced.
//
// Terraform creates the same index twice concurrently, and both halves of that go
// wrong in ways only the live API shows:
//
//   - Algolia restarts the index's task queue when it turns the index into a
//     replica, so the task ID the provider is waiting on never publishes even though
//     the write landed. The create used to hang for the full 30-minute budget and
//     then fail.
//   - Nothing orders the destroy either, and unlinking a replica means writing its
//     primary's settings - which is how an index gets created in Algolia. Unlinking
//     before attempting the delete recreated the primary the same destroy had just
//     removed, leaving an empty index behind. CheckDestroy is what catches that.
//
// Referencing the resource is still the better configuration, and the schema says
// so. This test is about the provider surviving the other one.
func TestE2EReplicaCreatedByTwoResources(t *testing.T) {
	requireE2E(t)

	client := e2eSearchClient(t)
	suffix := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	primary := fmt.Sprintf("tf-e2e-pair-primary-%s", suffix)
	replica := fmt.Sprintf("tf-e2e-pair-replica-%s", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			checkIndexAbsent(client, replica),
			// The one that regressed: a destroy that recreates this index reports
			// success while leaving it behind.
			checkIndexAbsent(client, primary),
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "algolia_index" "primary" {
  name                = %[1]q
  deletion_protection = false

  advanced {
    replicas = [%[2]q]
  }
}

resource "algolia_index" "replica" {
  name                = %[2]q
  deletion_protection = false

  pagination {
    hits_per_page = 77
  }
}
`, primary, replica),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkPrimaryReplicas(client, primary, replica),
					// The settings the re-sent write carried have to be the ones that
					// stuck, not the defaults of an index Algolia rebuilt as a replica.
					checkIndexHitsPerPage(client, replica, 77),
					resource.TestCheckResourceAttr("algolia_index.replica", "primary", primary),
				),
			},
		},
	})
}

const (
	ruleObjectID    = "e2e-promo-rule"
	synonymObjectID = "e2e-phone-syn"
)

// clusterNames holds the index names one run operates on. They carry a random
// suffix so concurrent runs against the same application cannot collide.
type clusterNames struct {
	primary        string
	standard       string
	secondStandard string
	virtual        string
	rogue          string
}

func newClusterNames() clusterNames {
	suffix := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	return clusterNames{
		primary:        fmt.Sprintf("tf-e2e-primary-%s", suffix),
		standard:       fmt.Sprintf("tf-e2e-standard-%s", suffix),
		secondStandard: fmt.Sprintf("tf-e2e-standard2-%s", suffix),
		virtual:        fmt.Sprintf("tf-e2e-virtual-%s", suffix),
		rogue:          fmt.Sprintf("tf-e2e-rogue-%s", suffix),
	}
}

func virtualEntry(replicaName string) string {
	return "virtual(" + replicaName + ")"
}

// clusterConfig renders the cluster. advanced.replicas references the replica
// resources rather than repeating their names, which is what orders the two writes
// that create each replica index: without the dependency Terraform runs them
// concurrently, and the plain index's own settings task then never publishes
// because Algolia restarts the queue when it turns the index into a replica. The
// same edge orders destroy correctly, since Algolia refuses to delete an index
// that is still listed as a replica.
func clusterConfig(names clusterNames, hitsPerPage int, secondStandard bool, ruleDescription string) string {
	replicas := "[algolia_index.standard.name]"
	second := ""
	if secondStandard {
		replicas = "[algolia_index.standard.name, algolia_index.standard_second.name]"
		second = fmt.Sprintf(`
resource "algolia_index" "standard_second" {
  name                = %q
  deletion_protection = false
}
`, names.secondStandard)
	}

	return fmt.Sprintf(`
resource "algolia_index" "primary" {
  name                = %[1]q
  deletion_protection = false

  attributes {
    searchable_attributes = ["name", "description"]
  }

  pagination {
    hits_per_page = %[4]d
  }

  advanced {
    replicas = %[5]s
  }
}

resource "algolia_index" "standard" {
  name                = %[2]q
  deletion_protection = false
}
%[6]s
resource "algolia_virtual_index" "virtual" {
  name                = %[3]q
  primary_index_name  = algolia_index.primary.name
  deletion_protection = false

  ranking {
    custom_ranking = ["asc(price)"]
  }
}

resource "algolia_rule" "promo" {
  index_name  = algolia_index.primary.name
  object_id   = %[7]q
  description = %[8]q
  enabled     = true

  conditions {
    pattern   = "shoes"
    anchoring = "contains"
  }

  consequence {
    params_json = jsonencode({ filters = "brand:apple" })
  }
}

resource "algolia_synonym" "phone" {
  index_name = algolia_index.primary.name
  object_id  = %[9]q
  type       = "synonym"
  synonyms   = ["iphone", "ios phone"]
}
`, names.primary, names.standard, names.virtual, hitsPerPage, replicas, second,
		ruleObjectID, ruleDescription, synonymObjectID)
}

// driftCluster changes every managed object behind Terraform's back, in the ways
// that actually happen: someone edits settings in the dashboard, a replica gets
// unlinked, a rule is deleted, a synonym is rewritten.
//
// The replicas list drifts in both directions at once - the virtual entry removed,
// an undeclared standard entry added - because the two are reconciled by different
// resources and only doing both at once shows they do not fight.
func driftCluster(t *testing.T, client *search.APIClient, names clusterNames) func() {
	t.Helper()

	return func() {
		t.Helper()

		setResp, err := client.SetSettings(client.NewApiSetSettingsRequest(names.primary, search.NewIndexSettings(
			search.WithIndexSettingsHitsPerPage(999),
			search.WithIndexSettingsReplicas([]string{names.standard, names.rogue}),
		)))
		if err != nil {
			t.Fatalf("drift settings of %s: %v", names.primary, err)
		}
		if _, err := client.WaitForTask(names.primary, setResp.TaskID); err != nil {
			t.Fatalf("wait for settings drift on %s: %v", names.primary, err)
		}

		ruleResp, err := client.DeleteRule(client.NewApiDeleteRuleRequest(names.primary, ruleObjectID))
		if err != nil {
			t.Fatalf("delete rule %s out of band: %v", ruleObjectID, err)
		}
		if _, err := client.WaitForTask(names.primary, ruleResp.TaskID); err != nil {
			t.Fatalf("wait for rule deletion on %s: %v", names.primary, err)
		}

		synonymResp, err := client.SaveSynonym(client.NewApiSaveSynonymRequest(names.primary, synonymObjectID,
			search.NewSynonymHit(synonymObjectID, search.SYNONYM_TYPE_SYNONYM,
				search.WithSynonymHitSynonyms([]string{"drifted", "values"}),
			)))
		if err != nil {
			t.Fatalf("rewrite synonym %s out of band: %v", synonymObjectID, err)
		}
		if _, err := client.WaitForTask(names.primary, synonymResp.TaskID); err != nil {
			t.Fatalf("wait for synonym drift on %s: %v", names.primary, err)
		}
	}
}

// checkPrimaryReplicas asserts the API reports exactly the given replicas list,
// order-insensitively. Exactness is the point: a check that only looked for the
// entries it wanted would pass while an undeclared one was still linked.
func checkPrimaryReplicas(client *search.APIClient, primaryIndexName string, want ...string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		settings, err := client.GetSettings(client.NewApiGetSettingsRequest(primaryIndexName))
		if err != nil {
			return fmt.Errorf("reading settings of %s: %w", primaryIndexName, err)
		}

		got := append([]string(nil), settings.Replicas...)
		sort.Strings(got)
		sort.Strings(want)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			return fmt.Errorf("index %s replicas = %v, want %v", primaryIndexName, got, want)
		}

		return nil
	}
}

func checkRuleDescription(client *search.APIClient, indexName, objectID, want string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		rule, err := client.GetRule(client.NewApiGetRuleRequest(indexName, objectID))
		if err != nil {
			return fmt.Errorf("reading rule %s on %s: %w", objectID, indexName, err)
		}
		if rule.Description == nil {
			return fmt.Errorf("rule %s on %s has no description, want %q", objectID, indexName, want)
		}
		if *rule.Description != want {
			return fmt.Errorf("rule %s on %s description = %q, want %q", objectID, indexName, *rule.Description, want)
		}

		return nil
	}
}

func checkSynonyms(client *search.APIClient, indexName, objectID string, want ...string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		hit, err := client.GetSynonym(client.NewApiGetSynonymRequest(indexName, objectID))
		if err != nil {
			return fmt.Errorf("reading synonym %s on %s: %w", objectID, indexName, err)
		}

		got := append([]string(nil), hit.Synonyms...)
		sort.Strings(got)
		sort.Strings(want)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			return fmt.Errorf("synonym %s on %s = %v, want %v", objectID, indexName, got, want)
		}

		return nil
	}
}

// checkClusterAbsent asserts destroy removed every index the configuration
// managed. The rogue replica is deliberately not checked: Terraform never managed
// it, so leaving it is correct.
func checkClusterAbsent(client *search.APIClient, names clusterNames) resource.TestCheckFunc {
	return resource.ComposeAggregateTestCheckFunc(
		checkIndexAbsent(client, names.primary),
		checkIndexAbsent(client, names.standard),
		checkIndexAbsent(client, names.secondStandard),
		checkIndexAbsent(client, names.virtual),
	)
}

// deleteIndexIfPresent removes an index left behind on purpose, tolerating its
// absence so a run that failed before creating it still cleans up quietly.
func deleteIndexIfPresent(t *testing.T, client *search.APIClient, indexName string) {
	t.Helper()

	if _, err := client.GetSettings(client.NewApiGetSettingsRequest(indexName)); err != nil {
		return
	}
	if _, err := client.DeleteIndex(client.NewApiDeleteIndexRequest(indexName)); err != nil {
		t.Logf("could not delete leftover index %s, delete it by hand: %v", indexName, err)
	}
}
