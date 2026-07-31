package agent

import (
	"context"
	"fmt"

	"github.com/algolia/terraform-provider-algolia/internal/deletionprotection"

	agentStudio "github.com/algolia/algoliasearch-client-go/v4/algolia/agent-studio"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"
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

	doc, err := createAgent(ctx, r.client, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating agent", "Could not create agent: "+err.Error())
		return
	}

	// Publish if requested.
	if plan.Publish.ValueBool() {
		doc, err = publishAgent(ctx, r.client, doc.agent.Id)
		if err != nil {
			resp.Diagnostics.AddError("Error publishing agent", "Agent created but could not be published: "+err.Error())
			return
		}
	}

	resp.Diagnostics.Append(hydrateAgentResourceState(ctx, doc, plan.DeletionProtection, &plan)...)
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

	doc, err := getAgent(ctx, r.client, agentID)
	if err != nil {
		if algoliaerr.IsNotFound(err) {
			tflog.Warn(ctx, "Agent not found; removing from state", map[string]interface{}{"id": agentID})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading agent", "Could not read agent "+agentID+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(hydrateAgentResourceState(ctx, doc, state.DeletionProtection, &state)...)
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

	doc, err := updateAgent(ctx, r.client, agentID, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating agent", "Could not update agent "+agentID+": "+err.Error())
		return
	}

	// Only publish on update when transitioning from draft to published.
	if shouldPublishAfterUpdate(state, plan) {
		doc, err = publishAgent(ctx, r.client, agentID)
		if err != nil {
			resp.Diagnostics.AddError("Error publishing agent", "Agent updated but could not be published: "+err.Error())
			return
		}
	}

	resp.Diagnostics.Append(hydrateAgentResourceState(ctx, doc, plan.DeletionProtection, &plan)...)
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

	if deletionprotection.Enabled(state.DeletionProtection) {
		resp.Diagnostics.Append(deletionprotection.Refuse(fmt.Sprintf("agent %q", agentID)))
		return
	}

	tflog.Debug(ctx, "Deleting agent", map[string]interface{}{"id": agentID})

	if err := r.client.DeleteAgent(r.client.NewApiDeleteAgentRequest(agentID), agentStudio.WithContext(ctx)); err != nil {
		resp.Diagnostics.AddError("Error deleting agent", "Could not delete agent "+agentID+": "+err.Error())
		return
	}
}

func (r *agentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	doc, err := getAgent(ctx, r.client, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing agent", "Could not import agent "+req.ID+": "+err.Error())
		return
	}

	var state AgentResourceModel
	resp.Diagnostics.Append(hydrateImportedAgentResourceState(ctx, doc, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
