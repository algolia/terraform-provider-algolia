package synonym

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func buildSynonymHit(model *SynonymResourceModel) (*search.SynonymHit, diag.Diagnostics) {
	var diags diag.Diagnostics

	synonymType := canonicalSynonymType(model.Type.ValueString())
	hit := search.NewSynonymHit(model.ObjectID.ValueString(), apiSynonymType(synonymType))

	switch synonymType {
	case "synonym":
		synonyms := setStrings(model.Synonyms)
		if len(synonyms) == 0 {
			diags.AddError("Missing synonyms", "type = \"synonym\" requires synonyms.")
			return nil, diags
		}
		sort.Strings(synonyms)
		hit.Synonyms = synonyms
	case "oneWaySynonym":
		if model.Input.IsNull() || model.Input.IsUnknown() || model.Input.ValueString() == "" {
			diags.AddError("Missing input", "type = \"oneWaySynonym\" requires input.")
			return nil, diags
		}
		synonyms := setStrings(model.Synonyms)
		if len(synonyms) == 0 {
			diags.AddError("Missing synonyms", "type = \"oneWaySynonym\" requires synonyms.")
			return nil, diags
		}
		sort.Strings(synonyms)
		hit.Input = stringPtr(model.Input.ValueString())
		hit.Synonyms = synonyms
	case "altCorrection1", "altCorrection2":
		if model.Word.IsNull() || model.Word.IsUnknown() || model.Word.ValueString() == "" {
			diags.AddError("Missing word", "Alternative correction synonym types require word.")
			return nil, diags
		}
		corrections := setStrings(model.Corrections)
		if len(corrections) == 0 {
			diags.AddError("Missing corrections", "Alternative correction synonym types require corrections.")
			return nil, diags
		}
		sort.Strings(corrections)
		hit.Word = stringPtr(model.Word.ValueString())
		hit.Corrections = corrections
	case "placeholder":
		if model.Placeholder.IsNull() || model.Placeholder.IsUnknown() || model.Placeholder.ValueString() == "" {
			diags.AddError("Missing placeholder", "type = \"placeholder\" requires placeholder.")
			return nil, diags
		}
		replacements := setStrings(model.Replacements)
		if len(replacements) == 0 {
			diags.AddError("Missing replacements", "type = \"placeholder\" requires replacements.")
			return nil, diags
		}
		sort.Strings(replacements)
		hit.Placeholder = stringPtr(model.Placeholder.ValueString())
		hit.Replacements = replacements
	default:
		diags.AddError("Unsupported synonym type", "Unknown synonym type "+model.Type.ValueString())
		return nil, diags
	}

	return hit, diags
}

func hydrateSynonymModel(indexName string, hit *search.SynonymHit, model *SynonymResourceModel) diag.Diagnostics {
	priorSynonyms := model.Synonyms
	priorCorrections := model.Corrections
	priorReplacements := model.Replacements

	model.ID = types.StringValue(synonymResourceID(indexName, hit.GetObjectID()))
	model.IndexName = types.StringValue(indexName)
	model.ObjectID = types.StringValue(hit.GetObjectID())
	model.Type = types.StringValue(canonicalSynonymType(string(hit.GetType())))
	model.Synonyms = nullableStringSet(priorSynonyms, hit.GetSynonyms())
	model.Input = nullableString(hit.Input)
	model.Word = nullableString(hit.Word)
	model.Corrections = nullableStringSet(priorCorrections, hit.GetCorrections())
	model.Placeholder = nullableString(hit.Placeholder)
	model.Replacements = nullableStringSet(priorReplacements, hit.GetReplacements())

	return nil
}

func parseSynonymImportID(id string) (string, string, error) {
	index := strings.Index(id, "/")
	if index <= 0 || index == len(id)-1 {
		return "", "", fmt.Errorf("expected import ID in the form <index_name>/<object_id>")
	}
	return id[:index], id[index+1:], nil
}

func synonymResourceID(indexName, objectID string) string {
	return indexName + "/" + objectID
}

func canonicalSynonymType(value string) string {
	switch value {
	case "onewaysynonym", "oneWaySynonym":
		return "oneWaySynonym"
	case "altcorrection1", "altCorrection1":
		return "altCorrection1"
	case "altcorrection2", "altCorrection2":
		return "altCorrection2"
	default:
		return value
	}
}

func apiSynonymType(value string) search.SynonymType {
	switch value {
	case "oneWaySynonym":
		return search.SYNONYM_TYPE_ONE_WAY_SYNONYM
	case "altCorrection1":
		return search.SYNONYM_TYPE_ALT_CORRECTION1
	case "altCorrection2":
		return search.SYNONYM_TYPE_ALT_CORRECTION2
	case "placeholder":
		return search.SYNONYM_TYPE_PLACEHOLDER
	default:
		return search.SYNONYM_TYPE_SYNONYM
	}
}

func waitForSynonymTask(client *search.APIClient, indexName string, taskID int64) error {
	deadline := time.Now().Add(30 * time.Minute)
	interval := 2 * time.Second
	for time.Now().Before(deadline) {
		resp, err := client.GetTask(client.NewApiGetTaskRequest(indexName, taskID))
		if err != nil {
			return err
		}
		if resp.Status == search.TASK_STATUS_PUBLISHED {
			return nil
		}
		time.Sleep(interval)
		if interval < 10*time.Second {
			interval += time.Second
		}
	}
	return fmt.Errorf("task %d on index %q did not complete within 30 minutes", taskID, indexName)
}

// nullableStringSet converts an API string slice into a Terraform set. For an
// Optional, non-Computed attribute the planned value is the configuration
// verbatim, so emitting a known empty set where the plan held null makes
// Terraform reject the apply with "Provider produced inconsistent result after
// apply". When the API returns nothing, the prior value therefore decides: a
// null prior stays null, while a prior that was explicitly configured as `[]`
// stays a known empty set.
func nullableStringSet(prior types.Set, values []string) types.Set {
	if len(values) == 0 {
		if prior.IsNull() || prior.IsUnknown() {
			return types.SetNull(types.StringType)
		}

		return types.SetValueMust(types.StringType, []attr.Value{})
	}

	attrValues := make([]attr.Value, 0, len(values))
	for _, value := range values {
		attrValues = append(attrValues, types.StringValue(value))
	}
	return types.SetValueMust(types.StringType, attrValues)
}

func setStrings(value types.Set) []string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	values := make([]string, 0, len(value.Elements()))
	for _, element := range value.Elements() {
		if stringValue, ok := element.(types.String); ok && !stringValue.IsNull() && !stringValue.IsUnknown() {
			values = append(values, stringValue.ValueString())
		}
	}
	return values
}

func stringPtr(value string) *string {
	return &value
}

func nullableString(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}
