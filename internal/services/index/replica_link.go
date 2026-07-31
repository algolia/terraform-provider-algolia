package index

import (
	"context"
	"strings"
	"sync"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

// standardReplicasOnly is a validator for advanced.replicas rejecting a
// virtual(...) entry.
//
// The primary's replicas list has two owners, split by the kind of entry: this
// attribute owns standard replicas, and each algolia_virtual_index owns its own
// virtual(...) entry. The two sets are disjoint, which is what keeps them from
// overwriting one another - so an entry declared on the wrong side is a category
// error, caught here rather than becoming a silent tug of war.
func standardReplicasOnly() validator.List {
	return listvalidator.ValueStringsAre(virtualReplicaNameRejected{})
}

type virtualReplicaNameRejected struct{}

func (virtualReplicaNameRejected) Description(context.Context) string {
	return "must be a standard replica name; virtual replicas are declared with algolia_virtual_index"
}

func (v virtualReplicaNameRejected) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (virtualReplicaNameRejected) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	name := req.ConfigValue.ValueString()
	if !isVirtualReplicaName(name) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Virtual replica declared on the wrong resource",
		"The entry "+name+" names a virtual replica, and this list owns standard replicas only. "+
			"Declare it with an algolia_virtual_index resource instead, which manages its own entry in "+
			"this index's replicas - that is what stops the two from overwriting each other.\n\n"+
			"Remove this entry; the algolia_virtual_index resource keeps the replica linked on its own.",
	)
}

// mergeStandardReplicas builds the replicas list to write for a primary index:
// the standard replicas its configuration declares, plus the virtual(...) entries
// Algolia currently reports.
//
// Writing only the configured list would unlink every virtual replica, because the
// API takes this field as the complete set. Preserving the virtual entries is safe
// precisely because they are not this attribute's to declare - each belongs to an
// algolia_virtual_index resource, which is also how removing one is expressed, so
// nothing here is being made unremovable.
//
// Reading the primary is what makes this a read-modify-write, so callers hold the
// per-primary lock around it and the write that follows.
func mergeStandardReplicas(ctx context.Context, client *search.APIClient, indexName string, configured []string) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	settings, err := client.GetSettings(client.NewApiGetSettingsRequest(indexName), search.WithContext(ctx))
	if err != nil {
		if algoliaerr.IsNotFound(err) {
			// The index does not exist yet, so there is nothing to preserve. This is
			// the ordinary case on Create.
			return configured, diags
		}

		diags.AddError(algoliaerr.Object(indexKind, indexName).Message(algoliaerr.Read, err))

		return nil, diags
	}

	declared := make(map[string]struct{}, len(configured))
	for _, entry := range configured {
		declared[entry] = struct{}{}
	}

	merged := append([]string(nil), configured...)
	for _, entry := range settings.Replicas {
		if !isVirtualReplicaName(entry) {
			continue
		}

		// A configured standard entry whose virtual form is already linked is a
		// real disagreement, not something to merge: Algolia cannot hold one index
		// as a replica twice, once in each mode.
		if _, ok := declared[strings.TrimSuffix(strings.TrimPrefix(entry, "virtual("), ")")]; ok {
			diags.AddError(
				"Replica declared as both standard and virtual",
				"Index "+indexName+" lists "+entry+", so that replica is managed by an "+
					"algolia_virtual_index resource, but this index's advanced.replicas also declares it "+
					"as a standard replica. Algolia cannot hold one index as a replica in both modes.\n\n"+
					"Remove it from advanced.replicas to keep it virtual, or remove the "+
					"algolia_virtual_index resource to make it standard.",
			)

			return nil, diags
		}

		merged = append(merged, entry)
	}

	tflog.Debug(ctx, "Merged replicas for primary index", map[string]any{
		"name":       indexName,
		"configured": configured,
		"merged":     merged,
	})

	return merged, diags
}

// standardReplicasOf keeps only the standard entries of a primary's replicas list.
//
// advanced.replicas means "this index's standard replicas", so that is what it must
// read back as. Surfacing the virtual entries too would put values in state that the
// attribute's configuration never declares - and since they cannot be declared there,
// every refresh would plan a change that no apply could settle.
//
// Virtual replicas stay observable on their own algolia_virtual_index resources,
// which is where they are declared.
func standardReplicasOf(replicas []string) []string {
	if replicas == nil {
		return nil
	}

	standard := make([]string, 0, len(replicas))
	for _, entry := range replicas {
		if isVirtualReplicaName(entry) {
			continue
		}
		standard = append(standard, entry)
	}

	return standard
}
