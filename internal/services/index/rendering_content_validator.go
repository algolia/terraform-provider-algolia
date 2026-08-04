package index

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type renderingContentJSONValidator struct{}

func (renderingContentJSONValidator) Description(context.Context) string {
	return "must be valid rendering content supported by this provider version"
}

func (v renderingContentJSONValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (renderingContentJSONValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if _, err := decodeRenderingContent(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid rendering_content",
			"Rendering content contains invalid or unsupported JSON: "+err.Error(),
		)
	}
}
