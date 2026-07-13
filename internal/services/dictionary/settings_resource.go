package dictionary

import (
	"context"
	"fmt"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &dictionarySettingsResource{}
	_ resource.ResourceWithConfigure   = &dictionarySettingsResource{}
	_ resource.ResourceWithImportState = &dictionarySettingsResource{}
)

// dictionarySettingsResource manages the application-level singleton
// dictionary settings (disableStandardEntries). Unlike dictionaryEntryResource,
// it needs the application ID (stored at Configure time) to use as the
// Terraform resource ID, since there is no natural per-resource identifier.
type dictionarySettingsResource struct {
	client *search.APIClient
	appID  string
}

// NewSettingsResource returns the algolia_dictionary_settings resource.
func NewSettingsResource() resource.Resource {
	return &dictionarySettingsResource{}
}

func (r *dictionarySettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dictionary_settings"
}

func (r *dictionarySettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = dictionarySettingsResourceSchema()
}

func (r *dictionarySettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = data.Client
	r.appID = data.AppID
}

func (r *dictionarySettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DictionarySettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entries, diags := expandDictionarySettings(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Setting dictionary settings", map[string]any{"app_id": r.appID})

	if err := r.saveDictionarySettings(entries); err != nil {
		resp.Diagnostics.AddError("Error setting dictionary settings", err.Error())
		return
	}

	current, err := r.client.GetDictionarySettings()
	if err != nil {
		resp.Diagnostics.AddError("Error reading dictionary settings", "Could not read back dictionary settings after creation: "+err.Error())
		return
	}

	plan.ID = types.StringValue(r.appID)
	resp.Diagnostics.Append(flattenDictionarySettings(ctx, current.GetDisableStandardEntries(), &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dictionarySettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DictionarySettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.client.GetDictionarySettings()
	if err != nil {
		resp.Diagnostics.AddError("Error reading dictionary settings", err.Error())
		return
	}

	// Dictionary settings are an application-level singleton that always
	// exists (defaults to nothing disabled), so Read never removes the
	// resource from state - it just reflects the current values.
	state.ID = types.StringValue(r.appID)
	resp.Diagnostics.Append(flattenDictionarySettings(ctx, current.GetDisableStandardEntries(), &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dictionarySettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DictionarySettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entries, diags := expandDictionarySettings(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating dictionary settings", map[string]any{"app_id": r.appID})

	if err := r.saveDictionarySettings(entries); err != nil {
		resp.Diagnostics.AddError("Error updating dictionary settings", err.Error())
		return
	}

	current, err := r.client.GetDictionarySettings()
	if err != nil {
		resp.Diagnostics.AddError("Error reading dictionary settings", "Could not read back dictionary settings after update: "+err.Error())
		return
	}

	plan.ID = types.StringValue(r.appID)
	resp.Diagnostics.Append(flattenDictionarySettings(ctx, current.GetDisableStandardEntries(), &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dictionarySettingsResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Dictionary settings are application-level global state: there is
	// nothing to "delete" via the API. Removing the Terraform resource
	// instead resets standard entries to their defaults (nothing disabled),
	// mirroring how algolia_personalization_strategy resets on Delete.
	tflog.Debug(ctx, "Resetting dictionary settings to defaults", map[string]any{"app_id": r.appID})

	if err := r.saveDictionarySettings(search.StandardEntries{}); err != nil {
		resp.Diagnostics.AddError("Error resetting dictionary settings", err.Error())
	}
}

func (r *dictionarySettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Dictionary settings are a singleton: any import ID is accepted (the
	// app ID is the conventional choice) and the current settings are read
	// directly, mirroring algolia_personalization_strategy's import.
	tflog.Debug(ctx, "Importing dictionary settings", map[string]any{"import_id": req.ID, "app_id": r.appID})

	current, err := r.client.GetDictionarySettings()
	if err != nil {
		resp.Diagnostics.AddError("Error importing dictionary settings", err.Error())
		return
	}

	var state DictionarySettingsResourceModel
	state.ID = types.StringValue(r.appID)
	resp.Diagnostics.Append(flattenDictionarySettings(ctx, current.GetDisableStandardEntries(), &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// saveDictionarySettings sets the given StandardEntries and waits for the
// resulting application-level task to complete before returning. Reuses the
// waitForDictionaryTask helper already defined in resource.go for the
// algolia_dictionary_entry resource.
func (r *dictionarySettingsResource) saveDictionarySettings(entries search.StandardEntries) error {
	params := search.NewDictionarySettingsParams(entries)

	updateResp, err := r.client.SetDictionarySettings(r.client.NewApiSetDictionarySettingsRequest(params))
	if err != nil {
		return err
	}

	return waitForDictionaryTask(r.client, updateResp.TaskID)
}
