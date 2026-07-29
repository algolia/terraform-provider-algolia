package recommend

import (
	"context"
	"fmt"

	recommendapi "github.com/algolia/algoliasearch-client-go/v4/algolia/recommend"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"
	"github.com/algolia/terraform-provider-algolia/internal/algoliawait"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &recommendRuleResource{}
	_ resource.ResourceWithConfigure   = &recommendRuleResource{}
	_ resource.ResourceWithImportState = &recommendRuleResource{}
)

// recommendRuleResource manages an algolia_recommend_rule resource. It
// embeds base (see client.go) for Configure-time wiring and an on-demand
// Recommend client - the Recommend API is not region-routed, so base here
// only needs appID/apiKey (unlike, say, abtest.base, which also carries an
// analyticsRegion).
type recommendRuleResource struct {
	base
}

// recommendRuleKind names this resource inside a diagnostic sentence.
const recommendRuleKind = "Recommend rule"

// recommendRuleSubject identifies the Recommend rule a diagnostic is about.
// Recommend rules are scoped to an index, so both parts are needed to name one
// unambiguously.
func recommendRuleSubject(indexName, objectID string) algoliaerr.Subject {
	return algoliaerr.Object(recommendRuleKind, objectID).In("index", indexName)
}

// NewResource returns the algolia_recommend_rule resource.
func NewResource() resource.Resource {
	return &recommendRuleResource{}
}

func (r *recommendRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_recommend_rule"
}

func (r *recommendRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = recommendRuleResourceSchema()
}

func (r *recommendRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(r.configure(req.ProviderData)...)
}

func (r *recommendRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RecommendRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := plan.IndexName.ValueString()
	recommendModel := recommendapi.RecommendModels(plan.Model.ValueString())

	objectID := plan.ObjectID.ValueString()
	if plan.ObjectID.IsNull() || plan.ObjectID.IsUnknown() || objectID == "" {
		generated, err := generateObjectID()
		if err != nil {
			resp.Diagnostics.AddError("Error generating object_id", "Could not generate a random object_id: "+err.Error())
			return
		}
		objectID = generated
	}
	plan.ObjectID = types.StringValue(objectID)

	tflog.Debug(ctx, "Creating Recommend rule", map[string]any{
		"index_name": indexName,
		"model":      string(recommendModel),
		"object_id":  objectID,
	})

	rule, expandDiags := expandRecommendRule(objectID, &plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	batchReq := client.NewApiBatchRecommendRulesRequest(indexName, recommendModel).
		WithRecommendRule([]recommendapi.RecommendRule{*rule})
	batchResp, err := client.BatchRecommendRules(batchReq, recommendapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(recommendRuleSubject(indexName, objectID).Message(algoliaerr.Create, err))
		return
	}

	// The rule now exists in Algolia. Persist the identifying attributes
	// immediately, before waiting on the task or reading the rule back, so
	// that a failure in either of those steps surfaces as an error on a
	// resource Terraform knows about, instead of orphaning a rule that exists
	// remotely but not in state (which no subsequent apply would ever adopt).
	// Every remaining attribute is already known from the plan, so this state
	// is consistent with what Terraform planned.
	plan.ID = types.StringValue(recommendRuleResourceID(indexName, string(recommendModel), objectID))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := waitForRecommendRuleTask(ctx, client, indexName, recommendModel, batchResp.TaskID); err != nil {
		resp.Diagnostics.AddError(algoliaerr.WaitMessage(recommendRuleKind, algoliaerr.Create, err))
		return
	}

	apiResp, err := getRecommendRule(client, client.NewApiGetRecommendRuleRequest(indexName, recommendModel, objectID), recommendapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(recommendRuleSubject(indexName, objectID).Message(algoliaerr.Read, err))
		return
	}

	resp.Diagnostics.Append(flattenRecommendRule(indexName, string(recommendModel), apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *recommendRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RecommendRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := state.IndexName.ValueString()
	recommendModel := recommendapi.RecommendModels(state.Model.ValueString())
	objectID := state.ObjectID.ValueString()

	apiResp, err := getRecommendRule(client, client.NewApiGetRecommendRuleRequest(indexName, recommendModel, objectID), recommendapi.WithContext(ctx))
	if err != nil {
		if algoliaerr.IsNotFound(err) {
			tflog.Warn(ctx, "Recommend rule not found; removing from state", map[string]any{
				"index_name": indexName,
				"model":      string(recommendModel),
				"object_id":  objectID,
			})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(recommendRuleSubject(indexName, objectID).Message(algoliaerr.Read, err))
		return
	}

	resp.Diagnostics.Append(flattenRecommendRule(indexName, string(recommendModel), apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *recommendRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RecommendRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := plan.IndexName.ValueString()
	recommendModel := recommendapi.RecommendModels(plan.Model.ValueString())
	objectID := plan.ObjectID.ValueString()

	tflog.Debug(ctx, "Updating Recommend rule", map[string]any{
		"index_name": indexName,
		"model":      string(recommendModel),
		"object_id":  objectID,
	})

	rule, expandDiags := expandRecommendRule(objectID, &plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	batchReq := client.NewApiBatchRecommendRulesRequest(indexName, recommendModel).
		WithRecommendRule([]recommendapi.RecommendRule{*rule})
	batchResp, err := client.BatchRecommendRules(batchReq, recommendapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(recommendRuleSubject(indexName, objectID).Message(algoliaerr.Update, err))
		return
	}

	if err := waitForRecommendRuleTask(ctx, client, indexName, recommendModel, batchResp.TaskID); err != nil {
		resp.Diagnostics.AddError(algoliaerr.WaitMessage(recommendRuleKind, algoliaerr.Update, err))
		return
	}

	apiResp, err := getRecommendRule(client, client.NewApiGetRecommendRuleRequest(indexName, recommendModel, objectID), recommendapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(recommendRuleSubject(indexName, objectID).Message(algoliaerr.Read, err))
		return
	}

	resp.Diagnostics.Append(flattenRecommendRule(indexName, string(recommendModel), apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *recommendRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RecommendRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := state.IndexName.ValueString()
	recommendModel := recommendapi.RecommendModels(state.Model.ValueString())
	objectID := state.ObjectID.ValueString()

	tflog.Debug(ctx, "Deleting Recommend rule", map[string]any{
		"index_name": indexName,
		"model":      string(recommendModel),
		"object_id":  objectID,
	})

	deleteResp, err := client.DeleteRecommendRule(client.NewApiDeleteRecommendRuleRequest(indexName, recommendModel, objectID), recommendapi.WithContext(ctx))
	if err != nil {
		if algoliaerr.IsNotFound(err) {
			return
		}

		resp.Diagnostics.AddError(recommendRuleSubject(indexName, objectID).Message(algoliaerr.Delete, err))
		return
	}

	if err := waitForRecommendRuleTask(ctx, client, indexName, recommendModel, deleteResp.TaskID); err != nil {
		if algoliaerr.IsNotFound(err) {
			return
		}

		resp.Diagnostics.AddError(algoliaerr.WaitMessage(recommendRuleKind, algoliaerr.Delete, err))
	}
}

func (r *recommendRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	indexName, modelName, objectID, err := parseRecommendRuleImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := getRecommendRule(client, client.NewApiGetRecommendRuleRequest(indexName, recommendapi.RecommendModels(modelName), objectID), recommendapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(algoliaerr.Object(recommendRuleKind, req.ID).Message(algoliaerr.Import, err))
		return
	}

	var state RecommendRuleResourceModel
	resp.Diagnostics.Append(flattenRecommendRule(indexName, modelName, apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// waitForRecommendRuleTask polls GetRecommendStatus until the given task
// completes, mirroring waitForRuleTask/waitForSynonymTask in the
// rule/synonym packages (which poll search.GetTask instead - Recommend has
// its own, differently-routed task-status endpoint per model/index).
func waitForRecommendRuleTask(ctx context.Context, client *recommendapi.APIClient, indexName string, model recommendapi.RecommendModels, taskID int64) error {
	return algoliawait.Until(ctx, fmt.Sprintf("task %d on index %q", taskID, indexName), func(ctx context.Context) (bool, error) {
		resp, err := client.GetRecommendStatus(client.NewApiGetRecommendStatusRequest(indexName, model, taskID), recommendapi.WithContext(ctx))
		if err != nil {
			return false, err
		}

		return resp.Status == recommendapi.TASK_STATUS_PUBLISHED, nil
	})
}
