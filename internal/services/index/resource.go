package index

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"
	"github.com/algolia/terraform-provider-algolia/internal/algoliawait"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &indexResource{}
	_ resource.ResourceWithImportState = &indexResource{}
)

type indexResource struct {
	client *search.APIClient
}

// indexKind names this resource inside a diagnostic sentence. An index is
// addressed by name alone, so it needs no parent qualifier.
const indexKind = "index"

func NewResource() resource.Resource {
	return &indexResource{}
}

func (r *indexResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_index"
}

func (r *indexResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = indexResourceSchema()
}

func (r *indexResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *indexResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IndexResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := plan.Name.ValueString()
	tflog.Debug(ctx, "Creating index", map[string]interface{}{"name": indexName})

	// Remember which blocks were null in the plan so we can preserve them after read.
	nullBlocks := captureNullBlocks(&plan)

	settings, diags := expandIndexSettings(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	declaredReplicas := replicasDeclared(ctx, req.Config, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !declaredReplicas {
		settings.Replicas = nil
	}

	// Hold the primary's replica lock across the merge below and the write that
	// follows: both read the current list, and an algolia_virtual_index linking
	// itself in between would otherwise be dropped by this write.
	if settings.Replicas != nil {
		defer lockPrimaryReplicas(indexName)()

		merged, mergeDiags := mergeStandardReplicas(ctx, r.client, indexName, settings.Replicas)
		resp.Diagnostics.Append(mergeDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		settings.Replicas = merged
	}

	setResp, err := r.client.SetSettings(r.client.NewApiSetSettingsRequest(indexName, settings), search.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(algoliaerr.Object(indexKind, indexName).Message(algoliaerr.Create, err))
		return
	}

	// The index now exists in Algolia: SetSettings is what creates it (see
	// "settings-as-index" in AGENTS.md). Persist the identifying attributes
	// before waiting on the task or reading the index back, so a failure in
	// either step lands as a diagnostic on a resource Terraform knows about,
	// instead of orphaning an index that exists remotely but not in state --
	// nothing would ever adopt it, because the next apply would call Create
	// again.
	//
	// Only wholly-known values may be written here, since Terraform rejects an
	// apply result containing unknowns. Every settings block attribute and every
	// metadata attribute is Computed and is only resolved by the read-back
	// below, so they cannot be carried over from the plan; the early state
	// therefore holds the index name plus the deletion_protection guard and
	// leaves the rest null. That is deliberate: if the wait or read-back does
	// fail, the next plan sees a diff for every configured setting and
	// re-applies it, which is exactly the reconciliation an orphan never gets.
	resp.Diagnostics.Append(resp.State.Set(ctx, newIndexIdentityState(indexName, plan.DeletionProtection))...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err = waitForSettingsWrite(ctx, r.client, indexName, settings, setResp.TaskID); err != nil {
		resp.Diagnostics.AddError("Error waiting for index creation", "Could not wait for task: "+err.Error())
		return
	}

	// Save a snapshot of the plan before reading back, so we can preserve
	// values the API accepts on write but doesn't return on read.
	planSnapshot := plan

	found, diags := r.readIndex(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Error reading index", "Index "+indexName+" could not be found immediately after it was created.")
		return
	}

	resp.Diagnostics.Append(preservePlannedValues(ctx, &planSnapshot, &plan)...)
	if !declaredReplicas {
		resp.Diagnostics.Append(preserveUndeclaredReplicas(planSnapshot.Advanced, &plan.Advanced)...)
	}
	restoreNullBlocks(&plan, nullBlocks)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *indexResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state IndexResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Remember which blocks were null in state so we preserve them after read.
	nullBlocks := captureNullBlocks(&state)

	// Save a snapshot of the state before reading back, so we can preserve
	// values the API accepts on write but doesn't return on read.
	stateSnapshot := state

	found, diags := r.readIndex(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		// Deleted outside Terraform. Removing it from state lets the next plan
		// recreate it; leaving a diagnostic here would instead block plan, apply
		// and destroy alike until the operator ran `terraform state rm`.
		tflog.Warn(ctx, "Index not found; removing from state", map[string]interface{}{"name": state.Name.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(preservePlannedValues(ctx, &stateSnapshot, &state)...)
	restoreNullBlocks(&state, nullBlocks)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *indexResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan IndexResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := plan.Name.ValueString()
	tflog.Debug(ctx, "Updating index", map[string]interface{}{"name": indexName})

	nullBlocks := captureNullBlocks(&plan)

	settings, diags := expandIndexSettings(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	declaredReplicas := replicasDeclared(ctx, req.Config, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !declaredReplicas {
		settings.Replicas = nil
	}

	// See Create: the merge and the write share the primary's replica lock.
	if settings.Replicas != nil {
		defer lockPrimaryReplicas(indexName)()

		merged, mergeDiags := mergeStandardReplicas(ctx, r.client, indexName, settings.Replicas)
		resp.Diagnostics.Append(mergeDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		settings.Replicas = merged
	}

	setResp, err := r.client.SetSettings(r.client.NewApiSetSettingsRequest(indexName, settings), search.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(algoliaerr.Object(indexKind, indexName).Message(algoliaerr.Update, err))
		return
	}

	// No early state write here, unlike Create: Update runs against a resource
	// that is already in state, so a failing wait or read-back cannot orphan
	// anything. Writing the plan early would be actively harmful -- it would
	// record the desired settings as achieved without ever confirming them, and
	// the next plan would then show no diff. Leaving the prior state in place
	// means the next apply calls SetSettings again, which is idempotent.
	if err = waitForSettingsWrite(ctx, r.client, indexName, settings, setResp.TaskID); err != nil {
		resp.Diagnostics.AddError("Error waiting for index update", "Could not wait for task: "+err.Error())
		return
	}

	planSnapshot := plan

	found, diags := r.readIndex(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Error reading index", "Index "+indexName+" could not be found immediately after it was updated.")
		return
	}

	resp.Diagnostics.Append(preservePlannedValues(ctx, &planSnapshot, &plan)...)
	if !declaredReplicas {
		resp.Diagnostics.Append(preserveUndeclaredReplicas(planSnapshot.Advanced, &plan.Advanced)...)
	}
	restoreNullBlocks(&plan, nullBlocks)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *indexResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state IndexResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := state.Name.ValueString()

	// Fail safe on an absent value. The schema defaults deletion_protection to true,
	// so null should not occur after a normal apply; when it does (legacy state, or a
	// state written before import seeded the default) treating it as "unprotected"
	// would delete a production index. Require an explicit false to proceed.
	if state.DeletionProtection.IsNull() || state.DeletionProtection.ValueBool() {
		resp.Diagnostics.AddError(
			"Deletion Protection Enabled",
			fmt.Sprintf("Cannot delete index %q because deletion_protection is enabled. "+
				"Set deletion_protection = false and apply before destroying.", indexName),
		)
		return
	}

	tflog.Debug(ctx, "Deleting index", map[string]interface{}{"name": indexName})

	if err := deleteIndexWithUnlink(ctx, r.client, indexName); err != nil {
		resp.Diagnostics.AddError(algoliaerr.Object(indexKind, indexName).Message(algoliaerr.Delete, err))
		return
	}

	// A published delete task is not proof the index went away; see
	// confirmIndexDeleted. Erroring here keeps the resource in state, which is what
	// makes the next destroy retry it.
	if err := confirmIndexDeleted(ctx, r.client, indexName); err != nil {
		resp.Diagnostics.AddError("Index still exists after deletion", deleteNotConfirmedDetail(indexName, err))
		return
	}
}

func (r *indexResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Importing must populate the settings blocks here rather than relying on the
	// subsequent Read. Read() preserves whichever blocks were null beforehand (see
	// captureNullBlocks/restoreNullBlocks), which during a passthrough import is all
	// of them -- so the freshly read settings would be discarded and the imported
	// state would contain nothing but the index name.
	model := IndexResourceModel{
		Name: types.StringValue(req.ID),
		// deletion_protection is a provider-side guard with no Algolia API
		// representation, so it cannot be read back. Seed the schema default (true)
		// so an imported index stays protected until the operator explicitly opts
		// out. Leaving it null makes the Delete guard evaluate to false, which
		// silently destroys an index the configuration marked as protected.
		DeletionProtection: types.BoolValue(true),
	}

	found, diags := r.readIndex(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		// Importing is the one path that must fail on a 404: there is nothing to
		// bring under management, and succeeding would write state for an index
		// that does not exist. This is the opposite of Read's behaviour, which is
		// why readIndex reports absence rather than deciding for both callers.
		resp.Diagnostics.AddError("Error reading index", "Could not read index "+req.ID+": the index does not exist.")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

// readIndex populates model from the Algolia API.
//
// The first return value reports whether the index exists. A 404 is not an error
// here because the two callers need opposite behaviour: Read has to drop the
// resource from state so the next plan recreates it, while ImportState has to
// fail, since importing an index that does not exist cannot produce usable state.
// Turning the 404 into a diagnostic in here would force both onto the same
// behaviour, which is what made an out-of-band `algolia indices delete` wedge the
// resource until the operator ran `terraform state rm`. The flag is only
// meaningful when the returned diagnostics carry no error.
func (r *indexResource) readIndex(ctx context.Context, model *IndexResourceModel) (bool, diag.Diagnostics) {
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

// preservePlannedValues copies non-null planned block attribute values into the
// read-back model when the API returned null for those attributes. This handles
// fields the Algolia API accepts on write but does not include in GetSettings
// responses (e.g., relevancyStrictness, allowCompressionOfIntegerArray).
func preservePlannedValues(ctx context.Context, plan, state *IndexResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	blockPairs := []struct {
		planObj   types.Object
		stateObj  *types.Object
		attrTypes map[string]attr.Type
	}{
		{plan.Ranking, &state.Ranking, rankingAttrTypes},
		{plan.Performance, &state.Performance, performanceAttrTypes},
		{plan.Advanced, &state.Advanced, advancedAttrTypes},
		{plan.Faceting, &state.Faceting, facetingAttrTypes},
		{plan.Highlighting, &state.Highlighting, highlightingAttrTypes},
		{plan.Pagination, &state.Pagination, paginationAttrTypes},
		{plan.Typos, &state.Typos, typosAttrTypes},
		{plan.Languages, &state.Languages, languagesAttrTypes},
		{plan.QueryStrategy, &state.QueryStrategy, queryStrategyAttrTypes},
		{plan.Attributes, &state.Attributes, attributesAttrTypes},
	}

	for _, bp := range blockPairs {
		if bp.planObj.IsNull() || bp.planObj.IsUnknown() {
			continue
		}
		if bp.stateObj.IsNull() || bp.stateObj.IsUnknown() {
			continue
		}

		planAttrs := bp.planObj.Attributes()
		stateAttrs := bp.stateObj.Attributes()
		changed := false

		for k, planVal := range planAttrs {
			stateVal, ok := stateAttrs[k]
			if !ok {
				continue
			}
			// If planned value was set but API returned null, keep the planned value.
			if !planVal.IsNull() && !planVal.IsUnknown() && stateVal.IsNull() {
				stateAttrs[k] = planVal
				changed = true
				continue
			}
			// For JSON-encoded string fields, preserve the planned value if
			// the API returned semantically equivalent JSON (e.g. different
			// array element ordering).
			planStr, planIsStr := planVal.(types.String)
			stateStr, stateIsStr := stateVal.(types.String)
			if planIsStr && stateIsStr && !planStr.IsNull() && !stateStr.IsNull() &&
				!planStr.IsUnknown() && !stateStr.IsUnknown() &&
				planStr.ValueString() != stateStr.ValueString() &&
				jsonSemanticallyEqual(planStr.ValueString(), stateStr.ValueString()) {
				stateAttrs[k] = planVal
				changed = true
			}
		}

		if changed {
			newObj, d := types.ObjectValue(bp.attrTypes, stateAttrs)
			diags.Append(d...)
			if !diags.HasError() {
				*bp.stateObj = newObj
			}
		}
	}

	return diags
}

// nullBlocks tracks which blocks were null before a read operation.
type nullBlocks struct {
	attributes, ranking, faceting, highlighting, pagination bool
	typos, languages, queryStrategy, performance, advanced  bool
}

// captureNullBlocks records which blocks are currently null on the model.
func captureNullBlocks(m *IndexResourceModel) nullBlocks {
	return nullBlocks{
		attributes:    m.Attributes.IsNull(),
		ranking:       m.Ranking.IsNull(),
		faceting:      m.Faceting.IsNull(),
		highlighting:  m.Highlighting.IsNull(),
		pagination:    m.Pagination.IsNull(),
		typos:         m.Typos.IsNull(),
		languages:     m.Languages.IsNull(),
		queryStrategy: m.QueryStrategy.IsNull(),
		performance:   m.Performance.IsNull(),
		advanced:      m.Advanced.IsNull(),
	}
}

// restoreNullBlocks sets blocks back to null if they were null before the read.
func restoreNullBlocks(m *IndexResourceModel, nb nullBlocks) {
	if nb.attributes {
		m.Attributes = types.ObjectNull(attributesAttrTypes)
	}
	if nb.ranking {
		m.Ranking = types.ObjectNull(rankingAttrTypes)
	}
	if nb.faceting {
		m.Faceting = types.ObjectNull(facetingAttrTypes)
	}
	if nb.highlighting {
		m.Highlighting = types.ObjectNull(highlightingAttrTypes)
	}
	if nb.pagination {
		m.Pagination = types.ObjectNull(paginationAttrTypes)
	}
	if nb.typos {
		m.Typos = types.ObjectNull(typosAttrTypes)
	}
	if nb.languages {
		m.Languages = types.ObjectNull(languagesAttrTypes)
	}
	if nb.queryStrategy {
		m.QueryStrategy = types.ObjectNull(queryStrategyAttrTypes)
	}
	if nb.performance {
		m.Performance = types.ObjectNull(performanceAttrTypes)
	}
	if nb.advanced {
		m.Advanced = types.ObjectNull(advancedAttrTypes)
	}
}

// waitForIndexTask polls GetTask until the task reaches "published" status.
// This replaces the SDK's built-in WaitForTask whose retry-count options were
// not being applied, causing it to time out on large indexes.
func waitForIndexTask(ctx context.Context, client *search.APIClient, indexName string, taskID int64) error {
	return algoliawait.Until(ctx, fmt.Sprintf("task %d on index %q", taskID, indexName), func(ctx context.Context) (bool, error) {
		resp, err := client.GetTask(client.NewApiGetTaskRequest(indexName, taskID), search.WithContext(ctx))
		if err != nil {
			return false, err
		}

		return resp.Status == search.TASK_STATUS_PUBLISHED, nil
	})
}

// newIndexIdentityState returns the minimal state to persist immediately after a
// successful SetSettings, before the task wait and read-back that can still fail.
// It carries only wholly-known values, since Terraform rejects an apply result
// containing unknowns: the index name, and the deletion_protection guard so a
// later destroy still honours it. Everything else is Computed and resolved only
// by the read-back, so it is left null deliberately -- a subsequent plan then
// sees a diff for each configured setting and re-applies it, which is the
// reconciliation an orphaned index would never get.
func newIndexIdentityState(indexName string, deletionProtection types.Bool) IndexResourceModel {
	model := IndexResourceModel{
		Name:               types.StringValue(indexName),
		DeletionProtection: deletionProtection,
	}

	// The settings blocks must be *typed* nulls. A zero-value types.Object carries
	// no attribute-type information, and the framework rejects it with a "Value
	// Conversion Error" rather than treating it as null.
	restoreNullBlocks(&model, nullBlocks{
		attributes: true, ranking: true, faceting: true, highlighting: true, pagination: true,
		typos: true, languages: true, queryStrategy: true, performance: true, advanced: true,
	})

	return model
}

// jsonSemanticallyEqual returns true if two strings are both valid JSON and
// represent the same data structure, ignoring key order, whitespace, and
// array element order (the Algolia API may return array elements in a
// different order than what was sent).
func jsonSemanticallyEqual(a, b string) bool {
	var va, vb any
	if err := json.Unmarshal([]byte(a), &va); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &vb); err != nil {
		return false
	}
	return reflect.DeepEqual(normalizeJSON(va), normalizeJSON(vb))
}

// normalizeJSON recursively sorts arrays of strings/numbers and normalizes
// maps so that JSON comparison is order-independent.
func normalizeJSON(v any) any {
	switch val := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(val))
		for k, v := range val {
			m[k] = normalizeJSON(v)
		}
		return m
	case []any:
		normalized := make([]any, len(val))
		for i, item := range val {
			normalized[i] = normalizeJSON(item)
		}
		// Sort the slice if all elements are strings.
		allStrings := true
		for _, item := range normalized {
			if _, ok := item.(string); !ok {
				allStrings = false
				break
			}
		}
		if allStrings {
			sort.Slice(normalized, func(i, j int) bool {
				return normalized[i].(string) < normalized[j].(string)
			})
		}
		return normalized
	default:
		return v
	}
}
