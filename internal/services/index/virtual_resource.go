package index

import (
	"context"
	"fmt"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &virtualIndexResource{}
	_ resource.ResourceWithConfigure   = &virtualIndexResource{}
	_ resource.ResourceWithImportState = &virtualIndexResource{}
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

	if err = waitForIndexTask(ctx, r.client, indexName, setResp.TaskID); err != nil {
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
		// guards those records, and the warning repeats on every refresh until the
		// primary's replicas list is corrected.
		// The primary read back from the API, not the prior state's: reaching this
		// case means Algolia reported one, and that is the index to name.
		resp.Diagnostics.AddWarning(
			"Virtual index is a standard replica",
			standardReplicaDetail(state.Name.ValueString(), state.PrimaryIndexName.ValueString()),
		)
	}

	indexRead := virtualToIndexModel(state)
	resp.Diagnostics.Append(preservePlannedValues(ctx, &stateSnapshot, &indexRead)...)
	virtualFromIndexModel(indexRead, &state)
	restoreVirtualNullBlocks(&state, nullBlocks)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
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

	if err = waitForIndexTask(ctx, r.client, indexName, setResp.TaskID); err != nil {
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
	// Fail safe on an absent value: treating null as "unprotected" would delete a
	// production index. Require an explicit false to proceed.
	if state.DeletionProtection.IsNull() || state.DeletionProtection.ValueBool() {
		resp.Diagnostics.AddError(
			"Deletion Protection Enabled",
			fmt.Sprintf("Cannot delete virtual index %q because deletion_protection is enabled. Set deletion_protection = false and apply before destroying.", indexName),
		)
		return
	}

	primaryIndexName := state.PrimaryIndexName.ValueString()
	if primaryIndexName != "" {
		if err := removeVirtualReplicaLink(ctx, r.client, primaryIndexName, indexName); err != nil {
			resp.Diagnostics.AddError("Error unlinking virtual replica", "Could not unlink virtual replica "+indexName+" from primary index "+primaryIndexName+": "+err.Error())
			return
		}
	}

	delResp, err := r.client.DeleteIndex(r.client.NewApiDeleteIndexRequest(indexName), search.WithContext(ctx))
	if err != nil {
		if algoliaerr.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError(algoliaerr.Object(virtualIndexKind, indexName).Message(algoliaerr.Delete, err))
		return
	}

	if err = waitForIndexTask(ctx, r.client, indexName, delResp.TaskID); err != nil {
		resp.Diagnostics.AddError("Error waiting for virtual index deletion", "Could not wait for task: "+err.Error())
		return
	}

	if err = confirmIndexDeleted(ctx, r.client, indexName); err != nil {
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
		"To restore it as a virtual replica, list it as " + virtualReplicaName(indexName) +
		" in the primary index's replicas. To keep it as a standard replica, manage it with " +
		"algolia_index rather than algolia_virtual_index."
}

// unlinkedVirtualIndexDetail explains an index that exists but has stopped
// being a virtual replica, in terms of the thing that usually causes it.
func unlinkedVirtualIndexDetail(indexName, primaryIndexName string) string {
	detail := "Index " + indexName + " exists but reports no primary index, so Algolia no longer " +
		"treats it as a virtual replica."
	if primaryIndexName != "" {
		detail += " Its replica link on primary index " + primaryIndexName + " appears to have been removed."
	}

	return detail + " The usual cause is a wholesale write of the primary index's replicas list - " +
		"from algolia_index's advanced.replicas, or from outside Terraform - that omitted " +
		virtualReplicaName(indexName) + "."
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
