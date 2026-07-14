package recommend

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"

	recommendapi "github.com/algolia/algoliasearch-client-go/v4/algolia/recommend"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// expandRecommendRule converts the Terraform plan into a RecommendRule for
// BatchRecommendRules.
func expandRecommendRule(objectID string, model *RecommendRuleResourceModel) (*recommendapi.RecommendRule, diag.Diagnostics) {
	var diags diag.Diagnostics

	rule := recommendapi.NewEmptyRecommendRule()
	rule.ObjectID = &objectID

	if !model.Condition.IsNull() && !model.Condition.IsUnknown() && model.Condition.ValueString() != "" {
		var condition recommendapi.Condition
		if err := json.Unmarshal([]byte(model.Condition.ValueString()), &condition); err != nil {
			diags.AddError(
				"Invalid condition JSON",
				"The `condition` attribute must be JSON-encoded (e.g. jsonencode({ filters = \"brand:apple\" })). "+
					"Failed to parse: "+err.Error(),
			)
			return nil, diags
		}
		rule.Condition = &condition
	}

	if model.Consequence.IsNull() || model.Consequence.IsUnknown() || model.Consequence.ValueString() == "" {
		diags.AddError("Missing consequence", "The `consequence` attribute is required.")
		return nil, diags
	}
	var consequence recommendapi.Consequence
	if err := json.Unmarshal([]byte(model.Consequence.ValueString()), &consequence); err != nil {
		diags.AddError(
			"Invalid consequence JSON",
			"The `consequence` attribute must be JSON-encoded (e.g. jsonencode({ hide = [{ objectID = \"42\" }] "+
				"})). Failed to parse: "+err.Error(),
		)
		return nil, diags
	}
	rule.Consequence = &consequence

	if !model.Description.IsNull() && !model.Description.IsUnknown() {
		description := model.Description.ValueString()
		rule.Description = &description
	}

	if !model.Enabled.IsNull() && !model.Enabled.IsUnknown() {
		enabled := model.Enabled.ValueBool()
		rule.Enabled = &enabled
	}

	if !model.Validity.IsNull() && !model.Validity.IsUnknown() && model.Validity.ValueString() != "" {
		var validity []recommendapi.TimeRange
		if err := json.Unmarshal([]byte(model.Validity.ValueString()), &validity); err != nil {
			diags.AddError(
				"Invalid validity JSON",
				"The `validity` attribute must be a JSON-encoded array of time ranges (e.g. jsonencode([{ from = "+
					"1893456000, until = 1893542400 }])). Failed to parse: "+err.Error(),
			)
			return nil, diags
		}
		rule.Validity = validity
	}

	return rule, diags
}

// parseRecommendRuleImportID parses a `terraform import` ID in the form
// <index_name>/<model>/<object_id>, mirroring the two-part
// parseRuleImportID/parseSynonymImportID helpers in the rule/synonym
// packages, extended to three parts.
func parseRecommendRuleImportID(id string) (string, string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("expected import ID in the form <index_name>/<model>/<object_id>")
	}

	return parts[0], parts[1], parts[2], nil
}

// recommendRuleResourceID builds the Terraform identifier for a Recommend
// rule.
func recommendRuleResourceID(indexName, model, objectID string) string {
	return indexName + "/" + model + "/" + objectID
}

// generateObjectID returns a randomly generated RFC 4122 version 4 UUID,
// used as the object_id when the user does not supply one. Mirrors
// internal/services/dictionary's generateObjectID (not imported across
// service packages, per that package's own note on jsonSemanticallyEqual).
func generateObjectID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("could not generate random object_id: %w", err)
	}

	buf[6] = (buf[6] & 0x0f) | 0x40 // version 4
	buf[8] = (buf[8] & 0x3f) | 0x80 // variant 10

	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16]), nil
}
