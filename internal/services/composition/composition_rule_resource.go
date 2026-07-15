package composition

import (
	"context"
	"errors"

	compositionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/composition"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &compositionRuleResource{}
	_ resource.ResourceWithConfigure   = &compositionRuleResource{}
	_ resource.ResourceWithImportState = &compositionRuleResource{}
)

// compositionRuleResource manages an algolia_composition_rule resource. It
// embeds base (see client.go) for Configure-time wiring and an on-demand
// Composition client - the Composition API is not region-routed, so base
// here only needs appID/apiKey.
type compositionRuleResource struct {
	base
}

// NewRuleResource returns the algolia_composition_rule resource.
func NewRuleResource() resource.Resource {
	return &compositionRuleResource{}
}

func (r *compositionRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_composition_rule"
}

func (r *compositionRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = compositionRuleResourceSchema()
}

func (r *compositionRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(r.configure(req.ProviderData)...)
}

func (r *compositionRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CompositionRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	compositionID := plan.CompositionID.ValueString()

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

	tflog.Debug(ctx, "Creating composition rule", map[string]any{"composition_id": compositionID, "object_id": objectID})

	rule, expandDiags := expandCompositionRule(objectID, &plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	putResp, err := client.PutCompositionRule(client.NewApiPutCompositionRuleRequest(compositionID, objectID, rule))
	if err != nil {
		resp.Diagnostics.AddError("Error creating composition rule", "Could not create composition rule "+objectID+" on composition "+compositionID+": "+err.Error())
		return
	}

	if err := waitForCompositionTask(client, compositionID, putResp.TaskID); err != nil {
		resp.Diagnostics.AddError("Error waiting for composition rule creation", "Could not confirm composition rule creation: "+err.Error())
		return
	}

	apiResp, err := client.GetRule(client.NewApiGetRuleRequest(compositionID, objectID))
	if err != nil {
		resp.Diagnostics.AddError("Error reading composition rule", "Could not read composition rule "+objectID+" on composition "+compositionID+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(flattenCompositionRule(compositionID, apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *compositionRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CompositionRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	compositionID := state.CompositionID.ValueString()
	objectID := state.ObjectID.ValueString()

	apiResp, err := client.GetRule(client.NewApiGetRuleRequest(compositionID, objectID))
	if err != nil {
		var apiErr *compositionapi.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			tflog.Warn(ctx, "Composition rule not found; removing from state", map[string]any{"composition_id": compositionID, "object_id": objectID})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading composition rule", "Could not read composition rule "+objectID+" on composition "+compositionID+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(flattenCompositionRule(compositionID, apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *compositionRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CompositionRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	compositionID := plan.CompositionID.ValueString()
	objectID := plan.ObjectID.ValueString()

	tflog.Debug(ctx, "Updating composition rule", map[string]any{"composition_id": compositionID, "object_id": objectID})

	rule, expandDiags := expandCompositionRule(objectID, &plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	putResp, err := client.PutCompositionRule(client.NewApiPutCompositionRuleRequest(compositionID, objectID, rule))
	if err != nil {
		resp.Diagnostics.AddError("Error updating composition rule", "Could not update composition rule "+objectID+" on composition "+compositionID+": "+err.Error())
		return
	}

	if err := waitForCompositionTask(client, compositionID, putResp.TaskID); err != nil {
		resp.Diagnostics.AddError("Error waiting for composition rule update", "Could not confirm composition rule update: "+err.Error())
		return
	}

	apiResp, err := client.GetRule(client.NewApiGetRuleRequest(compositionID, objectID))
	if err != nil {
		resp.Diagnostics.AddError("Error reading composition rule", "Could not read composition rule "+objectID+" on composition "+compositionID+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(flattenCompositionRule(compositionID, apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *compositionRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CompositionRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	compositionID := state.CompositionID.ValueString()
	objectID := state.ObjectID.ValueString()

	tflog.Debug(ctx, "Deleting composition rule", map[string]any{"composition_id": compositionID, "object_id": objectID})

	deleteResp, err := client.DeleteCompositionRule(client.NewApiDeleteCompositionRuleRequest(compositionID, objectID))
	if err != nil {
		var apiErr *compositionapi.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return
		}

		resp.Diagnostics.AddError("Error deleting composition rule", "Could not delete composition rule "+objectID+" on composition "+compositionID+": "+err.Error())
		return
	}

	if err := waitForCompositionTask(client, compositionID, deleteResp.TaskID); err != nil {
		var apiErr *compositionapi.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return
		}

		resp.Diagnostics.AddError("Error waiting for composition rule deletion", "Could not confirm composition rule deletion: "+err.Error())
	}
}

func (r *compositionRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	compositionID, objectID, err := parseCompositionRuleImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := client.GetRule(client.NewApiGetRuleRequest(compositionID, objectID))
	if err != nil {
		resp.Diagnostics.AddError("Error importing composition rule", "Could not import composition rule "+req.ID+": "+err.Error())
		return
	}

	var state CompositionRuleResourceModel
	resp.Diagnostics.Append(flattenCompositionRule(compositionID, apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
