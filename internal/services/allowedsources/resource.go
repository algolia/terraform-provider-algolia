package allowedsources

import (
	"context"
	"errors"
	"fmt"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &allowedSourcesResource{}
	_ resource.ResourceWithConfigure   = &allowedSourcesResource{}
	_ resource.ResourceWithImportState = &allowedSourcesResource{}
)

// allowedSourcesResource manages the application-level singleton allowlist
// of source IP addresses/ranges permitted to use the Algolia API (the
// "Sources" security setting). Like the dictionary settings resource, it
// needs the application ID (stored at Configure time) to use as the
// Terraform resource ID, since there is no natural per-resource identifier.
type allowedSourcesResource struct {
	client *search.APIClient
	appID  string
}

// NewResource returns the algolia_allowed_sources resource.
func NewResource() resource.Resource {
	return &allowedSourcesResource{}
}

func (r *allowedSourcesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_allowed_sources"
}

func (r *allowedSourcesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = allowedSourcesResourceSchema()
}

func (r *allowedSourcesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *allowedSourcesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AllowedSourcesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sources, diags := expandSources(ctx, plan.Source)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Replacing allowed sources", map[string]any{"app_id": r.appID, "count": len(sources)})

	if err := r.replaceSources(sources); err != nil {
		resp.Diagnostics.AddError("Error setting allowed sources", err.Error())
		return
	}

	current, err := r.client.GetSources()
	if err != nil {
		resp.Diagnostics.AddError("Error reading allowed sources", "Could not read back allowed sources after creation: "+err.Error())
		return
	}

	plan.ID = types.StringValue(r.appID)
	resp.Diagnostics.Append(flattenSources(ctx, current, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *allowedSourcesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AllowedSourcesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.client.GetSources()
	if err != nil {
		resp.Diagnostics.AddError("Error reading allowed sources", err.Error())
		return
	}

	// The allowed sources allowlist is an application-level singleton that
	// always exists (an empty list simply means no IP restrictions are
	// configured), so Read never removes the resource from state - it just
	// reflects the current values.
	state.ID = types.StringValue(r.appID)
	resp.Diagnostics.Append(flattenSources(ctx, current, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *allowedSourcesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AllowedSourcesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sources, diags := expandSources(ctx, plan.Source)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Replacing allowed sources", map[string]any{"app_id": r.appID, "count": len(sources)})

	if err := r.replaceSources(sources); err != nil {
		resp.Diagnostics.AddError("Error updating allowed sources", err.Error())
		return
	}

	current, err := r.client.GetSources()
	if err != nil {
		resp.Diagnostics.AddError("Error reading allowed sources", "Could not read back allowed sources after update: "+err.Error())
		return
	}

	plan.ID = types.StringValue(r.appID)
	resp.Diagnostics.Append(flattenSources(ctx, current, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *allowedSourcesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Allowed sources are application-level global state: there is no "delete"
	// endpoint for the whole list, and a single ReplaceSources([]) call is not
	// an option either because the generated Go client rejects an empty
	// `source` slice client-side ("Parameter `source` is required when calling
	// `ReplaceSources`") before ever making the HTTP request. Each entry is
	// therefore removed individually via DeleteSource.
	//
	// The entries to remove come from state, never from GetSources: enumerating
	// the application's allowlist and deleting all of it would destroy entries
	// this resource never created - for example an address added through the
	// Algolia dashboard after the last apply.
	var state AllowedSourcesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	managed, diags := managedSourceValues(ctx, state.Source)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Clearing managed allowed sources", map[string]any{"app_id": r.appID, "count": len(managed)})

	if err := deleteSources(managed, func(value string) error {
		_, err := r.client.DeleteSource(r.client.NewApiDeleteSourceRequest(value))
		return err
	}); err != nil {
		resp.Diagnostics.AddError("Error clearing allowed sources", err.Error())
	}
}

func (r *allowedSourcesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Allowed sources are a singleton: any import ID is accepted (the app ID
	// is the conventional choice) and the current sources are read directly,
	// mirroring algolia_dictionary_settings' import.
	tflog.Debug(ctx, "Importing allowed sources", map[string]any{"import_id": req.ID, "app_id": r.appID})

	current, err := r.client.GetSources()
	if err != nil {
		resp.Diagnostics.AddError("Error importing allowed sources", err.Error())
		return
	}

	var state AllowedSourcesResourceModel
	state.ID = types.StringValue(r.appID)
	resp.Diagnostics.Append(flattenSources(ctx, current, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// replaceSources calls ReplaceSources with the given full list. There is no
// task to wait on here: unlike SetSettings/SetDictionarySettings,
// ReplaceSources takes effect immediately and its response only carries an
// updatedAt timestamp.
func (r *allowedSourcesResource) replaceSources(sources []search.Source) error {
	_, err := r.client.ReplaceSources(r.client.NewApiReplaceSourcesRequest(sources))
	return err
}

// deleteSources removes exactly the given source values, one call per value, and
// stops at the first real failure. An entry that is already gone is not a
// failure: a destroy has to stay idempotent, whether the entry was removed out
// of band or by an earlier destroy that failed part-way through this loop.
func deleteSources(values []string, deleteSource func(value string) error) error {
	for _, value := range values {
		if err := deleteSource(value); err != nil && !isSourceNotFound(err) {
			return fmt.Errorf("could not delete source %q: %w", value, err)
		}
	}

	return nil
}

func isSourceNotFound(err error) bool {
	var apiErr *search.APIError

	return errors.As(err, &apiErr) && apiErr.Status == 404
}
