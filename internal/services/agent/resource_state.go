package agent

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func hydrateAgentResourceState(ctx context.Context, resp *AgentResponse, deletionProtection types.Bool, model *AgentResourceModel) diag.Diagnostics {
	diags := flattenAgentResponse(ctx, resp, model)
	if diags.HasError() {
		return diags
	}

	model.Publish = remotePublishValue(resp.Status)
	model.DeletionProtection = deletionProtectionValue(deletionProtection)

	return diags
}

func hydrateImportedAgentResourceState(ctx context.Context, resp *AgentResponse, model *AgentResourceModel) diag.Diagnostics {
	return hydrateAgentResourceState(ctx, resp, types.BoolValue(true), model)
}

func hydrateAgentDataSourceState(ctx context.Context, resp *AgentResponse, model *AgentDataSourceModel) diag.Diagnostics {
	resourceModel := &AgentResourceModel{}
	diags := flattenAgentResponse(ctx, resp, resourceModel)
	if diags.HasError() {
		return diags
	}

	model.ID = resourceModel.ID
	model.Name = resourceModel.Name
	model.Description = resourceModel.Description
	model.Instructions = resourceModel.Instructions
	model.SystemPrompt = resourceModel.SystemPrompt
	model.ProviderID = resourceModel.ProviderID
	model.Model = resourceModel.Model
	model.TemplateType = resourceModel.TemplateType
	model.Config = resourceModel.Config
	model.Publish = remotePublishValue(resp.Status)
	model.ToolAlgoliaSearch = resourceModel.ToolAlgoliaSearch
	model.ToolAlgoliaRecommend = resourceModel.ToolAlgoliaRecommend
	model.ToolClientSide = resourceModel.ToolClientSide
	model.ToolMCP = resourceModel.ToolMCP
	model.Status = resourceModel.Status
	model.CreatedAt = resourceModel.CreatedAt
	model.UpdatedAt = resourceModel.UpdatedAt

	return diags
}

func validatePublishTransition(state, plan AgentResourceModel) error {
	if state.Status.IsNull() || state.Status.IsUnknown() || plan.Publish.IsNull() || plan.Publish.IsUnknown() {
		return nil
	}

	if state.Status.ValueString() == "published" && !plan.Publish.ValueBool() {
		return fmt.Errorf("unpublish is not supported by Agent Studio; keep publish = true or recreate the agent in draft state")
	}

	return nil
}

func shouldPublishAfterUpdate(state, plan AgentResourceModel) bool {
	if plan.Publish.IsNull() || plan.Publish.IsUnknown() || !plan.Publish.ValueBool() {
		return false
	}

	if state.Status.IsNull() || state.Status.IsUnknown() {
		return true
	}

	return state.Status.ValueString() != "published"
}

func remotePublishValue(status string) types.Bool {
	return types.BoolValue(status == "published")
}

func deletionProtectionValue(v types.Bool) types.Bool {
	if v.IsNull() || v.IsUnknown() {
		return types.BoolValue(true)
	}

	return v
}
