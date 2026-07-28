package ingestion

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/call"
	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/transport"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Every Ingestion object is created with a server-assigned UUID that only the
// create response carries, and each Create then reads the object back before
// writing state. These tests drive the five create* bodies against a fake host
// that accepts the create and fails the read-back, and assert that the identity
// the API just assigned was persisted anyway. Without that early write the
// object exists in Algolia with no Terraform record: the next apply creates a
// duplicate and nothing ever adopts the first one.
//
// They also assert the persisted state is fully known, because Terraform rejects
// an apply result that still contains unknown values - which is the trap in
// writing state before the read-back has resolved the Computed attributes.

const testCreatedUUID = "11111111-2222-3333-4444-555555555555"

func TestCreateAuthentication_persistsIDBeforeFailingReadBack(t *testing.T) {
	ctx := context.Background()
	client := newTestIngestionClient(t, createThenFailHandler(`{"authenticationID":"`+testCreatedUUID+`","name":"tf-test-auth","createdAt":"2024-01-01T00:00:00Z"}`))

	plan := AuthenticationResourceModel{
		ID:               types.StringUnknown(),
		AuthenticationID: types.StringUnknown(),
		Type:             types.StringValue("algolia"),
		Name:             types.StringValue("tf-test-auth"),
		Platform:         types.StringNull(),
		Input:            types.StringValue(`{"appID":"app","apiKey":"key"}`),
		CreatedAt:        types.StringUnknown(),
		UpdatedAt:        types.StringUnknown(),
	}

	resp := &resource.CreateResponse{State: emptyState(t, authenticationResourceSchema())}
	createAuthentication(ctx, client, &plan, resp)

	assertOrphanGuard(t, resp, "authentication")

	var got AuthenticationResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading back the persisted state: %v", diags)
	}
	if got.ID.ValueString() != testCreatedUUID || got.AuthenticationID.ValueString() != testCreatedUUID {
		t.Errorf("persisted id/authentication_id = %q/%q, want %q", got.ID.ValueString(), got.AuthenticationID.ValueString(), testCreatedUUID)
	}
	if !got.CreatedAt.IsNull() || !got.UpdatedAt.IsNull() {
		t.Errorf("persisted created_at/updated_at = %v/%v, want null: both are knowable only from the read-back that failed", got.CreatedAt, got.UpdatedAt)
	}
	if got.Input.ValueString() != `{"appID":"app","apiKey":"key"}` {
		t.Errorf("persisted input = %q, want the configured credentials", got.Input.ValueString())
	}
}

func TestCreateSource_persistsIDBeforeFailingReadBack(t *testing.T) {
	ctx := context.Background()
	client := newTestIngestionClient(t, createThenFailHandler(`{"sourceID":"`+testCreatedUUID+`","name":"tf-test-source","createdAt":"2024-01-01T00:00:00Z"}`))

	plan := SourceResourceModel{
		ID:               types.StringUnknown(),
		SourceID:         types.StringUnknown(),
		Type:             types.StringValue("push"),
		Name:             types.StringValue("tf-test-source"),
		Input:            types.StringNull(),
		AuthenticationID: types.StringNull(),
		CreatedAt:        types.StringUnknown(),
		UpdatedAt:        types.StringUnknown(),
	}

	resp := &resource.CreateResponse{State: emptyState(t, sourceResourceSchema())}
	createSource(ctx, client, &plan, resp)

	assertOrphanGuard(t, resp, "source")

	var got SourceResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading back the persisted state: %v", diags)
	}
	if got.ID.ValueString() != testCreatedUUID || got.SourceID.ValueString() != testCreatedUUID {
		t.Errorf("persisted id/source_id = %q/%q, want %q", got.ID.ValueString(), got.SourceID.ValueString(), testCreatedUUID)
	}
	if !got.CreatedAt.IsNull() || !got.UpdatedAt.IsNull() {
		t.Errorf("persisted created_at/updated_at = %v/%v, want null", got.CreatedAt, got.UpdatedAt)
	}
}

func TestCreateDestination_persistsIDBeforeFailingReadBack(t *testing.T) {
	ctx := context.Background()
	client := newTestIngestionClient(t, createThenFailHandler(`{"destinationID":"`+testCreatedUUID+`","name":"tf-test-destination","createdAt":"2024-01-01T00:00:00Z"}`))

	plan := DestinationResourceModel{
		ID:                types.StringUnknown(),
		DestinationID:     types.StringUnknown(),
		Type:              types.StringValue("search"),
		Name:              types.StringValue("tf-test-destination"),
		Input:             types.StringValue(`{"indexName":"tf-test-index"}`),
		AuthenticationID:  types.StringNull(),
		TransformationIDs: types.ListNull(types.StringType),
		CreatedAt:         types.StringUnknown(),
		UpdatedAt:         types.StringUnknown(),
	}

	resp := &resource.CreateResponse{State: emptyState(t, destinationResourceSchema())}
	createDestination(ctx, client, &plan, resp)

	assertOrphanGuard(t, resp, "destination")

	var got DestinationResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading back the persisted state: %v", diags)
	}
	if got.ID.ValueString() != testCreatedUUID || got.DestinationID.ValueString() != testCreatedUUID {
		t.Errorf("persisted id/destination_id = %q/%q, want %q", got.ID.ValueString(), got.DestinationID.ValueString(), testCreatedUUID)
	}
	// transformation_ids is the model's only non-scalar attribute: it has to
	// survive as a *typed* null list, since a zero-value types.List carries no
	// element type and the framework rejects it at conversion time.
	if !got.TransformationIDs.IsNull() {
		t.Errorf("persisted transformation_ids = %v, want the planned null list", got.TransformationIDs)
	}
	if got.TransformationIDs.ElementType(ctx) != types.StringType {
		t.Errorf("persisted transformation_ids element type = %v, want types.StringType", got.TransformationIDs.ElementType(ctx))
	}
}

func TestCreateTransformation_persistsIDBeforeFailingReadBack(t *testing.T) {
	ctx := context.Background()
	client := newTestIngestionClient(t, createThenFailHandler(`{"transformationID":"`+testCreatedUUID+`","createdAt":"2024-01-01T00:00:00Z"}`))

	// The realistic plan for a transformation whose logic is configured through
	// `input`: `code` is Optional+Computed with no UseStateForUnknown, so it is
	// unknown until the API derives it.
	plan := TransformationResourceModel{
		ID:                types.StringUnknown(),
		TransformationID:  types.StringUnknown(),
		Name:              types.StringValue("tf-test-transformation"),
		Code:              types.StringUnknown(),
		Type:              types.StringValue("code"),
		Input:             types.StringValue(`{"code":"return record"}`),
		Description:       types.StringNull(),
		AuthenticationIDs: types.ListNull(types.StringType),
		CreatedAt:         types.StringUnknown(),
		UpdatedAt:         types.StringUnknown(),
	}

	resp := &resource.CreateResponse{State: emptyState(t, transformationResourceSchema())}
	createTransformation(ctx, client, &plan, resp)

	assertOrphanGuard(t, resp, "transformation")

	var got TransformationResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading back the persisted state: %v", diags)
	}
	if got.ID.ValueString() != testCreatedUUID || got.TransformationID.ValueString() != testCreatedUUID {
		t.Errorf("persisted id/transformation_id = %q/%q, want %q", got.ID.ValueString(), got.TransformationID.ValueString(), testCreatedUUID)
	}
	if !got.Code.IsNull() {
		t.Errorf("persisted code = %v, want null: it is Optional+Computed and only the read-back knows what the API derived", got.Code)
	}
	if got.AuthenticationIDs.ElementType(ctx) != types.StringType {
		t.Errorf("persisted authentication_ids element type = %v, want types.StringType", got.AuthenticationIDs.ElementType(ctx))
	}
}

func TestCreateTask_persistsIDBeforeFailingReadBack(t *testing.T) {
	ctx := context.Background()
	client := newTestIngestionClient(t, createThenFailHandler(`{"taskID":"`+testCreatedUUID+`","createdAt":"2024-01-01T00:00:00Z"}`))

	// failure_threshold, notifications and policies are Optional+Computed
	// because the API substitutes its own defaults, so they are unknown in the
	// plan when the configuration omits them; enabled is Optional+Computed too
	// but carries a static default, so it is known.
	plan := TaskResourceModel{
		ID:                 types.StringUnknown(),
		TaskID:             types.StringUnknown(),
		SourceID:           types.StringValue("source-uuid"),
		DestinationID:      types.StringValue("destination-uuid"),
		Action:             types.StringValue("replace"),
		SubscriptionAction: types.StringNull(),
		Cron:               types.StringNull(),
		Enabled:            types.BoolValue(true),
		FailureThreshold:   types.Int64Unknown(),
		Input:              types.StringNull(),
		Notifications:      types.StringUnknown(),
		Policies:           types.StringUnknown(),
		Cursor:             types.StringNull(),
		CreatedAt:          types.StringUnknown(),
		UpdatedAt:          types.StringUnknown(),
		LastRun:            types.StringUnknown(),
		NextRun:            types.StringUnknown(),
	}

	resp := &resource.CreateResponse{State: emptyState(t, taskResourceSchema())}
	createTask(ctx, client, &plan, resp)

	assertOrphanGuard(t, resp, "task")

	var got TaskResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading back the persisted state: %v", diags)
	}
	if got.ID.ValueString() != testCreatedUUID || got.TaskID.ValueString() != testCreatedUUID {
		t.Errorf("persisted id/task_id = %q/%q, want %q", got.ID.ValueString(), got.TaskID.ValueString(), testCreatedUUID)
	}
	if !got.FailureThreshold.IsNull() || !got.Notifications.IsNull() || !got.Policies.IsNull() {
		t.Errorf("persisted failure_threshold/notifications/policies = %v/%v/%v, want null: all three are resolved by the API and only the read-back knows them", got.FailureThreshold, got.Notifications, got.Policies)
	}
	if !got.LastRun.IsNull() || !got.NextRun.IsNull() {
		t.Errorf("persisted last_run/next_run = %v/%v, want null", got.LastRun, got.NextRun)
	}
	if !got.Enabled.ValueBool() {
		t.Error("persisted enabled = false, want the planned value true")
	}
	if got.Action.ValueString() != "replace" {
		t.Errorf("persisted action = %q, want the planned value", got.Action.ValueString())
	}
}

// assertOrphanGuard checks the invariant shared by all five resources: the
// create succeeded, the read-back failed, and state still records the object.
func assertOrphanGuard(t *testing.T, resp *resource.CreateResponse, kind string) {
	t.Helper()

	if !resp.Diagnostics.HasError() {
		t.Fatalf("create%s reported no error although the read-back failed; this test is no longer exercising the failure path", kind)
	}
	if resp.State.Raw.IsNull() {
		t.Fatalf("create%s left state empty after the %s was created: it exists in Algolia with no Terraform record, so the next apply creates a duplicate and nothing ever adopts the orphan", kind, kind)
	}
	if !resp.State.Raw.IsFullyKnown() {
		t.Fatalf("create%s persisted unknown values, which Terraform rejects as an apply result: %s", kind, resp.State.Raw)
	}
}

// createThenFailHandler answers the create call with the given body and fails
// every subsequent read, the shape of a create that succeeds remotely and a
// read-back that does not. 4xx is not retried by the v4 transport, so the read
// fails on the first attempt.
func createThenFailHandler(createBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(createBody))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"cannot read the object back","status":400}`))
	}
}

// newTestIngestionClient returns an Ingestion client whose only host is a test
// server running the given handler. Setting Hosts explicitly is what makes the
// client skip its region-derived defaults, so no analytics region is involved.
func newTestIngestionClient(t *testing.T, handler http.HandlerFunc) *ingestionapi.APIClient {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := ingestionapi.NewClientWithConfig(ingestionapi.IngestionConfiguration{
		Configuration: transport.Configuration{
			AppID:  "test-app",
			ApiKey: "test-key",
			Hosts: []transport.StatefulHost{
				transport.NewStatefulHost("http", server.Listener.Addr().String(), func(call.Kind) bool { return true }),
			},
		},
	})
	if err != nil {
		t.Fatalf("could not build test Ingestion client: %v", err)
	}

	return client
}

// emptyState mirrors how the framework hands Create its response state: a null
// object of the resource's own schema (see fwserver.CreateResource).
func emptyState(t *testing.T, resourceSchema schema.Schema) tfsdk.State {
	t.Helper()

	return tfsdk.State{
		Raw:    tftypes.NewValue(resourceSchema.Type().TerraformType(context.Background()), nil),
		Schema: resourceSchema,
	}
}
