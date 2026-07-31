package ingestion

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// removalTestSchema is a one-attribute stand-in for the task schema, so the
// modifier can be driven through the framework - including its own create,
// destroy and no-op guards - without building a value for every task attribute.
func removalTestSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"cron": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{requiresReplaceOnRemoval()},
			},
		},
	}
}

// rawObject builds a whole-resource value for removalTestSchema. Passing nil
// produces the null object the framework uses to mean "no state yet" on create
// and "no plan" on destroy.
func rawObject(ctx context.Context, s schema.Schema, cron *string) tftypes.Value {
	objectType := s.Type().TerraformType(ctx)
	if cron == nil {
		return tftypes.NewValue(objectType, nil)
	}

	return tftypes.NewValue(objectType, map[string]tftypes.Value{
		"cron": tftypes.NewValue(tftypes.String, *cron),
	})
}

func TestRequiresReplaceOnRemoval(t *testing.T) {
	ctx := context.Background()
	s := removalTestSchema()
	scheduled := "0 0 * * *"
	rescheduled := "30 4 * * 1"

	cases := []struct {
		name        string
		stateRaw    *string // nil means no prior state, i.e. a create
		planRaw     *string // nil means no plan, i.e. a destroy
		stateValue  types.String
		planValue   types.String
		configValue types.String
		want        bool
	}{
		{
			name:        "removed from configuration",
			stateRaw:    &scheduled,
			planRaw:     &scheduled,
			stateValue:  types.StringValue(scheduled),
			planValue:   types.StringNull(),
			configValue: types.StringNull(),
			want:        true,
		},
		{
			// Changing a schedule is a plain PATCH, so replacing the task here
			// would destroy and recreate it for no reason.
			name:        "value changed",
			stateRaw:    &scheduled,
			planRaw:     &rescheduled,
			stateValue:  types.StringValue(scheduled),
			planValue:   types.StringValue(rescheduled),
			configValue: types.StringValue(rescheduled),
			want:        false,
		},
		{
			name:        "unchanged",
			stateRaw:    &scheduled,
			planRaw:     &scheduled,
			stateValue:  types.StringValue(scheduled),
			planValue:   types.StringValue(scheduled),
			configValue: types.StringValue(scheduled),
			want:        false,
		},
		{
			// A create has no prior state; a null configuration value simply means
			// the task is on-demand from the start.
			name:        "create without the attribute",
			stateRaw:    nil,
			planRaw:     nil,
			stateValue:  types.StringNull(),
			planValue:   types.StringNull(),
			configValue: types.StringNull(),
			want:        false,
		},
		{
			// On destroy the configuration is empty for every attribute. Treating
			// that as a removal would tell Terraform to replace a resource it is
			// trying to delete.
			name:        "destroy",
			stateRaw:    &scheduled,
			planRaw:     nil,
			stateValue:  types.StringValue(scheduled),
			planValue:   types.StringNull(),
			configValue: types.StringNull(),
			want:        false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				Path:        path.Root("cron"),
				State:       tfsdk.State{Schema: s, Raw: rawObject(ctx, s, tc.stateRaw)},
				Plan:        tfsdk.Plan{Schema: s, Raw: rawObject(ctx, s, tc.planRaw)},
				Config:      tfsdk.Config{Schema: s, Raw: rawObject(ctx, s, tc.planRaw)},
				StateValue:  tc.stateValue,
				PlanValue:   tc.planValue,
				ConfigValue: tc.configValue,
			}
			resp := &planmodifier.StringResponse{PlanValue: tc.planValue}

			requiresReplaceOnRemoval().PlanModifyString(ctx, req, resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("plan modifier reported an error: %v", resp.Diagnostics)
			}
			if resp.RequiresReplace != tc.want {
				t.Errorf("RequiresReplace = %v, want %v", resp.RequiresReplace, tc.want)
			}
		})
	}
}

// TestTaskResourceSchema_RemovalRequiresReplace pins the wiring. The Ingestion
// API can set and change these two fields but not clear them, so without a
// replace-on-removal modifier a configuration that drops one proposes the same
// removal on every plan and never converges.
func TestTaskResourceSchema_RemovalRequiresReplace(t *testing.T) {
	s := taskResourceSchema()

	for _, name := range []string{"cron", "subscription_action"} {
		t.Run(name, func(t *testing.T) {
			attribute, ok := s.Attributes[name].(schema.StringAttribute)
			if !ok {
				t.Fatalf("expected %s to be a string attribute", name)
			}
			if len(attribute.PlanModifiers) == 0 {
				t.Fatalf("expected %s to carry a replace-on-removal plan modifier", name)
			}
		})
	}
}
