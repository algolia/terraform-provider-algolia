package composition

import (
	"context"
	"fmt"

	compositionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/composition"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"
	"github.com/algolia/terraform-provider-algolia/internal/algoliawait"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &compositionResource{}
	_ resource.ResourceWithConfigure   = &compositionResource{}
	_ resource.ResourceWithImportState = &compositionResource{}
)

// compositionResource manages an algolia_composition resource. It embeds
// base (see client.go) for Configure-time wiring and an on-demand
// Composition client - the Composition API is not region-routed, so base
// here only needs appID/apiKey.
type compositionResource struct {
	base
}

// compositionKind names this resource inside a diagnostic sentence. A composition
// is addressed by its own objectID alone, so unlike a composition rule it needs
// no parent qualifier.
const compositionKind = "composition"

// NewResource returns the algolia_composition resource.
func NewResource() resource.Resource {
	return &compositionResource{}
}

func (r *compositionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_composition"
}

func (r *compositionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = compositionResourceSchema()
}

func (r *compositionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(r.configure(req.ProviderData)...)
}

func (r *compositionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CompositionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	objectID := plan.ObjectID.ValueString()
	tflog.Debug(ctx, "Creating composition", map[string]any{"object_id": objectID})

	comp, expandDiags := expandComposition(objectID, &plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	putResp, err := client.PutComposition(client.NewApiPutCompositionRequest(objectID, comp), compositionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(algoliaerr.Object(compositionKind, objectID).Message(algoliaerr.Create, err))
		return
	}

	if err := waitForCompositionTask(ctx, client, objectID, putResp.TaskID); err != nil {
		resp.Diagnostics.AddError(algoliaerr.WaitMessage(compositionKind, algoliaerr.Create, err))
		return
	}

	apiResp, err := client.GetComposition(client.NewApiGetCompositionRequest(objectID), compositionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(algoliaerr.Object(compositionKind, objectID).Message(algoliaerr.Read, err))
		return
	}

	resp.Diagnostics.Append(flattenComposition(apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *compositionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CompositionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	objectID := state.ObjectID.ValueString()

	apiResp, err := client.GetComposition(client.NewApiGetCompositionRequest(objectID), compositionapi.WithContext(ctx))
	if err != nil {
		if algoliaerr.IsNotFound(err) {
			tflog.Warn(ctx, "Composition not found; removing from state", map[string]any{"object_id": objectID})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(algoliaerr.Object(compositionKind, objectID).Message(algoliaerr.Read, err))
		return
	}

	resp.Diagnostics.Append(flattenComposition(apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *compositionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CompositionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	objectID := plan.ObjectID.ValueString()
	tflog.Debug(ctx, "Updating composition", map[string]any{"object_id": objectID})

	comp, expandDiags := expandComposition(objectID, &plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	putResp, err := client.PutComposition(client.NewApiPutCompositionRequest(objectID, comp), compositionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(algoliaerr.Object(compositionKind, objectID).Message(algoliaerr.Update, err))
		return
	}

	if err := waitForCompositionTask(ctx, client, objectID, putResp.TaskID); err != nil {
		resp.Diagnostics.AddError(algoliaerr.WaitMessage(compositionKind, algoliaerr.Update, err))
		return
	}

	apiResp, err := client.GetComposition(client.NewApiGetCompositionRequest(objectID), compositionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(algoliaerr.Object(compositionKind, objectID).Message(algoliaerr.Read, err))
		return
	}

	resp.Diagnostics.Append(flattenComposition(apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *compositionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CompositionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	objectID := state.ObjectID.ValueString()
	tflog.Debug(ctx, "Deleting composition", map[string]any{"object_id": objectID})

	deleteResp, err := client.DeleteComposition(client.NewApiDeleteCompositionRequest(objectID), compositionapi.WithContext(ctx))
	if err != nil {
		if algoliaerr.IsNotFound(err) {
			return
		}

		resp.Diagnostics.AddError(algoliaerr.Object(compositionKind, objectID).Message(algoliaerr.Delete, err))
		return
	}

	if err := waitForCompositionTask(ctx, client, objectID, deleteResp.TaskID); err != nil {
		if algoliaerr.IsNotFound(err) {
			return
		}

		resp.Diagnostics.AddError(algoliaerr.WaitMessage(compositionKind, algoliaerr.Delete, err))
	}
}

func (r *compositionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := client.GetComposition(client.NewApiGetCompositionRequest(req.ID), compositionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(algoliaerr.Object(compositionKind, req.ID).Message(algoliaerr.Import, err))
		return
	}

	var state CompositionResourceModel
	resp.Diagnostics.Append(flattenComposition(apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// waitForCompositionTask polls GetTask until the given task completes,
// mirroring waitForRecommendRuleTask in the recommend package (which polls a
// differently-routed task-status endpoint per index/model - Composition has
// its own, scoped by compositionID, shared by both algolia_composition and
// algolia_composition_rule).
func waitForCompositionTask(ctx context.Context, client *compositionapi.APIClient, compositionID string, taskID int64) error {
	return algoliawait.Until(ctx, fmt.Sprintf("task %d on composition %q", taskID, compositionID), func(ctx context.Context) (bool, error) {
		resp, err := client.GetTask(client.NewApiGetTaskRequest(compositionID, taskID), compositionapi.WithContext(ctx))
		if err != nil {
			return false, err
		}

		return resp.Status == compositionapi.TASK_STATUS_PUBLISHED, nil
	})
}
