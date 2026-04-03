package agent

import (
	"context"
	"fmt"

	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &agentDataSource{}

type agentDataSource struct {
	client *Client
}

func NewDataSource() datasource.DataSource {
	return &agentDataSource{}
}

func (d *agentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent"
}

func (d *agentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = agentDataSourceSchema()
}

func (d *agentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	agentClient, ok := data.AgentClient.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Agent Client Type",
			fmt.Sprintf("Expected *agent.Client, got: %T", data.AgentClient),
		)
		return
	}

	d.client = agentClient
}

func (d *agentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model AgentResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	agentID := model.ID.ValueString()
	tflog.Debug(ctx, "Reading agent data source", map[string]interface{}{"id": agentID})

	apiResp, err := d.client.GetAgent(ctx, agentID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading agent", "Could not read agent "+agentID+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(flattenAgentResponse(ctx, apiResp, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set plan-only fields for data source.
	model.Publish = types.BoolValue(apiResp.Status == "published")
	model.DeletionProtection = types.BoolValue(true)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
