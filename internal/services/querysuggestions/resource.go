package querysuggestions

import (
	"context"
	"errors"
	"fmt"
	"time"

	suggestions "github.com/algolia/algoliasearch-client-go/v4/algolia/query-suggestions"
	"github.com/algolia/terraform-provider-algolia/internal/analyticsregion"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &querySuggestionsResource{}
	_ resource.ResourceWithConfigure   = &querySuggestionsResource{}
	_ resource.ResourceWithImportState = &querySuggestionsResource{}
)

type querySuggestionsResource struct {
	appID                  string
	apiKey                 string
	querySuggestionsRegion string
}

func NewResource() resource.Resource {
	return &querySuggestionsResource{}
}

func (r *querySuggestionsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_query_suggestions"
}

func (r *querySuggestionsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = querySuggestionsResourceSchema()
}

func (r *querySuggestionsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.querySuggestionsRegion = data.QuerySuggestionsRegion
}

func (r *querySuggestionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan QuerySuggestionsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, configDiags := buildConfigurationWithIndex(&plan)
	resp.Diagnostics.Append(configDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := client.CreateConfig(client.NewApiCreateConfigRequest(config)); err != nil {
		resp.Diagnostics.AddError("Error creating Query Suggestions config", "Could not create Query Suggestions config "+plan.IndexName.ValueString()+": "+err.Error())
		return
	}

	apiResp, err := waitForQuerySuggestionsConfig(ctx, client, plan.IndexName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Query Suggestions config", "Could not read Query Suggestions config after creation: "+err.Error())
		return
	}

	resp.Diagnostics.Append(hydrateQuerySuggestionsModel(apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *querySuggestionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state QuerySuggestionsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := client.GetConfig(client.NewApiGetConfigRequest(state.IndexName.ValueString()))
	if err != nil {
		var apiErr *suggestions.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading Query Suggestions config", "Could not read Query Suggestions config "+state.IndexName.ValueString()+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(hydrateQuerySuggestionsModel(apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *querySuggestionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan QuerySuggestionsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	configWithIndex, configDiags := buildConfigurationWithIndex(&plan)
	resp.Diagnostics.Append(configDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	configuration := suggestions.NewConfiguration(configWithIndex.GetSourceIndices())
	configuration.Languages = configWithIndex.Languages
	configuration.Exclude = configWithIndex.Exclude

	if _, err := client.UpdateConfig(client.NewApiUpdateConfigRequest(plan.IndexName.ValueString(), configuration)); err != nil {
		resp.Diagnostics.AddError("Error updating Query Suggestions config", "Could not update Query Suggestions config "+plan.IndexName.ValueString()+": "+err.Error())
		return
	}

	apiResp, err := waitForQuerySuggestionsConfig(ctx, client, plan.IndexName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Query Suggestions config", "Could not read Query Suggestions config after update: "+err.Error())
		return
	}

	resp.Diagnostics.Append(hydrateQuerySuggestionsModel(apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *querySuggestionsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state QuerySuggestionsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := client.DeleteConfig(client.NewApiDeleteConfigRequest(state.IndexName.ValueString())); err != nil {
		var apiErr *suggestions.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return
		}

		resp.Diagnostics.AddError("Error deleting Query Suggestions config", "Could not delete Query Suggestions config "+state.IndexName.ValueString()+": "+err.Error())
	}
}

func (r *querySuggestionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	indexName, err := parseImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := client.GetConfig(client.NewApiGetConfigRequest(indexName))
	if err != nil {
		resp.Diagnostics.AddError("Error importing Query Suggestions config", "Could not import Query Suggestions config "+req.ID+": "+err.Error())
		return
	}

	var state QuerySuggestionsResourceModel
	resp.Diagnostics.Append(hydrateQuerySuggestionsModel(apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *querySuggestionsResource) client() (*suggestions.APIClient, diag.Diagnostics) {
	var diags diag.Diagnostics

	client, err := analyticsregion.NewQuerySuggestionsClient(r.appID, r.apiKey, r.querySuggestionsRegion)
	if err != nil {
		diags.AddError("Unable to create Query Suggestions client", err.Error())
		return nil, diags
	}

	tflog.Debug(context.Background(), "Configured Query Suggestions client", map[string]any{"region": r.querySuggestionsRegion})
	return client, diags
}

func waitForQuerySuggestionsConfig(ctx context.Context, client *suggestions.APIClient, indexName string) (*suggestions.ConfigurationResponse, error) {
	deadline := time.Now().Add(2 * time.Minute)
	interval := 2 * time.Second
	for time.Now().Before(deadline) {
		resp, err := client.GetConfig(client.NewApiGetConfigRequest(indexName))
		if err == nil {
			return resp, nil
		}
		var apiErr *suggestions.APIError
		if errors.As(err, &apiErr) && apiErr.Status != 404 {
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
		if interval < 10*time.Second {
			interval += time.Second
		}
	}
	return nil, fmt.Errorf("query suggestions config %q did not become readable within 2 minutes", indexName)
}
