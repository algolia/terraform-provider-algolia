package recommend

import (
	"context"
	"errors"
	"fmt"
	"time"

	recommendapi "github.com/algolia/algoliasearch-client-go/v4/algolia/recommend"
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
	batchResp, err := client.BatchRecommendRules(batchReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Recommend rule", "Could not create Recommend rule "+objectID+" on index "+indexName+": "+err.Error())
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

	if err := waitForRecommendRuleTask(client, indexName, recommendModel, batchResp.TaskID); err != nil {
		resp.Diagnostics.AddError("Error waiting for Recommend rule creation", "Could not confirm Recommend rule creation: "+err.Error())
		return
	}

	apiResp, err := getRecommendRule(client, client.NewApiGetRecommendRuleRequest(indexName, recommendModel, objectID))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Recommend rule", "Could not read Recommend rule "+objectID+" on index "+indexName+": "+err.Error())
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

	apiResp, err := getRecommendRule(client, client.NewApiGetRecommendRuleRequest(indexName, recommendModel, objectID))
	if err != nil {
		var apiErr *recommendapi.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			tflog.Warn(ctx, "Recommend rule not found; removing from state", map[string]any{
				"index_name": indexName,
				"model":      string(recommendModel),
				"object_id":  objectID,
			})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading Recommend rule", "Could not read Recommend rule "+objectID+" on index "+indexName+": "+err.Error())
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
	batchResp, err := client.BatchRecommendRules(batchReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Recommend rule", "Could not update Recommend rule "+objectID+" on index "+indexName+": "+err.Error())
		return
	}

	if err := waitForRecommendRuleTask(client, indexName, recommendModel, batchResp.TaskID); err != nil {
		resp.Diagnostics.AddError("Error waiting for Recommend rule update", "Could not confirm Recommend rule update: "+err.Error())
		return
	}

	apiResp, err := getRecommendRule(client, client.NewApiGetRecommendRuleRequest(indexName, recommendModel, objectID))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Recommend rule", "Could not read Recommend rule "+objectID+" on index "+indexName+": "+err.Error())
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

	deleteResp, err := client.DeleteRecommendRule(client.NewApiDeleteRecommendRuleRequest(indexName, recommendModel, objectID))
	if err != nil {
		var apiErr *recommendapi.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return
		}

		resp.Diagnostics.AddError("Error deleting Recommend rule", "Could not delete Recommend rule "+objectID+" on index "+indexName+": "+err.Error())
		return
	}

	if err := waitForRecommendRuleTask(client, indexName, recommendModel, deleteResp.TaskID); err != nil {
		var apiErr *recommendapi.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return
		}

		resp.Diagnostics.AddError("Error waiting for Recommend rule deletion", "Could not confirm Recommend rule deletion: "+err.Error())
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

	apiResp, err := getRecommendRule(client, client.NewApiGetRecommendRuleRequest(indexName, recommendapi.RecommendModels(modelName), objectID))
	if err != nil {
		resp.Diagnostics.AddError("Error importing Recommend rule", "Could not import Recommend rule "+req.ID+": "+err.Error())
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
func waitForRecommendRuleTask(client *recommendapi.APIClient, indexName string, model recommendapi.RecommendModels, taskID int64) error {
	deadline := time.Now().Add(30 * time.Minute)
	interval := 2 * time.Second
	for time.Now().Before(deadline) {
		resp, err := client.GetRecommendStatus(client.NewApiGetRecommendStatusRequest(indexName, model, taskID))
		if err != nil {
			return err
		}
		if resp.Status == recommendapi.TASK_STATUS_PUBLISHED {
			return nil
		}
		time.Sleep(interval)
		if interval < 10*time.Second {
			interval += time.Second
		}
	}
	return fmt.Errorf("task %d on index %q did not complete within 30 minutes", taskID, indexName)
}
