package agentprovider

import (
	"context"
	"fmt"

	agentstudio "github.com/algolia/terraform-provider-algolia/internal/services/agent"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &agentProviderDataSource{}
	_ datasource.DataSourceWithConfigure = &agentProviderDataSource{}
	_ datasource.DataSource              = &agentProviderModelsDataSource{}
	_ datasource.DataSourceWithConfigure = &agentProviderModelsDataSource{}
)

type agentProviderDataSource struct {
	client *agentstudio.Client
}

func NewDataSource() datasource.DataSource {
	return &agentProviderDataSource{}
}

func (d *agentProviderDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_provider"
}

func (d *agentProviderDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = agentProviderDataSourceSchema()
}

func (d *agentProviderDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*providertypes.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
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

	d.client = agentClient
}

func (d *agentProviderDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model AgentProviderDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	providerID := model.ProviderID.ValueString()
	tflog.Debug(ctx, "Reading agent provider data source", map[string]any{"provider_id": providerID})

	apiResp, err := d.client.GetProvider(ctx, providerID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading provider", "Could not read provider "+providerID+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(hydrateAgentProviderDataSourceState(ctx, apiResp, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

type agentProviderModelsDataSource struct {
	client *agentstudio.Client
}

func NewModelsDataSource() datasource.DataSource {
	return &agentProviderModelsDataSource{}
}

func (d *agentProviderModelsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_provider_models"
}

func (d *agentProviderModelsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = agentProviderModelsDataSourceSchema()
}

func (d *agentProviderModelsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*providertypes.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
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

	d.client = agentClient
}

func (d *agentProviderModelsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model AgentProviderModelsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	providerID := model.ProviderID.ValueString()
	tflog.Debug(ctx, "Reading provider models data source", map[string]any{"provider_id": providerID})

	models, err := d.client.GetProviderModels(ctx, providerID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading provider models", "Could not read provider models for "+providerID+": "+err.Error())
		return
	}

	modelsValue, diags := types.ListValueFrom(ctx, types.StringType, models)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.Models = modelsValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
