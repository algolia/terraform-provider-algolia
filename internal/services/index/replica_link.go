package index

import (
	"context"
	"strings"
	"sync"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// A primary index's `replicas` setting is written by two resources:
// algolia_index sets the whole list from `advanced.replicas`, while
// algolia_virtual_index adds or removes a single `virtual(...)` entry. They
// share one API field, so the rules for touching it live here rather than in
// either resource.

// virtualReplicaName renders the `virtual(...)` form Algolia uses to mark a
// replica as virtual inside a primary index's replicas list.
func virtualReplicaName(replicaName string) string {
	return "virtual(" + replicaName + ")"
}

// isVirtualReplicaName reports whether an entry in a replicas list denotes a
// virtual replica rather than a standard one.
func isVirtualReplicaName(entry string) bool {
	return strings.HasPrefix(entry, "virtual(") && strings.HasSuffix(entry, ")")
}

// replicaLinkage describes how a primary index lists one of its replicas.
//
// It exists because a replica's own settings cannot answer the question: Algolia
// reports a primary index for standard and virtual replicas alike, so `primary`
// being set proves only that the index is a replica of something. The
// virtual(...) marker lives solely in the primary's replicas list, which is why
// classifying a replica costs a second read.
type replicaLinkage int

const (
	// replicaLinkagePrimaryAbsent: the primary index does not exist.
	replicaLinkagePrimaryAbsent replicaLinkage = iota
	// replicaLinkageUnlisted: the primary exists but does not list the replica.
	replicaLinkageUnlisted
	// replicaLinkageStandard: listed under its plain name, so Algolia keeps it as
	// a standard replica and copies the primary's records into it.
	replicaLinkageStandard
	// replicaLinkageVirtual: listed as virtual(name), so it is a view over the
	// primary's records.
	replicaLinkageVirtual
)

// classifyReplicaLinkage reports how primaryIndexName lists replicaName.
func classifyReplicaLinkage(ctx context.Context, client *search.APIClient, primaryIndexName, replicaName string) (replicaLinkage, error) {
	settings, err := client.GetSettings(client.NewApiGetSettingsRequest(primaryIndexName), search.WithContext(ctx))
	if err != nil {
		if algoliaerr.IsNotFound(err) {
			return replicaLinkagePrimaryAbsent, nil
		}

		return replicaLinkagePrimaryAbsent, err
	}

	virtualName := virtualReplicaName(replicaName)
	for _, entry := range settings.Replicas {
		switch entry {
		case virtualName:
			return replicaLinkageVirtual, nil
		case replicaName:
			return replicaLinkageStandard, nil
		}
	}

	return replicaLinkageUnlisted, nil
}

// primaryReplicaLocks serialises the read-modify-write of one primary index's
// replicas list.
//
// Every algolia_virtual_index on the same primary mutates that single list, and
// Terraform applies resources concurrently (ten at a time by default). Without
// this, two replicas linking at once both read the list as it was before either
// wrote, and the second write drops the first one's entry - which is why a
// configuration declaring several virtual replicas on one primary needed a
// second apply to converge.
//
// The lock covers a single provider process, which is the case that matters: one
// terraform apply. Concurrent applies against the same primary from separate
// processes stay racy, and no provider-side lock can change that.
var primaryReplicaLocks sync.Map

// lockPrimaryReplicas takes the lock for a primary index and returns the release
// function, for use as `defer lockPrimaryReplicas(name)()`.
func lockPrimaryReplicas(primaryIndexName string) func() {
	value, _ := primaryReplicaLocks.LoadOrStore(primaryIndexName, &sync.Mutex{})

	mu := value.(*sync.Mutex)
	mu.Lock()

	return mu.Unlock
}

// ensureVirtualReplicaLinked adds replicaName to primaryIndexName's replicas
// list in the `virtual(...)` form, unless it is already there.
func ensureVirtualReplicaLinked(ctx context.Context, client *search.APIClient, primaryIndexName, replicaName string) error {
	defer lockPrimaryReplicas(primaryIndexName)()

	settings, err := client.GetSettings(client.NewApiGetSettingsRequest(primaryIndexName), search.WithContext(ctx))
	if err != nil {
		return err
	}

	virtualName := virtualReplicaName(replicaName)
	replicas := make([]string, 0, len(settings.Replicas)+1)
	alreadyVirtual := false
	wasStandard := false

	for _, existing := range settings.Replicas {
		switch existing {
		case virtualName:
			alreadyVirtual = true
			replicas = append(replicas, existing)
		case replicaName:
			// Listed under its plain name, which makes it a standard replica holding
			// its own copy of the records. Drop that entry rather than adding the
			// virtual form beside it: a primary listing the same index twice, once in
			// each mode, is not a state Algolia can honour.
			wasStandard = true
		default:
			replicas = append(replicas, existing)
		}
	}

	if alreadyVirtual && !wasStandard {
		return nil
	}
	if !alreadyVirtual {
		replicas = append(replicas, virtualName)
	}

	setResp, err := client.SetSettings(client.NewApiSetSettingsRequest(primaryIndexName, search.NewIndexSettings(
		search.WithIndexSettingsReplicas(replicas),
	)), search.WithContext(ctx))
	if err != nil {
		return err
	}

	return waitForIndexTask(ctx, client, primaryIndexName, setResp.TaskID)
}

// removeVirtualReplicaLink drops replicaName from primaryIndexName's replicas
// list, tolerating a primary that no longer exists.
//
// Both forms of the entry are removed, not just virtual(...). Algolia refuses
// `deleteIndex` on an index that is still a replica, so leaving a plain-name entry
// behind - which is how the primary lists this index once Algolia has converted it
// to a standard replica - makes the delete that follows fail with a bare
// "cannot apply the deleteIndex operation on a replica index". The index is going
// away, so the primary must stop listing it in any form.
func removeVirtualReplicaLink(ctx context.Context, client *search.APIClient, primaryIndexName, replicaName string) error {
	defer lockPrimaryReplicas(primaryIndexName)()

	settings, err := client.GetSettings(client.NewApiGetSettingsRequest(primaryIndexName), search.WithContext(ctx))
	if err != nil {
		if algoliaerr.IsNotFound(err) {
			return nil
		}
		return err
	}

	virtualName := virtualReplicaName(replicaName)
	replicas := append([]string(nil), settings.Replicas...)
	filtered := replicas[:0]
	changed := false
	for _, existing := range replicas {
		if existing == virtualName || existing == replicaName {
			changed = true
			continue
		}
		filtered = append(filtered, existing)
	}
	if !changed {
		return nil
	}

	setResp, err := client.SetSettings(client.NewApiSetSettingsRequest(primaryIndexName, search.NewIndexSettings(
		search.WithIndexSettingsReplicas(filtered),
	)), search.WithContext(ctx))
	if err != nil {
		return err
	}

	return waitForIndexTask(ctx, client, primaryIndexName, setResp.TaskID)
}

// replicasDeclared reports whether `advanced.replicas` is set in configuration,
// as opposed to merely being present in the plan.
//
// The distinction matters because `replicas` is Optional+Computed with a
// useStateForKnownList() plan modifier: when configuration omits it, the plan
// value falls back to the last-refreshed state value instead of staying null.
// Expanding the plan therefore yields a list even for an index whose
// configuration says nothing about replicas, and writing that back makes
// Terraform an authoritative writer of a field the user never declared - which
// silently unlinks any virtual replica added since that refresh. Only a list the
// configuration actually declares may be written; otherwise the field is omitted
// and Algolia keeps whatever the index already has, which is what
// Optional+Computed is supposed to mean.
//
// The plan modifier itself is left alone deliberately: leaving `replicas` unknown
// when unconfigured would report it as "known after apply" on every plan of every
// index, so the fix belongs at the write, not in the diff.
func replicasDeclared(ctx context.Context, config tfsdk.Config, diags *diag.Diagnostics) bool {
	var advanced types.Object
	diags.Append(config.GetAttribute(ctx, path.Root("advanced"), &advanced)...)
	if diags.HasError() || advanced.IsNull() || advanced.IsUnknown() {
		return false
	}

	attribute, ok := advanced.Attributes()["replicas"]
	if !ok {
		return false
	}

	list, ok := attribute.(types.List)

	return ok && !list.IsNull() && !list.IsUnknown()
}

// preserveUndeclaredReplicas keeps the planned value of advanced.replicas in the
// applied state in place of what the read-back returned.
//
// This is the other half of not writing an undeclared replicas list. The plan
// value for such a list still comes from prior state, so when another resource
// links a virtual replica during the same apply the read-back comes back longer
// than the plan promised - and Terraform rejects an applied value that gained an
// element the plan did not contain ("Provider produced inconsistent result after
// apply"). Keeping the planned value honours that contract.
//
// The state is then briefly behind reality, which is harmless precisely because
// the field is no longer written when undeclared: the next refresh picks up the
// real list and nothing was ever sent to Algolia from the stale one.
func preserveUndeclaredReplicas(planned types.Object, applied *types.Object) diag.Diagnostics {
	var diags diag.Diagnostics

	if planned.IsNull() || planned.IsUnknown() || applied.IsNull() || applied.IsUnknown() {
		return diags
	}

	plannedReplicas, ok := planned.Attributes()["replicas"]
	if !ok || plannedReplicas.IsUnknown() {
		// An unknown planned value places no constraint on the result, and writing
		// one into state would itself be rejected.
		return diags
	}

	attributes := applied.Attributes()
	if _, ok := attributes["replicas"]; !ok {
		return diags
	}
	attributes["replicas"] = plannedReplicas

	updated, d := types.ObjectValue(advancedAttrTypes, attributes)
	diags.Append(d...)
	if !diags.HasError() {
		*applied = updated
	}

	return diags
}

// droppedVirtualReplicas returns the virtual replica entries present in current
// but missing from planned.
func droppedVirtualReplicas(current, planned []string) []string {
	kept := make(map[string]struct{}, len(planned))
	for _, entry := range planned {
		kept[entry] = struct{}{}
	}

	var dropped []string
	for _, entry := range current {
		if !isVirtualReplicaName(entry) {
			continue
		}
		if _, ok := kept[entry]; ok {
			continue
		}
		dropped = append(dropped, entry)
	}

	return dropped
}

// warnDroppedVirtualReplicas reports virtual replica links that a wholesale
// write of a primary index's replicas list is about to remove.
//
// The removal itself is honoured: an explicit `advanced.replicas` list is a
// complete declaration of what the primary's replicas should be, and quietly
// merging omitted entries back in would make removing a virtual replica through
// the primary impossible to express. But it is not allowed to happen silently,
// because unlinking a virtual replica empties it - a virtual replica is a view
// over the primary's records, not a copy of them - and the user who wrote the
// list may not realise another resource is contributing to it.
//
// This check never fails an apply on its own: if the primary's settings cannot
// be read (most often because the index does not exist yet, which is the normal
// case during Create) there is nothing to warn about.
func warnDroppedVirtualReplicas(ctx context.Context, client *search.APIClient, indexName string, planned []string, diags *diag.Diagnostics) {
	if planned == nil {
		// replicas is not configured, so this write leaves the list alone.
		return
	}

	settings, err := client.GetSettings(client.NewApiGetSettingsRequest(indexName), search.WithContext(ctx))
	if err != nil {
		tflog.Debug(ctx, "Could not read current replicas to check for dropped virtual replicas", map[string]any{
			"name":  indexName,
			"error": err.Error(),
		})

		return
	}

	dropped := droppedVirtualReplicas(settings.Replicas, planned)
	if len(dropped) == 0 {
		return
	}

	diags.AddWarning(
		"Virtual replica links removed",
		// Deliberately "about to write" rather than "configured": replicas is
		// Optional+Computed, so when it is absent from configuration the plan value
		// falls back to the last-refreshed state value and gets written back. The
		// list being sent is then not one the user wrote at all, and blaming their
		// configuration for it would send them looking for something that is not there.
		"The replicas list Terraform is about to write to index "+indexName+" omits "+
			strings.Join(dropped, ", ")+", which Algolia currently reports for it. Writing it "+
			"unlinks those replicas, and an unlinked virtual replica is empty: it is a view "+
			"over the primary index's records, not a copy of them.\n\n"+
			"If they are managed by algolia_virtual_index resources, list them in this index's "+
			"advanced.replicas as well. Both resources write the same Algolia field, so "+
			"whichever applies last decides what the primary ends up with.",
	)
}
