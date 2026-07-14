package ingestion

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &taskDataSource{}
	_ datasource.DataSourceWithConfigure = &taskDataSource{}
)

// taskDataSource reads an algolia_ingestion_task resource's configuration,
// including `input`/`notifications`/`policies`/`cursor` in full: GetTask
// does not redact any of them (like GetSource/GetDestination/
// GetTransformation, unlike GetAuthentication).
type taskDataSource struct {
	base
}

// NewTaskDataSource returns the algolia_ingestion_task data source.
func NewTaskDataSource() datasource.DataSource {
	return &taskDataSource{}
}

func (d *taskDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingestion_task"
}

func (d *taskDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = taskDataSourceSchema()
}

func (d *taskDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(d.configure(req.ProviderData)...)
}

func (d *taskDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model TaskDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := d.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	taskID := model.TaskID.ValueString()
	tflog.Debug(ctx, "Reading Ingestion task data source", map[string]any{"task_id": taskID})

	apiResp, err := client.GetTask(client.NewApiGetTaskRequest(taskID))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ingestion task", "Could not read task "+taskID+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(flattenTaskDataSource(apiResp, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
