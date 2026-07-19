package agent

import (
	"context"
	"errors"
	"fmt"

	agentStudio "github.com/algolia/algoliasearch-client-go/v4/algolia/agent-studio"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &agentResource{}
	_ resource.ResourceWithImportState = &agentResource{}
)

type agentResource struct {
	client *agentStudio.APIClient
}

func NewResource() resource.Resource {
	return &agentResource{}
}

func (r *agentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent"
}

func (r *agentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = agentResourceSchema()
}

func (r *agentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = data.AgentClient
}

func (r *agentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AgentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating agent", map[string]interface{}{"name": plan.Name.ValueString()})

	apiReq, diags := expandAgentConfigCreate(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.CreateAgent(r.client.NewApiCreateAgentRequest(apiReq), agentStudio.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error creating agent", "Could not create agent: "+err.Error())
		return
	}

	// Publish if requested.
	if plan.Publish.ValueBool() {
		apiResp, err = r.client.PublishAgent(r.client.NewApiPublishAgentRequest(apiResp.Id), agentStudio.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError("Error publishing agent", "Agent created but could not be published: "+err.Error())
			return
		}
	}

	resp.Diagnostics.Append(hydrateAgentResourceState(ctx, apiResp, plan.DeletionProtection, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *agentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AgentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	agentID := state.ID.ValueString()
	tflog.Debug(ctx, "Reading agent", map[string]interface{}{"id": agentID})

	apiResp, err := r.client.GetAgent(r.client.NewApiGetAgentRequest(agentID), agentStudio.WithContext(ctx))
	if err != nil {
		var apiErr *agentStudio.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			tflog.Warn(ctx, "Agent not found; removing from state", map[string]interface{}{"id": agentID})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading agent", "Could not read agent "+agentID+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(hydrateAgentResourceState(ctx, apiResp, state.DeletionProtection, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *agentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AgentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state AgentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validatePublishTransition(state, plan); err != nil {
		resp.Diagnostics.AddError("Unpublish Not Supported", err.Error())
		return
	}

	agentID := plan.ID.ValueString()
	tflog.Debug(ctx, "Updating agent", map[string]interface{}{"id": agentID})

	apiReq, diags := expandAgentConfigUpdate(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.UpdateAgent(r.client.NewApiUpdateAgentRequest(agentID, apiReq), agentStudio.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error updating agent", "Could not update agent "+agentID+": "+err.Error())
		return
	}

	// Only publish on update when transitioning from draft to published.
	if shouldPublishAfterUpdate(state, plan) {
		apiResp, err = r.client.PublishAgent(r.client.NewApiPublishAgentRequest(agentID), agentStudio.WithContext(ctx))
		if err != nil {
			resp.Diagnostics.AddError("Error publishing agent", "Agent updated but could not be published: "+err.Error())
			return
		}
	}

	resp.Diagnostics.Append(hydrateAgentResourceState(ctx, apiResp, plan.DeletionProtection, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *agentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AgentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	agentID := state.ID.ValueString()

	if deletionProtectionValue(state.DeletionProtection).ValueBool() {
		resp.Diagnostics.AddError(
			"Deletion Protection Enabled",
			fmt.Sprintf("Cannot delete agent %q because deletion_protection is enabled. "+
				"Set deletion_protection = false and apply before destroying.", agentID),
		)
		return
	}

	tflog.Debug(ctx, "Deleting agent", map[string]interface{}{"id": agentID})

	if err := r.client.DeleteAgent(r.client.NewApiDeleteAgentRequest(agentID), agentStudio.WithContext(ctx)); err != nil {
		resp.Diagnostics.AddError("Error deleting agent", "Could not delete agent "+agentID+": "+err.Error())
		return
	}
}

func (r *agentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	apiResp, err := r.client.GetAgent(r.client.NewApiGetAgentRequest(req.ID), agentStudio.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error importing agent", "Could not import agent "+req.ID+": "+err.Error())
		return
	}

	var state AgentResourceModel
	resp.Diagnostics.Append(hydrateImportedAgentResourceState(ctx, apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
