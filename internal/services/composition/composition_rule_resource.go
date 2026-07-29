package composition

import (
	"context"

	compositionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/composition"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &compositionRuleResource{}
	_ resource.ResourceWithConfigure   = &compositionRuleResource{}
	_ resource.ResourceWithImportState = &compositionRuleResource{}
)

// compositionRuleResource manages an algolia_composition_rule resource. It
// embeds base (see client.go) for Configure-time wiring and an on-demand
// Composition client - the Composition API is not region-routed, so base
// here only needs appID/apiKey.
type compositionRuleResource struct {
	base
}

// compositionRuleKind names this resource inside a diagnostic sentence.
const compositionRuleKind = "composition rule"

// compositionRuleSubject identifies the composition rule a diagnostic is about.
// Composition rules are scoped to a composition rather than to an index, so both
// parts are needed to name one unambiguously.
func compositionRuleSubject(compositionID, objectID string) algoliaerr.Subject {
	return algoliaerr.Object(compositionRuleKind, objectID).In("composition", compositionID)
}

// NewRuleResource returns the algolia_composition_rule resource.
func NewRuleResource() resource.Resource {
	return &compositionRuleResource{}
}

func (r *compositionRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_composition_rule"
}

func (r *compositionRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = compositionRuleResourceSchema()
}

func (r *compositionRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(r.configure(req.ProviderData)...)
}

func (r *compositionRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CompositionRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	compositionID := plan.CompositionID.ValueString()

	objectID := plan.ObjectID.ValueString()
	if plan.ObjectID.IsNull() || plan.ObjectID.IsUnknown() || objectID == "" {
		generated, err := generateObjectID()
		if err != nil {
			resp.Diagnostics.AddError("Error generating object_id", "Could not generate a random object_id: "+err.Error())
			return
		}
		objectID = generated
	}
	plan.ObjectID = types.StringValue(objectID)

	tflog.Debug(ctx, "Creating composition rule", map[string]any{"composition_id": compositionID, "object_id": objectID})

	rule, expandDiags := expandCompositionRule(objectID, &plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	putResp, err := client.PutCompositionRule(client.NewApiPutCompositionRuleRequest(compositionID, objectID, rule), compositionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(compositionRuleSubject(compositionID, objectID).Message(algoliaerr.Create, err))
		return
	}

	if err := waitForCompositionTask(ctx, client, compositionID, putResp.TaskID); err != nil {
		resp.Diagnostics.AddError(algoliaerr.WaitMessage(compositionRuleKind, algoliaerr.Create, err))
		return
	}

	apiResp, err := client.GetRule(client.NewApiGetRuleRequest(compositionID, objectID), compositionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(compositionRuleSubject(compositionID, objectID).Message(algoliaerr.Read, err))
		return
	}

	resp.Diagnostics.Append(flattenCompositionRule(compositionID, apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *compositionRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CompositionRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	compositionID := state.CompositionID.ValueString()
	objectID := state.ObjectID.ValueString()

	apiResp, err := client.GetRule(client.NewApiGetRuleRequest(compositionID, objectID), compositionapi.WithContext(ctx))
	if err != nil {
		if algoliaerr.IsNotFound(err) {
			tflog.Warn(ctx, "Composition rule not found; removing from state", map[string]any{"composition_id": compositionID, "object_id": objectID})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(compositionRuleSubject(compositionID, objectID).Message(algoliaerr.Read, err))
		return
	}

	resp.Diagnostics.Append(flattenCompositionRule(compositionID, apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *compositionRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CompositionRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	compositionID := plan.CompositionID.ValueString()
	objectID := plan.ObjectID.ValueString()

	tflog.Debug(ctx, "Updating composition rule", map[string]any{"composition_id": compositionID, "object_id": objectID})

	rule, expandDiags := expandCompositionRule(objectID, &plan)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	putResp, err := client.PutCompositionRule(client.NewApiPutCompositionRuleRequest(compositionID, objectID, rule), compositionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(compositionRuleSubject(compositionID, objectID).Message(algoliaerr.Update, err))
		return
	}

	if err := waitForCompositionTask(ctx, client, compositionID, putResp.TaskID); err != nil {
		resp.Diagnostics.AddError(algoliaerr.WaitMessage(compositionRuleKind, algoliaerr.Update, err))
		return
	}

	apiResp, err := client.GetRule(client.NewApiGetRuleRequest(compositionID, objectID), compositionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(compositionRuleSubject(compositionID, objectID).Message(algoliaerr.Read, err))
		return
	}

	resp.Diagnostics.Append(flattenCompositionRule(compositionID, apiResp, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *compositionRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CompositionRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	compositionID := state.CompositionID.ValueString()
	objectID := state.ObjectID.ValueString()

	tflog.Debug(ctx, "Deleting composition rule", map[string]any{"composition_id": compositionID, "object_id": objectID})

	deleteResp, err := client.DeleteCompositionRule(client.NewApiDeleteCompositionRuleRequest(compositionID, objectID), compositionapi.WithContext(ctx))
	if err != nil {
		if algoliaerr.IsNotFound(err) {
			return
		}

		resp.Diagnostics.AddError(compositionRuleSubject(compositionID, objectID).Message(algoliaerr.Delete, err))
		return
	}

	if err := waitForCompositionTask(ctx, client, compositionID, deleteResp.TaskID); err != nil {
		if algoliaerr.IsNotFound(err) {
			return
		}

		resp.Diagnostics.AddError(algoliaerr.WaitMessage(compositionRuleKind, algoliaerr.Delete, err))
	}
}

func (r *compositionRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	compositionID, objectID, err := parseCompositionRuleImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	client, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := client.GetRule(client.NewApiGetRuleRequest(compositionID, objectID), compositionapi.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError(algoliaerr.Object(compositionRuleKind, req.ID).Message(algoliaerr.Import, err))
		return
	}

	var state CompositionRuleResourceModel
	resp.Diagnostics.Append(flattenCompositionRule(compositionID, apiResp, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
