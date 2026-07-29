package rule

import (
	"context"
	"fmt"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"
	"github.com/algolia/terraform-provider-algolia/internal/algoliawait"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &ruleResource{}
	_ resource.ResourceWithConfigure   = &ruleResource{}
	_ resource.ResourceWithImportState = &ruleResource{}
)

type ruleResource struct {
	client *search.APIClient
}

// ruleKind names this resource inside a diagnostic sentence.
const ruleKind = "rule"

// ruleSubject identifies the rule a diagnostic is about. Rules are scoped to an
// index, so both parts are needed to name one unambiguously.
func ruleSubject(indexName, objectID string) algoliaerr.Subject {
	return algoliaerr.Object(ruleKind, objectID).In("index", indexName)
}

func NewResource() resource.Resource {
	return &ruleResource{}
}

func (r *ruleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule"
}

func (r *ruleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = ruleResourceSchema()
}

func (r *ruleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ruleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := plan.IndexName.ValueString()
	objectID := plan.ObjectID.ValueString()
	tflog.Debug(ctx, "Creating rule", map[string]any{"index_name": indexName, "object_id": objectID})

	body, diags := ruleRequestBodyFromModel(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	taskID, err := saveRuleRaw(ctx, r.client, indexName, objectID, body)
	if err != nil {
		resp.Diagnostics.AddError(ruleSubject(indexName, objectID).Message(algoliaerr.Create, err))
		return
	}

	if err := waitForRuleTask(ctx, r.client, indexName, taskID); err != nil {
		resp.Diagnostics.AddError(algoliaerr.WaitMessage(ruleKind, algoliaerr.Create, err))
		return
	}

	apiResp, rawParams, err := getRuleRaw(ctx, r.client, indexName, objectID)
	if err != nil {
		resp.Diagnostics.AddError(ruleSubject(indexName, objectID).Message(algoliaerr.Read, err))
		return
	}

	resp.Diagnostics.Append(hydrateRuleModel(indexName, apiResp, rawParams, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ruleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := state.IndexName.ValueString()
	objectID := state.ObjectID.ValueString()
	tflog.Debug(ctx, "Reading rule", map[string]any{"index_name": indexName, "object_id": objectID})

	apiResp, rawParams, err := getRuleRaw(ctx, r.client, indexName, objectID)
	if err != nil {
		if algoliaerr.IsNotFound(err) {
			tflog.Warn(ctx, "Rule not found; removing from state", map[string]any{"index_name": indexName, "object_id": objectID})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(ruleSubject(indexName, objectID).Message(algoliaerr.Read, err))
		return
	}

	resp.Diagnostics.Append(hydrateRuleModel(indexName, apiResp, rawParams, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ruleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := plan.IndexName.ValueString()
	objectID := plan.ObjectID.ValueString()
	tflog.Debug(ctx, "Updating rule", map[string]any{"index_name": indexName, "object_id": objectID})

	body, diags := ruleRequestBodyFromModel(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	taskID, err := saveRuleRaw(ctx, r.client, indexName, objectID, body)
	if err != nil {
		resp.Diagnostics.AddError(ruleSubject(indexName, objectID).Message(algoliaerr.Update, err))
		return
	}

	if err := waitForRuleTask(ctx, r.client, indexName, taskID); err != nil {
		resp.Diagnostics.AddError(algoliaerr.WaitMessage(ruleKind, algoliaerr.Update, err))
		return
	}

	apiResp, rawParams, err := getRuleRaw(ctx, r.client, indexName, objectID)
	if err != nil {
		resp.Diagnostics.AddError(ruleSubject(indexName, objectID).Message(algoliaerr.Read, err))
		return
	}

	resp.Diagnostics.Append(hydrateRuleModel(indexName, apiResp, rawParams, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ruleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := state.IndexName.ValueString()
	objectID := state.ObjectID.ValueString()

	deleteResp, err := r.client.DeleteRule(r.client.NewApiDeleteRuleRequest(indexName, objectID), search.WithContext(ctx))
	if err != nil {
		if algoliaerr.IsNotFound(err) {
			return
		}

		resp.Diagnostics.AddError(ruleSubject(indexName, objectID).Message(algoliaerr.Delete, err))
		return
	}

	if err := waitForRuleTask(ctx, r.client, indexName, deleteResp.TaskID); err != nil {
		if algoliaerr.IsNotFound(err) {
			return
		}

		resp.Diagnostics.AddError(algoliaerr.WaitMessage(ruleKind, algoliaerr.Delete, err))
	}
}

func (r *ruleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	indexName, objectID, err := parseRuleImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	apiResp, rawParams, err := getRuleRaw(ctx, r.client, indexName, objectID)
	if err != nil {
		resp.Diagnostics.AddError(algoliaerr.Object(ruleKind, req.ID).Message(algoliaerr.Import, err))
		return
	}

	var state RuleResourceModel
	resp.Diagnostics.Append(hydrateRuleModel(indexName, apiResp, rawParams, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func waitForRuleTask(ctx context.Context, client *search.APIClient, indexName string, taskID int64) error {
	return algoliawait.Until(ctx, fmt.Sprintf("task %d on index %q", taskID, indexName), func(ctx context.Context) (bool, error) {
		resp, err := client.GetTask(client.NewApiGetTaskRequest(indexName, taskID), search.WithContext(ctx))
		if err != nil {
			return false, err
		}

		return resp.Status == search.TASK_STATUS_PUBLISHED, nil
	})
}
