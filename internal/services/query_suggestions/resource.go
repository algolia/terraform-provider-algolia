package query_suggestions

import (
	"context"
	"fmt"
	"net/http"

	suggestions "github.com/algolia/algoliasearch-client-go/v4/algolia/query-suggestions"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &querySuggestionsConfigResource{}
	_ resource.ResourceWithImportState = &querySuggestionsConfigResource{}
)

type querySuggestionsConfigResource struct {
	appID  string
	apiKey string
}

func NewResource() resource.Resource {
	return &querySuggestionsConfigResource{}
}

func (r *querySuggestionsConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_query_suggestions_config"
}

func (r *querySuggestionsConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = querySuggestionsConfigResourceSchema()
}

func (r *querySuggestionsConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.appID = data.AppID
	r.apiKey = data.APIKey
}

func (r *querySuggestionsConfigResource) newClient(region string) (*suggestions.APIClient, error) {
	return suggestions.NewClient(r.appID, r.apiKey, suggestions.Region(region))
}

func (r *querySuggestionsConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan QuerySuggestionsConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := plan.IndexName.ValueString()
	region := plan.Region.ValueString()
	tflog.Debug(ctx, "Creating query suggestions config", map[string]interface{}{"index_name": indexName, "region": region})

	client, err := r.newClient(region)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Query Suggestions client", err.Error())
		return
	}

	cfg, diags := expandConfigurationWithIndex(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err = client.CreateConfig(client.NewApiCreateConfigRequest(cfg))
	if err != nil {
		resp.Diagnostics.AddError("Error creating Query Suggestions config", "Could not create config for index "+indexName+": "+err.Error())
		return
	}

	apiResp, err := client.GetConfig(client.NewApiGetConfigRequest(indexName))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Query Suggestions config after create", err.Error())
		return
	}

	deletionProtection := plan.DeletionProtection

	resp.Diagnostics.Append(flattenConfigurationResponse(ctx, apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Region = plan.Region
	plan.DeletionProtection = deletionProtection

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *querySuggestionsConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state QuerySuggestionsConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := state.IndexName.ValueString()
	region := state.Region.ValueString()
	tflog.Debug(ctx, "Reading query suggestions config", map[string]interface{}{"index_name": indexName, "region": region})

	client, err := r.newClient(region)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Query Suggestions client", err.Error())
		return
	}

	apiResp, err := client.GetConfig(client.NewApiGetConfigRequest(indexName))
	if err != nil {
		if isNotFound(err) {
			tflog.Warn(ctx, "Query Suggestions config not found; removing from state", map[string]interface{}{"index_name": indexName})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Query Suggestions config", "Could not read config for index "+indexName+": "+err.Error())
		return
	}

	region2 := state.Region
	deletionProtection := state.DeletionProtection

	resp.Diagnostics.Append(flattenConfigurationResponse(ctx, apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Region = region2
	state.DeletionProtection = deletionProtection

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *querySuggestionsConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan QuerySuggestionsConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := plan.IndexName.ValueString()
	region := plan.Region.ValueString()
	tflog.Debug(ctx, "Updating query suggestions config", map[string]interface{}{"index_name": indexName, "region": region})

	client, err := r.newClient(region)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Query Suggestions client", err.Error())
		return
	}

	cfg, diags := expandConfiguration(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err = client.UpdateConfig(client.NewApiUpdateConfigRequest(indexName, cfg))
	if err != nil {
		resp.Diagnostics.AddError("Error updating Query Suggestions config", "Could not update config for index "+indexName+": "+err.Error())
		return
	}

	apiResp, err := client.GetConfig(client.NewApiGetConfigRequest(indexName))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Query Suggestions config after update", err.Error())
		return
	}

	deletionProtection := plan.DeletionProtection

	resp.Diagnostics.Append(flattenConfigurationResponse(ctx, apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Region = plan.Region
	plan.DeletionProtection = deletionProtection

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *querySuggestionsConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state QuerySuggestionsConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexName := state.IndexName.ValueString()

	if !state.DeletionProtection.IsNull() && state.DeletionProtection.ValueBool() {
		resp.Diagnostics.AddError(
			"Deletion Protection Enabled",
			fmt.Sprintf("Cannot delete Query Suggestions config %q because deletion_protection is enabled. "+
				"Set deletion_protection = false and apply before destroying.", indexName),
		)
		return
	}

	region := state.Region.ValueString()
	tflog.Debug(ctx, "Deleting query suggestions config", map[string]interface{}{"index_name": indexName, "region": region})

	client, err := r.newClient(region)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Query Suggestions client", err.Error())
		return
	}

	_, err = client.DeleteConfig(client.NewApiDeleteConfigRequest(indexName))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Query Suggestions config", "Could not delete config for index "+indexName+": "+err.Error())
		return
	}
}

func (r *querySuggestionsConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("index_name"), req, resp)
}

// isNotFound checks whether an API error is a 404.
func isNotFound(err error) bool {
	type statusCoder interface {
		StatusCode() int
	}
	type httpStatusCoder interface {
		HTTPStatusCode() int
	}

	if sc, ok := err.(statusCoder); ok {
		return sc.StatusCode() == http.StatusNotFound
	}
	if hsc, ok := err.(httpStatusCoder); ok {
		return hsc.HTTPStatusCode() == http.StatusNotFound
	}

	return false
}
