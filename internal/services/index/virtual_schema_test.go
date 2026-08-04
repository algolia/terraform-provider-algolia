package index

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestVirtualIndexForbiddenSettingsAreComputedOnly(t *testing.T) {
	blocks := virtualIndexResourceSchema().Blocks

	tests := []struct {
		block     string
		attribute string
	}{
		{block: "attributes", attribute: "searchable_attributes"},
		{block: "attributes", attribute: "attribute_for_distinct"},
		{block: "faceting", attribute: "attributes_for_faceting"},
		{block: "typos", attribute: "disable_typo_tolerance_on_attributes"},
		{block: "languages", attribute: "attributes_to_transliterate"},
		{block: "languages", attribute: "decompounded_attributes"},
		{block: "languages", attribute: "index_languages"},
		{block: "advanced", attribute: "separators_to_index"},
	}

	for _, test := range tests {
		t.Run(test.block+"."+test.attribute, func(t *testing.T) {
			block, ok := blocks[test.block].(schema.SingleNestedBlock)
			if !ok {
				t.Fatalf("block %q has type %T", test.block, blocks[test.block])
			}

			switch attribute := block.Attributes[test.attribute].(type) {
			case schema.ListAttribute:
				if attribute.Optional || !attribute.Computed {
					t.Fatalf("Optional = %t, Computed = %t; want false, true", attribute.Optional, attribute.Computed)
				}
			case schema.StringAttribute:
				if attribute.Optional || !attribute.Computed {
					t.Fatalf("Optional = %t, Computed = %t; want false, true", attribute.Optional, attribute.Computed)
				}
			default:
				t.Fatalf("attribute has type %T", block.Attributes[test.attribute])
			}
		})
	}
}
