package agentprovider

import (
	"context"
	"errors"
	"fmt"

	agentstudio "github.com/algolia/terraform-provider-algolia/internal/services/agent"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                   = &agentProviderResource{}
	_ resource.ResourceWithConfigure      = &agentProviderResource{}
	_ resource.ResourceWithImportState    = &agentProviderResource{}
	_ resource.ResourceWithValidateConfig = &agentProviderResource{}
)

type agentProviderResource struct {
	client *agentstudio.Client
}

func NewResource() resource.Resource {
	return &agentProviderResource{}
}

func (r *agentProviderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_provider"
}

func (r *agentProviderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = agentProviderResourceSchema()
}

func (r *agentProviderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	agentClient, ok := data.AgentClient.(*agentstudio.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Agent Client Type",
			fmt.Sprintf("Expected *agent.Client, got: %T", data.AgentClient),
		)
		return
	}

	r.client = agentClient
}

func (r *agentProviderResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config AgentProviderResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(validateAgentProviderConfig(ctx, config)...)
}

func (r *agentProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AgentProviderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating agent provider", map[string]any{
		"name":          plan.Name.ValueString(),
		"provider_name": plan.ProviderName.ValueString(),
	})

	apiReq, diags := providerRequestFromModel(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.CreateProvider(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating provider", "Could not create provider: "+err.Error())
		return
	}

	resp.Diagnostics.Append(hydrateAgentProviderResourceState(ctx, apiResp, plan, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *agentProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AgentProviderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	providerID := state.ID.ValueString()
	tflog.Debug(ctx, "Reading agent provider", map[string]any{"id": providerID})

	apiResp, err := r.client.GetProvider(ctx, providerID)
	if err != nil {
		var apiErr *agentstudio.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			tflog.Warn(ctx, "Provider not found; removing from state", map[string]any{"id": providerID})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading provider", "Could not read provider "+providerID+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(hydrateAgentProviderResourceState(ctx, apiResp, state, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *agentProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AgentProviderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	providerID := plan.ID.ValueString()
	tflog.Debug(ctx, "Updating agent provider", map[string]any{"id": providerID})

	apiReq, diags := providerRequestFromModelForUpdate(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.UpdateProvider(ctx, providerID, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating provider", "Could not update provider "+providerID+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(hydrateAgentProviderResourceState(ctx, apiResp, plan, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *agentProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AgentProviderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	providerID := state.ID.ValueString()
	tflog.Debug(ctx, "Deleting agent provider", map[string]any{"id": providerID})

	if err := r.client.DeleteProvider(ctx, providerID); err != nil {
		resp.Diagnostics.AddError("Error deleting provider", "Could not delete provider "+providerID+": "+err.Error())
		return
	}
}

func (r *agentProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	apiResp, err := r.client.GetProvider(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing provider", "Could not import provider "+req.ID+": "+err.Error())
		return
	}

	var state AgentProviderResourceModel
	resp.Diagnostics.Append(hydrateImportedAgentProviderResourceState(ctx, apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
