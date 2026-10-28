package rule

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestUpgradeRuleStateV0(t *testing.T) {
	for _, promote := range []string{
		`[{"object_ids":["shoe-12","shoe-21","shoe-30"],"position":0},{"object_ids":["other"],"position":5}]`,
		`[]`,
		`null`,
	} {
		t.Run(promote, func(t *testing.T) {
			ctx := context.Background()
			oldJSON := []byte(`{"id":"products/pins","index_name":"products","object_id":"pins","description":"Keep me","enabled":false,"tags":["tag"],"scope":null,"conditions":[{"pattern":"shoes","anchoring":"is","alternatives":null,"context":null,"filters":null}],"consequence":[{"params_json":"{\"hitsPerPage\":10}","hide":["hidden"],"user_data":null,"filter_promotes":true,"redirect_index_name":null,"promote":` + promote + `}],"validity":[]}`)
			resp := resource.UpgradeStateResponse{State: tfsdk.State{Schema: ruleResourceSchema()}}
			(&ruleResource{}).UpgradeState(ctx)[0].StateUpgrader(ctx, resource.UpgradeStateRequest{
				RawState: &tfprotov6.RawState{JSON: oldJSON},
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			expected, err := (&tfprotov6.RawState{JSON: oldJSON}).Unmarshal(ruleResourceSchema().Type().TerraformType(ctx))
			if err != nil {
				t.Fatal(err)
			}
			if !resp.State.Raw.Equal(expected) {
				t.Fatalf("upgrade changed stored values: %s", resp.State.Raw)
			}
			var model RuleResourceModel
			if diags := resp.State.Get(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			// Refresh replaces the old set's arbitrary order with the API order.
			assertPromoteOrderRoundTrip(t, &model)
		})
	}
}

func TestRulePromoteOrderOnImport(t *testing.T) {
	assertPromoteOrderRoundTrip(t, &RuleResourceModel{})
}

func assertPromoteOrderRoundTrip(t *testing.T, model *RuleResourceModel) {
	t.Helper()
	payload := []byte(`{"objectID":"pins","consequence":{"promote":[{"objectIDs":["shoe-30","shoe-12","shoe-21"],"position":0},{"objectID":"other","position":5}]}}`)
	rule, err := decodeRuleResponse(payload)
	if err != nil {
		t.Fatal(err)
	}
	if diags := hydrateRuleModel("products", rule, nil, model); diags.HasError() {
		t.Fatal(diags)
	}
	promote := consequenceAttr(t, model.Consequence, "promote").(types.List)
	ids := promote.Elements()[0].(types.Object).Attributes()["object_ids"].(types.List)
	want := []string{"shoe-30", "shoe-12", "shoe-21"}
	if got := expandStringList(ids); !reflect.DeepEqual(got, want) {
		t.Fatalf("refreshed order = %v, want %v", got, want)
	}
	if ids.Equal(flattenStringList([]string{"shoe-12", "shoe-21", "shoe-30"})) {
		t.Fatal("reordering must change the Terraform value")
	}
	model.Description = types.StringValue("Unrelated update")
	body, diags := ruleRequestBodyFromModel(model)
	if diags.HasError() {
		t.Fatal(diags)
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	written, err := decodeRuleResponse(encoded)
	if err != nil {
		t.Fatal(err)
	}
	// The singular API form becomes a one-element group, retaining its position.
	if got := written.Consequence.Promote; len(got) != 2 ||
		!reflect.DeepEqual(got[0].PromoteObjectIDs.ObjectIDs, want) ||
		!reflect.DeepEqual(got[1].PromoteObjectIDs.ObjectIDs, []string{"other"}) ||
		got[1].PromoteObjectIDs.Position != 5 {
		t.Fatalf("write changed promote groups: %s", encoded)
	}
}

func TestUpgradeRuleStateV0Invalid(t *testing.T) {
	resp := resource.UpgradeStateResponse{State: tfsdk.State{Schema: ruleResourceSchema()}}
	upgradeRuleStateV0(context.Background(), resource.UpgradeStateRequest{
		RawState: &tfprotov6.RawState{JSON: []byte(`{"consequence":"invalid"}`)},
	}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid state to fail")
	}
}

// Check the published schema itself, including the data source which shares the
// same hydrate path but cannot use a resource state upgrader.
func TestRulePromoteSchemaIsOrdered(t *testing.T) {
	ctx := context.Background()
	for name, typ := range map[string]tftypes.Type{
		"resource":    ruleResourceSchema().Type().TerraformType(ctx),
		"data source": ruleDataSourceSchema().Type().TerraformType(ctx),
	} {
		consequence := typ.(tftypes.Object).AttributeTypes["consequence"].(tftypes.List).ElementType
		promote := consequence.(tftypes.Object).AttributeTypes["promote"].(tftypes.List).ElementType
		if _, ok := promote.(tftypes.Object).AttributeTypes["object_ids"].(tftypes.List); !ok {
			t.Errorf("%s object_ids is not a list", name)
		}
	}
}
