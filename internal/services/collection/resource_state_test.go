package collection

import (
	"context"
	"testing"

	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDeletionProtectionValue_DefaultsToTrue(t *testing.T) {
	if !deletionProtectionValue(types.BoolNull()).ValueBool() {
		t.Error("null -> expected true")
	}
	if !deletionProtectionValue(types.BoolUnknown()).ValueBool() {
		t.Error("unknown -> expected true")
	}
	if deletionProtectionValue(types.BoolValue(false)).ValueBool() {
		t.Error("explicit false must be preserved")
	}
}

func TestCommitValue_DefaultsToTrue(t *testing.T) {
	if !commitValue(types.BoolNull()).ValueBool() {
		t.Error("null -> expected true")
	}
	if !commitValue(types.BoolUnknown()).ValueBool() {
		t.Error("unknown -> expected true")
	}
	if commitValue(types.BoolValue(false)).ValueBool() {
		t.Error("explicit false must be preserved")
	}
}

func TestHydrateImportedCollectionResourceState_EnablesDeletionProtection(t *testing.T) {
	ctx := context.Background()
	resp := &CollectionResponse{
		ID:        "coll-7",
		Name:      "imported",
		IndexName: "products",
		CreatedAt: "2026-04-01T00:00:00Z",
	}

	model := &CollectionResourceModel{}
	diags := hydrateImportedCollectionResourceState(ctx, resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}

	if model.DeletionProtection.IsNull() || !model.DeletionProtection.ValueBool() {
		t.Errorf("expected deletion_protection=true after import, got %#v", model.DeletionProtection)
	}
	if model.Commit.IsNull() || !model.Commit.ValueBool() {
		t.Errorf("expected commit=true after import, got %#v", model.Commit)
	}
}

func TestHydrateCollectionResourceState_PreservesLocalFlags(t *testing.T) {
	ctx := context.Background()
	resp := &CollectionResponse{
		ID:        "coll-8",
		Name:      "preserved",
		IndexName: "products",
		CreatedAt: "2026-04-01T00:00:00Z",
	}

	model := &CollectionResourceModel{}
	diags := hydrateCollectionResourceState(ctx, resp, types.BoolValue(false), types.BoolValue(false), model)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}

	if model.Commit.ValueBool() {
		t.Errorf("expected commit=false to be preserved, got true")
	}
	if model.DeletionProtection.ValueBool() {
		t.Errorf("expected deletion_protection=false to be preserved, got true")
	}
}

func TestCollectionResourceSchema_HasRequiredFlags(t *testing.T) {
	s := collectionResourceSchema()

	commit, ok := s.Attributes["commit"].(resourceschema.BoolAttribute)
	if !ok {
		t.Fatal("expected commit to be a bool attribute")
	}
	if !commit.Optional || !commit.Computed {
		t.Error("commit must be Optional + Computed to support default")
	}

	dp, ok := s.Attributes["deletion_protection"].(resourceschema.BoolAttribute)
	if !ok {
		t.Fatal("expected deletion_protection to be a bool attribute")
	}
	if !dp.Optional || !dp.Computed {
		t.Error("deletion_protection must be Optional + Computed to support default")
	}
}

func TestParseCollectionImportID(t *testing.T) {
	cases := []struct {
		name         string
		id           string
		wantIndex    string
		wantID       string
		wantErr      bool
	}{
		{"well-formed", "products/deadbeef-uuid", "products", "deadbeef-uuid", false},
		{"index with hyphen", "prod-en/abc-123", "prod-en", "abc-123", false},
		{"empty string", "", "", "", true},
		{"no separator", "just-a-uuid", "", "", true},
		{"leading slash", "/uuid", "", "", true},
		{"trailing slash", "products/", "", "", true},
		{"only slash", "/", "", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			idx, id, err := parseCollectionImportID(tc.id)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got idx=%q id=%q", idx, id)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if idx != tc.wantIndex {
				t.Errorf("index: got %q, want %q", idx, tc.wantIndex)
			}
			if id != tc.wantID {
				t.Errorf("id: got %q, want %q", id, tc.wantID)
			}
		})
	}
}

func TestCollectionDataSourceSchema_OmitsResourceOnlyFlags(t *testing.T) {
	s := collectionDataSourceSchema()
	for _, field := range []string{"commit", "deletion_protection"} {
		if _, exists := s.Attributes[field]; exists {
			t.Errorf("data source must not expose %q", field)
		}
	}

	id, ok := s.Attributes["id"].(datasourceschema.StringAttribute)
	if !ok {
		t.Fatal("expected id to be a string attribute")
	}
	if !id.Required {
		t.Error("data source id must be Required for lookups")
	}
}
