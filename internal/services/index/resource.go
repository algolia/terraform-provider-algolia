package index

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
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

	setResp, err := r.client.SetSettings(r.client.NewApiSetSettingsRequest(indexName, settings))
	if err != nil {
		resp.Diagnostics.AddError("Error creating index", "Could not create index "+indexName+": "+err.Error())
		return
	}

	_, err = r.client.WaitForTask(indexName, setResp.TaskID)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for index creation", "Could not wait for task: "+err.Error())
		return
	}

	// Save a snapshot of the plan before reading back, so we can preserve
	// values the API accepts on write but doesn't return on read.
	planSnapshot := plan

	diags = r.readIndex(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(preservePlannedValues(ctx, &planSnapshot, &plan)...)
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

	diags := r.readIndex(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
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

	setResp, err := r.client.SetSettings(r.client.NewApiSetSettingsRequest(indexName, settings))
	if err != nil {
		resp.Diagnostics.AddError("Error updating index", "Could not update index "+indexName+": "+err.Error())
		return
	}

	_, err = r.client.WaitForTask(indexName, setResp.TaskID)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for index update", "Could not wait for task: "+err.Error())
		return
	}

	planSnapshot := plan

	diags = r.readIndex(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(preservePlannedValues(ctx, &planSnapshot, &plan)...)
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

	if !state.DeletionProtection.IsNull() && state.DeletionProtection.ValueBool() {
		resp.Diagnostics.AddError(
			"Deletion Protection Enabled",
			fmt.Sprintf("Cannot delete index %q because deletion_protection is enabled. "+
				"Set deletion_protection = false and apply before destroying.", indexName),
		)
		return
	}

	tflog.Debug(ctx, "Deleting index", map[string]interface{}{"name": indexName})

	delResp, err := r.client.DeleteIndex(r.client.NewApiDeleteIndexRequest(indexName))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting index", "Could not delete index "+indexName+": "+err.Error())
		return
	}

	_, err = r.client.WaitForTask(indexName, delResp.TaskID)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for index deletion", "Could not wait for task: "+err.Error())
		return
	}
}

func (r *indexResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func (r *indexResource) readIndex(ctx context.Context, model *IndexResourceModel) diag.Diagnostics {
	indexName := model.Name.ValueString()

	settingsResp, err := r.client.GetSettings(r.client.NewApiGetSettingsRequest(indexName))
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Error reading index", "Could not read index "+indexName+": "+err.Error())
		return diags
	}

	diags := flattenSettingsResponse(ctx, settingsResp, model)
	if diags.HasError() {
		return diags
	}

	// Fetch index metadata (entries, data_size, created_at, updated_at) from ListIndices.
	listResp, err := r.client.ListIndices(r.client.NewApiListIndicesRequest())
	if err != nil {
		// Non-fatal: metadata is optional, log and continue.
		tflog.Warn(ctx, "Could not list indices for metadata", map[string]interface{}{"error": err.Error()})
		model.Entries = types.Int64Value(0)
		model.DataSize = types.Int64Value(0)
		model.CreatedAt = types.StringValue("")
		model.UpdatedAt = types.StringValue("")
		return diags
	}

	for _, idx := range listResp.Items {
		if idx.Name == indexName {
			model.Entries = types.Int64Value(int64(idx.Entries))
			model.DataSize = types.Int64Value(idx.DataSize)
			model.CreatedAt = types.StringValue(idx.CreatedAt)
			model.UpdatedAt = types.StringValue(idx.UpdatedAt)
			return diags
		}
	}

	// Index not found in list (possibly just created, not yet visible).
	model.Entries = types.Int64Value(0)
	model.DataSize = types.Int64Value(0)
	model.CreatedAt = types.StringValue("")
	model.UpdatedAt = types.StringValue("")

	return diags
}

// preservePlannedValues copies non-null planned block attribute values into the
// read-back model when the API returned null for those attributes. This handles
// fields the Algolia API accepts on write but does not include in GetSettings
// responses (e.g., relevancyStrictness, allowCompressionOfIntegerArray).
func preservePlannedValues(ctx context.Context, plan, state *IndexResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	blockPairs := []struct {
		planObj  types.Object
		stateObj *types.Object
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
	attributes, ranking, faceting, highlighting, pagination       bool
	typos, languages, queryStrategy, performance, advanced        bool
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
