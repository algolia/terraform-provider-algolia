package index

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestRenderingContentJSONValidator(t *testing.T) {
	for _, test := range []struct {
		name      string
		value     string
		wantError bool
	}{
		{name: "valid", value: `{"facetOrdering":{"facets":{"order":["brand"]}}}`},
		{name: "invalid JSON", value: `{`, wantError: true},
		{name: "null JSON", value: `null`, wantError: true},
		{name: "trailing JSON", value: `{} {}`, wantError: true},
		{name: "unsupported field", value: `{"futureField":true}`, wantError: true},
		{name: "case variant field", value: `{"FacetOrdering":{"facets":{"order":["brand"]}}}`, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := validator.StringRequest{
				Path:        path.Root("rendering_content"),
				ConfigValue: types.StringValue(test.value),
			}
			response := &validator.StringResponse{}

			renderingContentJSONValidator{}.ValidateString(context.Background(), request, response)
			if response.Diagnostics.HasError() != test.wantError {
				t.Fatalf("diagnostics error = %t, want %t: %v", response.Diagnostics.HasError(), test.wantError, response.Diagnostics)
			}
		})
	}
}
