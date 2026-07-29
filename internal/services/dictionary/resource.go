package dictionary

import (
	"context"
	"fmt"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"
	"github.com/algolia/terraform-provider-algolia/internal/algoliawait"
	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &dictionaryEntryResource{}
	_ resource.ResourceWithConfigure   = &dictionaryEntryResource{}
	_ resource.ResourceWithImportState = &dictionaryEntryResource{}
)

type dictionaryEntryResource struct {
	client *search.APIClient
}

func NewResource() resource.Resource {
	return &dictionaryEntryResource{}
}

func (r *dictionaryEntryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dictionary_entry"
}

func (r *dictionaryEntryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = dictionaryEntryResourceSchema()
}

func (r *dictionaryEntryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
}

func (r *dictionaryEntryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DictionaryEntryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dictionaryType := search.DictionaryType(plan.Dictionary.ValueString())

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

	tflog.Debug(ctx, "Creating dictionary entry", map[string]any{"dictionary": string(dictionaryType), "object_id": objectID})

	entry, diags := expandDictionaryEntry(objectID, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.saveDictionaryEntry(ctx, dictionaryType, entry); err != nil {
		resp.Diagnostics.AddError("Error creating dictionary entry", "Could not create entry "+objectID+" in dictionary "+string(dictionaryType)+": "+err.Error())
		return
	}

	// The entry now exists in Algolia. Persist the identifying attributes
	// immediately, before waiting for it to become searchable, so that a failure
	// in that wait surfaces as an error on a resource Terraform knows about
	// instead of orphaning an entry that exists remotely but not in state. That
	// matters most here because object_id is generated: without this write a
	// retry would generate a *new* object_id and leave the previous entry behind
	// undeleted, whereas a recorded object_id is reused (it is Computed with
	// UseStateForUnknown) and the same entry is retried.
	//
	// Terraform rejects an apply result that still contains unknown values, so
	// every unknown has to be resolved here. id is derived from values already in
	// hand; state is Optional+Computed and therefore unknown when the
	// configuration omits it, but its value is already decided - expand has just
	// applied the stopwords default, and flatten maps a missing state to null for
	// the other dictionaries. Every other attribute is Optional-only, so the plan
	// holds the configuration verbatim and is already known.
	plan.ID = types.StringValue(dictionaryEntryResourceID(string(dictionaryType), objectID))
	if entry.State != nil {
		plan.State = types.StringValue(string(*entry.State))
	} else {
		plan.State = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fetched, err := waitForDictionaryEntry(ctx, r.client, dictionaryType, objectID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading dictionary entry", "Could not read back entry "+objectID+" in dictionary "+string(dictionaryType)+" after creation: "+err.Error())
		return
	}

	resp.Diagnostics.Append(flattenDictionaryEntry(dictionaryType, fetched, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dictionaryEntryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DictionaryEntryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dictionaryType := search.DictionaryType(state.Dictionary.ValueString())
	objectID := state.ObjectID.ValueString()

	entry, err := findDictionaryEntry(ctx, r.client, dictionaryType, objectID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading dictionary entry", "Could not read entry "+objectID+" in dictionary "+string(dictionaryType)+": "+err.Error())
		return
	}

	if entry == nil {
		tflog.Warn(ctx, "Dictionary entry not found; removing from state", map[string]any{"dictionary": string(dictionaryType), "object_id": objectID})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(flattenDictionaryEntry(dictionaryType, entry, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dictionaryEntryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DictionaryEntryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dictionaryType := search.DictionaryType(plan.Dictionary.ValueString())
	objectID := plan.ObjectID.ValueString()
	tflog.Debug(ctx, "Updating dictionary entry", map[string]any{"dictionary": string(dictionaryType), "object_id": objectID})

	entry, diags := expandDictionaryEntry(objectID, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.saveDictionaryEntry(ctx, dictionaryType, entry); err != nil {
		resp.Diagnostics.AddError("Error updating dictionary entry", "Could not update entry "+objectID+" in dictionary "+string(dictionaryType)+": "+err.Error())
		return
	}

	// The write succeeded, so the remote entry already matches the plan. Persist
	// it before the read-back for the same reason as in Create: a failure in the
	// wait below must not leave state describing the superseded entry. Nothing is
	// unknown here - id and state both resolve through UseStateForUnknown from
	// the prior state during an update.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fetched, err := waitForDictionaryEntry(ctx, r.client, dictionaryType, objectID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading dictionary entry", "Could not read back entry "+objectID+" in dictionary "+string(dictionaryType)+" after update: "+err.Error())
		return
	}

	resp.Diagnostics.Append(flattenDictionaryEntry(dictionaryType, fetched, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dictionaryEntryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DictionaryEntryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dictionaryType := search.DictionaryType(state.Dictionary.ValueString())
	objectID := state.ObjectID.ValueString()

	deleteReq := search.NewBatchDictionaryEntriesRequest(search.DICTIONARY_ACTION_DELETE_ENTRY, *search.NewDictionaryEntry(objectID))
	params := search.NewBatchDictionaryEntriesParams([]search.BatchDictionaryEntriesRequest{*deleteReq})

	updateResp, err := r.client.BatchDictionaryEntries(r.client.NewApiBatchDictionaryEntriesRequest(dictionaryType, params), search.WithContext(ctx))
	if err != nil {
		if algoliaerr.IsNotFound(err) {
			return
		}

		resp.Diagnostics.AddError("Error deleting dictionary entry", "Could not delete entry "+objectID+" in dictionary "+string(dictionaryType)+": "+err.Error())
		return
	}

	if err := waitForDictionaryTask(ctx, r.client, updateResp.TaskID); err != nil {
		if algoliaerr.IsNotFound(err) {
			return
		}

		resp.Diagnostics.AddError("Error waiting for dictionary entry deletion", "Could not confirm entry deletion: "+err.Error())
	}
}

func (r *dictionaryEntryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	dictionary, objectID, err := parseDictionaryEntryImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	dictionaryType := search.DictionaryType(dictionary)

	entry, err := findDictionaryEntry(ctx, r.client, dictionaryType, objectID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing dictionary entry", "Could not import entry "+req.ID+": "+err.Error())
		return
	}

	if entry == nil {
		resp.Diagnostics.AddError("Error importing dictionary entry", "No entry with object_id "+objectID+" was found in dictionary "+dictionary+".")
		return
	}

	var state DictionaryEntryResourceModel
	resp.Diagnostics.Append(flattenDictionaryEntry(dictionaryType, entry, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// saveDictionaryEntry upserts a single dictionary entry (the API treats
// addEntry as add-or-replace by objectID) and waits for the resulting task
// to complete.
func (r *dictionaryEntryResource) saveDictionaryEntry(ctx context.Context, dictionaryType search.DictionaryType, entry *search.DictionaryEntry) error {
	addReq := search.NewBatchDictionaryEntriesRequest(search.DICTIONARY_ACTION_ADD_ENTRY, *entry)
	params := search.NewBatchDictionaryEntriesParams([]search.BatchDictionaryEntriesRequest{*addReq})

	updateResp, err := r.client.BatchDictionaryEntries(r.client.NewApiBatchDictionaryEntriesRequest(dictionaryType, params), search.WithContext(ctx))
	if err != nil {
		return err
	}

	return waitForDictionaryTask(ctx, r.client, updateResp.TaskID)
}

// findDictionaryEntry pages through SearchDictionaryEntries looking for the
// hit whose ObjectID matches. There is no get-by-id endpoint for dictionary
// entries, so reads and deletes-confirmation rely on this search. A nil,
// nil return means the entry does not exist (treated like a 404).
func findDictionaryEntry(ctx context.Context, client *search.APIClient, dictionaryType search.DictionaryType, objectID string) (*search.DictionaryEntry, error) {
	const hitsPerPage = int32(1000)
	page := int32(0)

	for {
		params := search.NewSearchDictionaryEntriesParams(
			"",
			search.WithSearchDictionaryEntriesParamsPage(page),
			search.WithSearchDictionaryEntriesParamsHitsPerPage(hitsPerPage),
		)

		apiResp, err := client.SearchDictionaryEntries(client.NewApiSearchDictionaryEntriesRequest(dictionaryType, params), search.WithContext(ctx))
		if err != nil {
			return nil, err
		}

		for i := range apiResp.Hits {
			if apiResp.Hits[i].GetObjectID() == objectID {
				return &apiResp.Hits[i], nil
			}
		}

		page++
		if len(apiResp.Hits) == 0 || page >= apiResp.NbPages {
			return nil, nil
		}
	}
}

// waitForDictionaryEntry polls findDictionaryEntry until the entry becomes
// visible or the retry budget is exhausted. The task-completion wait
// (waitForDictionaryTask) only confirms the write was applied; the search
// index backing SearchDictionaryEntries can lag slightly behind.
func waitForDictionaryEntry(ctx context.Context, client *search.APIClient, dictionaryType search.DictionaryType, objectID string) (*search.DictionaryEntry, error) {
	return search.CreateIterable(
		func(*search.DictionaryEntry, error) (*search.DictionaryEntry, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}

			return findDictionaryEntry(ctx, client, dictionaryType, objectID)
		},
		func(entry *search.DictionaryEntry, err error) (bool, error) {
			if err != nil {
				return false, err
			}
			return entry != nil, nil
		},
		search.WithTimeout(func(count int) time.Duration {
			return time.Duration(min(200*count, 2000)) * time.Millisecond
		}),
		search.WithMaxRetries(30),
	)
}

// waitForDictionaryTask polls GetAppTask until the task reaches "published"
// status. This replaces the SDK's built-in WaitForAppTask whose retry-count
// options were not being applied. Dictionary batch tasks are
// application-level, so this uses GetAppTask rather than the per-index GetTask.
func waitForDictionaryTask(ctx context.Context, client *search.APIClient, taskID int64) error {
	return algoliawait.Until(ctx, fmt.Sprintf("app task %d", taskID), func(ctx context.Context) (bool, error) {
		resp, err := client.GetAppTask(client.NewApiGetAppTaskRequest(taskID), search.WithContext(ctx))
		if err != nil {
			return false, err
		}

		return resp.Status == search.TASK_STATUS_PUBLISHED, nil
	})
}
