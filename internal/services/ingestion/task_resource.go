package ingestion

import (
	"context"
	"errors"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &taskResource{}
	_ resource.ResourceWithConfigure   = &taskResource{}
	_ resource.ResourceWithImportState = &taskResource{}
)

// taskResource manages an algolia_ingestion_task resource. It embeds base
// (see client.go) for Configure-time wiring and an on-demand region-routed
// Ingestion client.
type taskResource struct {
	base
}

// NewTaskResource returns the algolia_ingestion_task resource.
func NewTaskResource() resource.Resource {
	return &taskResource{}
}

func (r *taskResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingestion_task"
}

func (r *taskResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = taskResourceSchema()
}

func (r *taskResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(r.configure(req.ProviderData)...)
}

func (r *taskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TaskResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTask(ctx, client, &plan, resp)
}

// createTask is the Create body, split out so that a unit test can drive it
// against an httptest-backed client (see identity_state_test.go) - the client
// base.client() builds always talks to the real, region-routed Ingestion API.
// Create itself only resolves the plan and the client. Same pattern as
// abtest.createABTest.
func createTask(ctx context.Context, client *ingestionapi.APIClient, plan *TaskResourceModel, resp *resource.CreateResponse) {
	create, expandDiags := expandTaskCreate(plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Ingestion task", map[string]any{
		"source_id":      plan.SourceID.ValueString(),
		"destination_id": plan.DestinationID.ValueString(),
	})

	createResp, err := client.CreateTask(client.NewApiCreateTaskRequest(create), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error creating Ingestion task", "Could not create task: "+err.Error())
		return
	}

	// The task now exists in Algolia under the server-assigned UUID in
	// createResp, which is the only handle Terraform will ever have on it.
	// Persist it before the read-back below, so that a failure there leaves a
	// resource Terraform can read, update and destroy instead of orphaning a
	// task that exists remotely but not in state - and a task, unlike the other
	// Ingestion resources, is live: an orphaned enabled task keeps running on
	// its schedule and writing to its destination index with nothing tracking
	// it, while the next apply creates a second task doing the same work.
	resp.Diagnostics.Append(resp.State.Set(ctx, newTaskIdentityState(*plan, createResp.TaskID))...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := client.GetTask(client.NewApiGetTaskRequest(createResp.TaskID), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ingestion task", "Could not read back task after creation: "+err.Error())
		return
	}

	// flattenTask compares the API's input/notifications/policies against
	// the plan's configured values and only adopts the API's encoding if
	// it's not semantically equivalent. It never touches cursor - see the
	// `cursor` attribute's schema description.
	resp.Diagnostics.Append(flattenTask(apiResp, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// newTaskIdentityState returns the state to persist immediately after a
// successful CreateTask, before the read-back that can still fail. Terraform
// rejects an apply result containing unknown values, and this resource has more
// unknowns to resolve than the other four, so each is accounted for here:
//
//   - id/task_id come from the create response.
//   - created_at/updated_at/last_run/next_run are Computed-only and knowable
//     only from the read-back, so they are written as null; the next Read fills
//     them in.
//   - failure_threshold/notifications/policies are Optional+Computed because the
//     API substitutes its own defaults, so they are unknown in the plan whenever
//     the configuration omits them (UseStateForUnknown has no prior state to
//     draw on during Create). A configured value is kept; an unknown becomes
//     null.
//   - enabled is also Optional+Computed but carries a static default, so the
//     plan value is always known and is used as-is.
//   - every remaining attribute (source_id, destination_id, action,
//     subscription_action, cron, input, cursor) is Required or Optional-only,
//     so the plan holds the configuration verbatim.
//
// The model has no Object/List/Set/Map attribute, so no typed null has to be
// constructed.
func newTaskIdentityState(plan TaskResourceModel, taskID string) TaskResourceModel {
	plan.ID = types.StringValue(taskID)
	plan.TaskID = types.StringValue(taskID)
	plan.CreatedAt = types.StringNull()
	plan.UpdatedAt = types.StringNull()
	plan.LastRun = types.StringNull()
	plan.NextRun = types.StringNull()
	if plan.FailureThreshold.IsUnknown() {
		plan.FailureThreshold = types.Int64Null()
	}
	if plan.Notifications.IsUnknown() {
		plan.Notifications = types.StringNull()
	}
	if plan.Policies.IsUnknown() {
		plan.Policies = types.StringNull()
	}

	return plan
}

func (r *taskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TaskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	taskID := state.TaskID.ValueString()
	apiResp, err := client.GetTask(client.NewApiGetTaskRequest(taskID), ingestionapi.WithContext(ctx))
	if err != nil {
		var apiErr *ingestionapi.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			tflog.Warn(ctx, "Ingestion task not found; removing from state", map[string]any{"task_id": taskID})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading Ingestion task", "Could not read task "+taskID+": "+err.Error())
		return
	}

	// flattenTask preserves state.Input/Notifications/Policies as-is when
	// they are semantically equal to the API's current values, so
	// out-of-band JSON formatting differences don't create a perpetual
	// diff. See the corresponding attributes' schema descriptions. Cursor
	// is left untouched entirely.
	resp.Diagnostics.Append(flattenTask(apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *taskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TaskResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	update, expandDiags := expandTaskUpdate(&plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	taskID := plan.TaskID.ValueString()
	tflog.Debug(ctx, "Updating Ingestion task", map[string]any{"task_id": taskID})

	if _, err := client.UpdateTask(client.NewApiUpdateTaskRequest(taskID, update)); err != nil {
		resp.Diagnostics.AddError("Error updating Ingestion task", "Could not update task "+taskID+": "+err.Error())
		return
	}

	apiResp, err := client.GetTask(client.NewApiGetTaskRequest(taskID), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ingestion task", "Could not read back task after update: "+err.Error())
		return
	}

	resp.Diagnostics.Append(flattenTask(apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *taskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TaskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	taskID := state.TaskID.ValueString()
	tflog.Debug(ctx, "Deleting Ingestion task", map[string]any{"task_id": taskID})

	if _, err := client.DeleteTask(client.NewApiDeleteTaskRequest(taskID)); err != nil {
		var apiErr *ingestionapi.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return
		}

		resp.Diagnostics.AddError("Error deleting Ingestion task", "Could not delete task "+taskID+": "+err.Error())
	}
}

func (r *taskResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	taskID := req.ID
	apiResp, err := client.GetTask(client.NewApiGetTaskRequest(taskID), ingestionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error importing Ingestion task", "Could not import task "+taskID+": "+err.Error())
		return
	}

	var state TaskResourceModel
	// cursor cannot be recovered on import: flattenTask never touches it
	// (see the `cursor` attribute's schema description), and starting from
	// a zero-value model leaves it null. Set it explicitly for clarity,
	// mirroring algolia_ingestion_authentication's `input` import handling.
	state.Cursor = types.StringNull()
	resp.Diagnostics.Append(flattenTask(apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
