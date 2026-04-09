package personalization

import (
	"context"
	"fmt"

	api "github.com/algolia/algoliasearch-client-go/v4/algolia/personalization"
	"github.com/algolia/terraform-provider-algolia/internal/analyticsregion"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.Resource                  = &personalizationStrategyResource{}
	_ resource.ResourceWithConfigure     = &personalizationStrategyResource{}
	_ resource.ResourceWithImportState   = &personalizationStrategyResource{}
	_ datasource.DataSource              = &personalizationStrategyDataSource{}
	_ datasource.DataSourceWithConfigure = &personalizationStrategyDataSource{}
)

type personalizationStrategyResource struct {
	appID                 string
	apiKey                string
	personalizationRegion string
}

type personalizationStrategyDataSource struct {
	appID                 string
	apiKey                string
	personalizationRegion string
}

func NewResource() resource.Resource {
	return &personalizationStrategyResource{}
}

func NewDataSource() datasource.DataSource {
	return &personalizationStrategyDataSource{}
}

func (r *personalizationStrategyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_personalization_strategy"
}

func (r *personalizationStrategyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = personalizationStrategyResourceSchema()
}

func (r *personalizationStrategyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.personalizationRegion = data.PersonalizationRegion
}

func (r *personalizationStrategyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PersonalizationStrategyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	strategy, diags := buildPersonalizationStrategyRequest(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := client.SetPersonalizationStrategy(client.NewApiSetPersonalizationStrategyRequest(strategy)); err != nil {
		resp.Diagnostics.AddError("Error setting personalization strategy", err.Error())
		return
	}

	apiResp, err := waitForPersonalizationStrategy(ctx, client, strategy)
	if err != nil {
		resp.Diagnostics.AddError("Error reading personalization strategy", err.Error())
		return
	}

	resp.Diagnostics.Append(hydratePersonalizationStrategyModel(apiResp, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *personalizationStrategyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PersonalizationStrategyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := client.GetPersonalizationStrategy()
	if err != nil {
		resp.Diagnostics.AddError("Error reading personalization strategy", err.Error())
		return
	}

	resp.Diagnostics.Append(hydratePersonalizationStrategyModel(apiResp, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *personalizationStrategyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PersonalizationStrategyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	strategy, diags := buildPersonalizationStrategyRequest(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := client.SetPersonalizationStrategy(client.NewApiSetPersonalizationStrategyRequest(strategy)); err != nil {
		resp.Diagnostics.AddError("Error updating personalization strategy", err.Error())
		return
	}

	apiResp, err := waitForPersonalizationStrategy(ctx, client, strategy)
	if err != nil {
		resp.Diagnostics.AddError("Error reading personalization strategy", err.Error())
		return
	}

	resp.Diagnostics.Append(hydratePersonalizationStrategyModel(apiResp, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *personalizationStrategyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := client.GetPersonalizationStrategy()
	if err != nil {
		resp.Diagnostics.AddError("Error reading personalization strategy", err.Error())
		return
	}

	strategy := disabledPersonalizationStrategy(current)
	if _, err := client.SetPersonalizationStrategy(client.NewApiSetPersonalizationStrategyRequest(strategy)); err != nil {
		resp.Diagnostics.AddError("Error resetting personalization strategy", err.Error())
		return
	}

	_, err = waitForPersonalizationStrategy(ctx, client, strategy)
	if err != nil {
		resp.Diagnostics.AddError("Error reading personalization strategy", err.Error())
	}
}

func (r *personalizationStrategyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != personalizationStrategyID {
		resp.Diagnostics.AddError("Invalid import ID", "expected import ID \"default\"")
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := client.GetPersonalizationStrategy()
	if err != nil {
		resp.Diagnostics.AddError("Error importing personalization strategy", err.Error())
		return
	}

	var state PersonalizationStrategyResourceModel
	resp.Diagnostics.Append(hydratePersonalizationStrategyModel(apiResp, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *personalizationStrategyResource) client() (*api.APIClient, diag.Diagnostics) {
	var diags diag.Diagnostics

	client, err := analyticsregion.NewPersonalizationClient(r.appID, r.apiKey, r.personalizationRegion)
	if err != nil {
		diags.AddError("Unable to create Personalization client", err.Error())
		return nil, diags
	}

	return client, diags
}

func (d *personalizationStrategyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_personalization_strategy"
}

func (d *personalizationStrategyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = personalizationStrategyDataSourceSchema()
}

func (d *personalizationStrategyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*providertypes.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *providertypes.ProviderData, got: %T", req.ProviderData),
		)
		return
	}

	d.appID = data.AppID
	d.apiKey = data.APIKey
	d.personalizationRegion = data.PersonalizationRegion
}

func (d *personalizationStrategyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	client, diags := (&personalizationStrategyResource{
		appID:                 d.appID,
		apiKey:                d.apiKey,
		personalizationRegion: d.personalizationRegion,
	}).client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := client.GetPersonalizationStrategy()
	if err != nil {
		resp.Diagnostics.AddError("Error reading personalization strategy", err.Error())
		return
	}

	var state PersonalizationStrategyDataSourceModel
	resp.Diagnostics.Append(hydratePersonalizationStrategyModel(apiResp, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
