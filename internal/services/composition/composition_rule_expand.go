package composition

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"

	compositionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/composition"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// expandCompositionRule converts the Terraform plan into a CompositionRule
// for PutCompositionRule.
func expandCompositionRule(objectID string, model *CompositionRuleResourceModel) (*compositionapi.CompositionRule, diag.Diagnostics) {
	var diags diag.Diagnostics

	rule := compositionapi.NewEmptyCompositionRule()
	rule.ObjectID = objectID

	if !model.Conditions.IsNull() && !model.Conditions.IsUnknown() && model.Conditions.ValueString() != "" {
		var conditions []compositionapi.Condition
		if err := json.Unmarshal([]byte(model.Conditions.ValueString()), &conditions); err != nil {
			diags.AddError(
				"Invalid conditions JSON",
				"The `conditions` attribute must be a JSON-encoded array of conditions (e.g. jsonencode([{ "+
					"filters = \"brand:apple\" }])). Failed to parse: "+err.Error(),
			)
			return nil, diags
		}
		rule.Conditions = conditions
	}

	if model.Consequence.IsNull() || model.Consequence.IsUnknown() || model.Consequence.ValueString() == "" {
		diags.AddError("Missing consequence", "The `consequence` attribute is required.")
		return nil, diags
	}
	var consequence compositionapi.CompositionRuleConsequence
	if err := json.Unmarshal([]byte(model.Consequence.ValueString()), &consequence); err != nil {
		diags.AddError(
			"Invalid consequence JSON",
			"The `consequence` attribute must be a JSON-encoded object with a `behavior` key, using the same "+
				"shape as algolia_composition's own `behavior` attribute. Failed to parse: "+err.Error(),
		)
		return nil, diags
	}
	rule.Consequence = consequence

	if !model.Description.IsNull() && !model.Description.IsUnknown() {
		description := model.Description.ValueString()
		rule.Description = &description
	}

	if !model.Enabled.IsNull() && !model.Enabled.IsUnknown() {
		enabled := model.Enabled.ValueBool()
		rule.Enabled = &enabled
	}

	if !model.Validity.IsNull() && !model.Validity.IsUnknown() && model.Validity.ValueString() != "" {
		var validity []compositionapi.TimeRange
		if err := json.Unmarshal([]byte(model.Validity.ValueString()), &validity); err != nil {
			diags.AddError(
				"Invalid validity JSON",
				"The `validity` attribute must be a JSON-encoded array of time ranges (e.g. jsonencode([{ from "+
					"= 1893456000, until = 1893542400 }])). Failed to parse: "+err.Error(),
			)
			return nil, diags
		}
		rule.Validity = validity
	}

	rule.Tags = expandStringList(model.Tags)

	return rule, diags
}

// expandStringList converts a Terraform list of strings into a []string,
// returning nil for a null/unknown list. Mirrors the identically named
// helper in internal/services/dictionary (replicated here rather than
// imported, per that package's own note on jsonSemanticallyEqual).
func expandStringList(list types.List) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	values := make([]string, 0, len(list.Elements()))
	for _, element := range list.Elements() {
		if value, ok := element.(types.String); ok && !value.IsNull() && !value.IsUnknown() {
			values = append(values, value.ValueString())
		}
	}

	return values
}

// parseCompositionRuleImportID parses a `terraform import` ID in the form
// <composition_id>/<object_id>, mirroring parseDictionaryEntryImportID in
// the dictionary package (and parseRuleImportID/parseSynonymImportID in the
// rule/synonym packages).
func parseCompositionRuleImportID(id string) (string, string, error) {
	index := strings.Index(id, "/")
	if index <= 0 || index == len(id)-1 {
		return "", "", fmt.Errorf("expected import ID in the form <composition_id>/<object_id>")
	}

	return id[:index], id[index+1:], nil
}

// compositionRuleResourceID builds the Terraform identifier for a
// composition rule.
func compositionRuleResourceID(compositionID, objectID string) string {
	return compositionID + "/" + objectID
}

// generateObjectID returns a randomly generated RFC 4122 version 4 UUID,
// used as the object_id when the user does not supply one. Mirrors
// recommend.generateObjectID/dictionary.generateObjectID (replicated here
// rather than imported, per those packages' own rationale).
func generateObjectID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("could not generate random object_id: %w", err)
	}

	buf[6] = (buf[6] & 0x0f) | 0x40 // version 4
	buf[8] = (buf[8] & 0x3f) | 0x80 // variant 10

	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16]), nil
}
