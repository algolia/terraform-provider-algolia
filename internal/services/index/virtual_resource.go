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
	// virtualIndexFound: the index exists and reports a primary index.
	virtualIndexFound
	// virtualIndexUnlinked: the index exists but reports no primary index, so
	// Algolia no longer considers it a replica of anything.
	virtualIndexUnlinked
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
	if model.PrimaryIndexName.IsNull() || model.PrimaryIndexName.ValueString() == "" {
		return virtualIndexUnlinked, diags
	}

	return virtualIndexFound, diags
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
