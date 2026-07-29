package apikey

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestAPIKeyResourceCreate_persistsKeyBeforeFailingReadBack pins the guarantee
// that makes a failed create recoverable: AddApiKey is the only call that ever
// returns the key value, and that value is both the credential and this
// resource's id. If Create returned without writing it to state, the key would
// exist in Algolia with real ACLs and Terraform could never read, rotate or
// delete it.
//
// The fake host below accepts the create and then fails every GET, which is the
// shape of a propagation failure: the wait for the key to become readable and
// the read-back that follows both go through GetApiKey.
func TestAPIKeyResourceCreate_persistsKeyBeforeFailingReadBack(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"key":"` + secretKey + `","createdAt":"2024-01-01T00:00:00Z"}`))
			return
		}

		// 4xx is not retried by the v4 transport, so the read fails on the
		// first attempt instead of exhausting the propagation retry budget.
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"key is not readable yet","status":400}`))
	}))
	defer server.Close()

	r := &apiKeyResource{client: newTestSearchClient(t, server), now: time.Now}
	resp := &resource.CreateResponse{State: emptyResourceState(t)}
	r.Create(ctx, resource.CreateRequest{Plan: apiKeyCreatePlan(t)}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Create() reported no error although every read-back failed; this test is no longer exercising the failure path")
	}
	if resp.State.Raw.IsNull() {
		t.Fatal("Create() left state empty after AddApiKey succeeded: the key exists in Algolia, its value can never be recovered, and Terraform cannot read, rotate or delete it")
	}
	if !resp.State.Raw.IsFullyKnown() {
		t.Fatalf("Create() persisted unknown values, which Terraform rejects as an apply result: %s", resp.State.Raw)
	}

	var got APIKeyResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading back the persisted state: %v", diags)
	}

	if got.ID.ValueString() != secretKey {
		t.Errorf("persisted id = %q, want the created key value", got.ID.ValueString())
	}
	if !got.CreatedAt.IsNull() {
		t.Errorf("persisted created_at = %v, want null: it is only knowable from the read-back that failed", got.CreatedAt)
	}
	if got.Description.ValueString() != "tf-acc-test-orphan-guard" {
		t.Errorf("persisted description = %q, want the planned value", got.Description.ValueString())
	}
	if elements := got.ACL.Elements(); len(elements) != 1 {
		t.Errorf("persisted acl = %v, want the planned single-element set", got.ACL)
	}
}

// apiKeyCreatePlan builds the plan Terraform hands to Create: the two Computed
// attributes (id, created_at) are unknown, and everything else holds the
// configuration verbatim.
func apiKeyCreatePlan(t *testing.T) tfsdk.Plan {
	t.Helper()

	ctx := context.Background()
	keySchema := apiKeyResourceSchema()
	plan := tfsdk.Plan{
		Raw:    tftypes.NewValue(keySchema.Type().TerraformType(ctx), nil),
		Schema: keySchema,
	}

	diags := plan.Set(ctx, &APIKeyResourceModel{
		ID:                     types.StringUnknown(),
		ACL:                    types.SetValueMust(types.StringType, []attr.Value{types.StringValue("search")}),
		Description:            types.StringValue("tf-acc-test-orphan-guard"),
		ExpiresAt:              types.StringNull(),
		Indexes:                types.SetNull(types.StringType),
		Referers:               types.SetNull(types.StringType),
		MaxHitsPerQuery:        types.Int64Null(),
		MaxQueriesPerIPPerHour: types.Int64Null(),
		QueryParameters:        types.StringNull(),
		CreatedAt:              types.StringUnknown(),
	})
	if diags.HasError() {
		t.Fatalf("could not build the create plan: %v", diags)
	}

	return plan
}
