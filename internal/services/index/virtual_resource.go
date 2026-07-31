package index

import (
	"context"
	"fmt"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"
	"github.com/algolia/terraform-provider-algolia/internal/deletionprotection"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &virtualIndexResource{}
	_ resource.ResourceWithConfigure   = &virtualIndexResource{}
	_ resource.ResourceWithImportState = &virtualIndexResource{}
	_ resource.ResourceWithModifyPlan  = &virtualIndexResource{}
)

type virtualIndexResource struct {
	client *search.APIClient
}

// virtualIndexKind names this resource inside a diagnostic sentence. Diagnostics
// about the underlying index rather than the virtual-replica resource use
// indexKind, so that they read identically to algolia_index's own.
const virtualIndexKind = "virtual index"

func NewVirtualResource() resource.Resource {
	return &virtualIndexResource{}
}

func (r *virtualIndexResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_index"
}

func (r *virtualIndexResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = virtualIndexResourceSchema()
}

func (r *virtualIndexResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*providertypes.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *providertypes.ProviderData, got: %T", req.ProviderData),
		)
		return
	}

	r.client = data.Client
}

func (r *virtualIndexResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan VirtualIndexResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := plan.Name.ValueString()
	primaryIndexName := plan.PrimaryIndexName.ValueString()
	tflog.Debug(ctx, "Creating virtual index", map[string]any{"name": indexName, "primary_index_name": primaryIndexName})

	if err := ensureVirtualReplicaLinked(ctx, r.client, primaryIndexName, indexName); err != nil {
		resp.Diagnostics.AddError("Error linking virtual replica", "Could not link virtual replica "+indexName+" to primary index "+primaryIndexName+": "+err.Error())
		return
	}

	indexPlan := virtualToIndexModel(plan)
	nullBlocks := captureNullBlocks(&indexPlan)
	settings, diags := expandIndexSettings(ctx, &indexPlan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	setResp, err := r.client.SetSettings(r.client.NewApiSetSettingsRequest(indexName, settings), search.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error creating virtual index", "Could not configure virtual index "+indexName+": "+err.Error())
		return
	}

	if err = waitForSettingsWrite(ctx, r.client, indexName, settings, setResp.TaskID); err != nil {
		resp.Diagnostics.AddError("Error waiting for virtual index creation", "Could not wait for task: "+err.Error())
		return
	}

	planSnapshot := indexPlan
	readState, diags := r.readVirtualIndex(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	switch readState {
	case virtualIndexAbsent:
		resp.Diagnostics.AddError("Error reading index", "Virtual index "+indexName+" could not be found immediately after it was created.")
		return
	case virtualIndexUnlinked:
		// The link was just written and confirmed, so losing it here is not
		// ordinary drift: something else unlinked the replica during this apply.
		resp.Diagnostics.AddError("Index is not a virtual replica", unlinkedVirtualIndexDetail(indexName, primaryIndexName))
		return
	case virtualIndexStandardReplica:
		resp.Diagnostics.AddError("Index is a standard replica", standardReplicaDetail(indexName, primaryIndexName))
		return
	}

	indexState := virtualToIndexModel(plan)
	resp.Diagnostics.Append(preservePlannedValues(ctx, &planSnapshot, &indexState)...)
	virtualFromIndexModel(indexState, &plan)
	restoreVirtualNullBlocks(&plan, nullBlocks)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *virtualIndexResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state VirtualIndexResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexState := virtualToIndexModel(state)
	nullBlocks := captureNullBlocks(&indexState)
	stateSnapshot := indexState
	priorPrimaryIndexName := state.PrimaryIndexName.ValueString()

	readState, diags := r.readVirtualIndex(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	switch readState {
	case virtualIndexAbsent:
		// Deleted outside Terraform: drop it from state so the next plan recreates
		// it, rather than wedging plan, apply and destroy behind a read error.
		tflog.Warn(ctx, "Virtual index not found; removing from state", map[string]any{"name": state.Name.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	case virtualIndexUnlinked:
		// The index is still there but is no longer a replica of anything, so it
		// cannot be described by this resource's state. Drop it for the same
		// reason as absence: raising an error here wedges plan, apply and destroy
		// alike, leaving `state rm` as the only way out.
		//
		// The alternatives are worse. Nulling primary_index_name keeps the resource
		// in state but forces replacement, which deletion_protection - on by
		// default - then refuses, wedging apply again. Leaving state untouched
		// produces no diff at all, so the link would never be repaired. Dropping it
		// makes the next apply call Create, which re-links the replica in place.
		//
		// The cost is that Terraform no longer tracks an index that still exists,
		// so a destroy run now would leave it behind. The warning says so, because
		// the index holding no records of its own makes that easy to overlook.
		resp.Diagnostics.AddWarning(
			"Virtual index is no longer a virtual replica",
			unlinkedVirtualIndexDetail(state.Name.ValueString(), priorPrimaryIndexName)+
				"\n\nIt has been removed from Terraform state, so the next apply will re-link it. "+
				"Note that the index itself still exists: because Terraform no longer tracks it, "+
				"a destroy run before that apply will leave it in place.",
		)
		resp.State.RemoveResource(ctx)
		return
	case virtualIndexStandardReplica:
		// Deliberately not an error and not a state removal. An error would fail
		// every refresh, and so plan, apply and destroy with it - the same wedge the
		// unlinked case above exists to avoid, and worse here because refresh runs
		// before configuration is considered, so no edit to the primary's replicas
		// could be applied past it. Removing it from state would make Terraform
		// forget an index that now holds a full copy of the primary's records.
		//
		// Keeping it in state leaves Delete reachable, so deletion_protection still
		// guards those records, and ModifyPlan turns the conversion into a planned
		// relink rather than something the operator has to notice and undo by hand.
		// The primary read back from the API, not the prior state's: reaching this
		// case means Algolia reported one, and that is the index to name.
		resp.Diagnostics.AddWarning(
			"Virtual index is a standard replica",
			standardReplicaDetail(state.Name.ValueString(), state.PrimaryIndexName.ValueString())+
				"\n\nThe next apply relinks it as a virtual replica, which Algolia does by dropping "+
				"the copied records. To keep the copy, manage this index with algolia_index instead.",
		)
	}

	indexRead := virtualToIndexModel(state)
	resp.Diagnostics.Append(preservePlannedValues(ctx, &stateSnapshot, &indexRead)...)
	virtualFromIndexModel(indexRead, &state)
	restoreVirtualNullBlocks(&state, nullBlocks)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ModifyPlan plans the repair of a replica Algolia has turned into a standard one.
//
// Read keeps such an index in state and only warns, for the reasons set out there:
// dropping it would plan a create, and the index now holds its own copy of the
// primary's records, so anything replacement-shaped risks data the resource is
// documented as merely viewing. The cost of that choice used to be that nothing
// brought the replica back on its own. The warning repeated on every refresh, and a
// configuration with no other change had nothing to apply, so the repair waited for
// an unrelated edit.
//
// Planning it here closes that gap: an unchanged configuration still shows an
// update, and Update relinks the entry in its virtual(...) form. The price is one
// read of the primary index per plan, on top of the one Read already makes -
// classifying a replica needs the primary's list, since the replica's own settings
// report a primary whichever kind it is.
func (r *virtualIndexResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Nothing to repair while creating, and a destroy is about to remove the entry
	// rather than restore it.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var state VirtualIndexResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := state.Name.ValueString()
	primaryIndexName := state.PrimaryIndexName.ValueString()
	if indexName == "" || primaryIndexName == "" {
		return
	}

	linkage, err := classifyReplicaLinkage(ctx, r.client, primaryIndexName, indexName)
	if err != nil {
		// A plan must not fail on a read the apply will make again anyway. Say why the
		// check was skipped instead, so a plan that misses a needed relink is not
		// silent about it.
		resp.Diagnostics.AddWarning(
			"Could not check the replica linkage of "+indexName,
			"Reading the replicas of primary index "+primaryIndexName+" failed, so this plan cannot "+
				"tell whether "+indexName+" is still a virtual replica: "+err.Error()+
				"\n\nPlan again once the API is reachable.",
		)

		return
	}
	if linkage != replicaLinkageStandard {
		return
	}

	tflog.Debug(ctx, "Planning a relink of a virtual replica Algolia keeps as a standard one", map[string]any{
		"name":               indexName,
		"primary_index_name": primaryIndexName,
	})

	// updated_at is Computed and the relink writes settings, so leaving it unknown is
	// both true and enough to turn an otherwise empty plan into an in-place update.
	// The warning Read raises explains why.
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("updated_at"), types.StringUnknown())...)
}

func (r *virtualIndexResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan VirtualIndexResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := plan.Name.ValueString()
	primaryIndexName := plan.PrimaryIndexName.ValueString()
	if err := ensureVirtualReplicaLinked(ctx, r.client, primaryIndexName, indexName); err != nil {
		resp.Diagnostics.AddError("Error linking virtual replica", "Could not confirm virtual replica linkage: "+err.Error())
		return
	}

	indexPlan := virtualToIndexModel(plan)
	nullBlocks := captureNullBlocks(&indexPlan)
	settings, diags := expandIndexSettings(ctx, &indexPlan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	setResp, err := r.client.SetSettings(r.client.NewApiSetSettingsRequest(indexName, settings), search.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(algoliaerr.Object(virtualIndexKind, indexName).Message(algoliaerr.Update, err))
		return
	}

	if err = waitForSettingsWrite(ctx, r.client, indexName, settings, setResp.TaskID); err != nil {
		resp.Diagnostics.AddError("Error waiting for virtual index update", "Could not wait for task: "+err.Error())
		return
	}

	planSnapshot := indexPlan
	readState, diags := r.readVirtualIndex(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	switch readState {
	case virtualIndexAbsent:
		resp.Diagnostics.AddError("Error reading index", "Virtual index "+indexName+" could not be found immediately after it was updated.")
		return
	case virtualIndexUnlinked:
		resp.Diagnostics.AddError("Index is not a virtual replica", unlinkedVirtualIndexDetail(indexName, primaryIndexName))
		return
	case virtualIndexStandardReplica:
		resp.Diagnostics.AddError("Index is a standard replica", standardReplicaDetail(indexName, primaryIndexName))
		return
	}

	indexState := virtualToIndexModel(plan)
	resp.Diagnostics.Append(preservePlannedValues(ctx, &planSnapshot, &indexState)...)
	virtualFromIndexModel(indexState, &plan)
	restoreVirtualNullBlocks(&plan, nullBlocks)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *virtualIndexResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state VirtualIndexResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := state.Name.ValueString()
	if deletionprotection.Enabled(state.DeletionProtection) {
		resp.Diagnostics.Append(deletionprotection.Refuse(fmt.Sprintf("virtual index %q", indexName)))
		return
	}

	if err := deleteIndexWithUnlink(ctx, r.client, indexName); err != nil {
		resp.Diagnostics.AddError(algoliaerr.Object(virtualIndexKind, indexName).Message(algoliaerr.Delete, err))
		return
	}

	if err := confirmIndexDeleted(ctx, r.client, indexName); err != nil {
		resp.Diagnostics.AddError("Index still exists after deletion", deleteNotConfirmedDetail(indexName, err))
	}
}

func (r *virtualIndexResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var state VirtualIndexResourceModel
	state.Name = types.StringValue(req.ID)
	state.DeletionProtection = types.BoolValue(true)

	readState, diags := r.readVirtualIndex(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	switch readState {
	case virtualIndexAbsent:
		// Unlike Read, an import of an absent index must fail loudly.
		resp.Diagnostics.AddError("Error reading index", "Could not read index "+req.ID+": the index does not exist.")
		return
	case virtualIndexUnlinked:
		// Also loud: importing an index that is not a virtual replica is a
		// mistake in the import command, not drift to be reconciled.
		resp.Diagnostics.AddError(
			"Index is not a virtual replica",
			"Index "+req.ID+" exists but reports no primary index, so it cannot be managed as "+
				"algolia_virtual_index. Import it as algolia_index instead.",
		)
		return
	case virtualIndexStandardReplica:
		// Also loud: adopting a standard replica as a virtual one would put a
		// record-bearing index under a resource that promises a view.
		resp.Diagnostics.AddError(
			"Index is a standard replica",
			standardReplicaDetail(req.ID, state.PrimaryIndexName.ValueString()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// virtualIndexState describes what readVirtualIndex found. "Exists but is no
// longer a virtual replica" is a state of its own because the right response to
// it differs per caller: recoverable drift during Read, an outright failure
// anywhere else.
type virtualIndexState int

const (
	// virtualIndexAbsent: no index by that name exists.
	virtualIndexAbsent virtualIndexState = iota
	// virtualIndexFound: the index exists and the primary lists it as virtual.
	virtualIndexFound
	// virtualIndexUnlinked: the index exists but is a replica of nothing.
	virtualIndexUnlinked
	// virtualIndexStandardReplica: the index exists and is a replica, but a
	// standard one, so Algolia has copied the primary's records into it.
	virtualIndexStandardReplica
)

// readVirtualIndex populates model from the Algolia API, reporting what it
// found. See readIndex for why absence is returned rather than raised.
func (r *virtualIndexResource) readVirtualIndex(ctx context.Context, model *VirtualIndexResourceModel) (virtualIndexState, diag.Diagnostics) {
	indexModel := virtualToIndexModel(*model)
	found, diags := r.readIndexModel(ctx, &indexModel)
	if diags.HasError() {
		return virtualIndexAbsent, diags
	}
	if !found {
		return virtualIndexAbsent, diags
	}

	virtualFromIndexModel(indexModel, model)

	indexName := model.Name.ValueString()
	primaryIndexName := model.PrimaryIndexName.ValueString()
	if model.PrimaryIndexName.IsNull() || primaryIndexName == "" {
		return virtualIndexUnlinked, diags
	}

	// A reported primary only proves this index is a replica of something. Whether
	// it is a *virtual* replica is recorded in the primary's replicas list, so that
	// has to be read too - see classifyReplicaLinkage.
	linkage, err := classifyReplicaLinkage(ctx, r.client, primaryIndexName, indexName)
	if err != nil {
		diags.AddError(algoliaerr.Object(indexKind, primaryIndexName).Message(algoliaerr.Read, err))
		return virtualIndexAbsent, diags
	}

	switch linkage {
	case replicaLinkageVirtual:
		return virtualIndexFound, diags
	case replicaLinkageStandard:
		return virtualIndexStandardReplica, diags
	default:
		// The primary is gone, or exists and no longer lists this index at all.
		// Either way the replica link is not there.
		return virtualIndexUnlinked, diags
	}
}

// standardReplicaDetail explains an index that Algolia keeps as a standard
// replica while Terraform manages it as a virtual one.
func standardReplicaDetail(indexName, primaryIndexName string) string {
	return "Index " + indexName + " is a standard replica of primary index " + primaryIndexName +
		", not a virtual one: the primary lists it as " + indexName + " rather than " +
		virtualReplicaName(indexName) + ". Algolia copies a primary index's records into a standard " +
		"replica, so this index now holds its own copy of them instead of being a view over them.\n\n" +
		"Restoring it as a virtual replica is this resource's own doing: it relinks its entry on " +
		"every write, and " + virtualReplicaName(indexName) + " cannot be declared in the primary " +
		"index's advanced.replicas, which owns standard entries only. To keep this index as a " +
		"standard replica, manage it with algolia_index rather than algolia_virtual_index."
}

// unlinkedVirtualIndexDetail explains an index that exists but has stopped
// being a virtual replica, in terms of the thing that usually causes it.
func unlinkedVirtualIndexDetail(indexName, primaryIndexName string) string {
	detail := "Index " + indexName + " exists but reports no primary index, so Algolia no longer " +
		"treats it as a virtual replica."
	if primaryIndexName != "" {
		detail += " Its replica link on primary index " + primaryIndexName + " appears to have been removed."
	}

	return detail + " The usual cause is a write to the primary index's replicas list from outside " +
		"Terraform that omitted " + virtualReplicaName(indexName) + ". A write through " +
		"algolia_index's advanced.replicas preserves virtual entries, so it is not this."
}

// readIndexModel populates model from the Algolia API, reporting whether the
// index exists. The flag is only meaningful when the returned diagnostics carry
// no error.
func (r *virtualIndexResource) readIndexModel(ctx context.Context, model *IndexResourceModel) (bool, diag.Diagnostics) {
	indexName := model.Name.ValueString()

	settingsResp, err := r.client.GetSettings(r.client.NewApiGetSettingsRequest(indexName), search.WithContext(ctx))
	if err != nil {
		var diags diag.Diagnostics
		if algoliaerr.IsNotFound(err) {
			return false, diags
		}
		diags.AddError(algoliaerr.Object(indexKind, indexName).Message(algoliaerr.Read, err))
		return false, diags
	}

	diags := flattenSettingsResponse(ctx, settingsResp, model)
	if diags.HasError() {
		return false, diags
	}

	applyIndexMetadata(ctx, r.client, model)

	return true, diags
}

func restoreVirtualNullBlocks(model *VirtualIndexResourceModel, blocks nullBlocks) {
	indexModel := virtualToIndexModel(*model)
	restoreNullBlocks(&indexModel, blocks)
	virtualFromIndexModel(indexModel, model)
}
