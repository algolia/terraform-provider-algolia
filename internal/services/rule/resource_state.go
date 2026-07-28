package rule

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// buildRuleRequest expands the model into the typed rule plus the raw
// `consequence.params` document. The raw document is kept out of the typed rule
// because search.ConsequenceParams has no AdditionalProperties, so
// round-tripping params_json through it silently drops every key the vendored
// client does not model yet. See ruleRequestBody for how the two are recombined.
func buildRuleRequest(model *RuleResourceModel) (*search.Rule, json.RawMessage, diag.Diagnostics) {
	var diags diag.Diagnostics

	consequence, rawParams, consequenceDiags := expandConsequence(model.Consequence)
	diags.Append(consequenceDiags...)
	if diags.HasError() {
		return nil, nil, diags
	}

	rule := search.NewRule(model.ObjectID.ValueString(), *consequence)

	if !model.Description.IsNull() && !model.Description.IsUnknown() {
		description := model.Description.ValueString()
		rule.Description = &description
	}
	if !model.Enabled.IsNull() && !model.Enabled.IsUnknown() {
		enabled := model.Enabled.ValueBool()
		rule.Enabled = &enabled
	}
	if !model.Scope.IsNull() && !model.Scope.IsUnknown() {
		scope := model.Scope.ValueString()
		rule.Scope = &scope
	}
	// A configured but empty list has to stay distinguishable from an absent
	// one: Rule.MarshalJSON emits `"tags": []` for a non-nil empty slice and
	// omits the key entirely for nil.
	rule.Tags = expandStringList(model.Tags)

	conditions, conditionDiags := expandConditions(model.Conditions)
	diags.Append(conditionDiags...)
	if diags.HasError() {
		return nil, nil, diags
	}
	if len(conditions) > 0 {
		rule.Conditions = conditions
	}

	validity, validityDiags := expandValidity(model.Validity)
	diags.Append(validityDiags...)
	if diags.HasError() {
		return nil, nil, diags
	}
	if len(validity) > 0 {
		rule.Validity = validity
	}

	return rule, rawParams, diags
}

// ruleRequestBodyFromModel expands the model into the SaveRule request body.
func ruleRequestBodyFromModel(model *RuleResourceModel) (map[string]any, diag.Diagnostics) {
	rule, rawParams, diags := buildRuleRequest(model)
	if diags.HasError() {
		return nil, diags
	}

	body, err := ruleRequestBody(rule, rawParams)
	if err != nil {
		diags.AddError("Error encoding rule", err.Error())
		return nil, diags
	}

	return body, diags
}

// ruleRequestBody serialises the typed rule and swaps `consequence.params` for
// the user's own JSON document so unmodelled search parameters reach the API.
// The document is spliced in as a json.RawMessage, which re-encodes byte for
// byte, so nothing about it is normalised on the way out either.
func ruleRequestBody(rule *search.Rule, rawParams json.RawMessage) (map[string]any, error) {
	encoded, err := json.Marshal(rule)
	if err != nil {
		return nil, fmt.Errorf("encode rule: %w", err)
	}

	body, err := decodeJSONObject(encoded)
	if err != nil {
		return nil, fmt.Errorf("re-encode rule: %w", err)
	}

	if rawParams == nil {
		return body, nil
	}

	consequence, ok := body["consequence"].(map[string]any)
	if !ok {
		consequence = map[string]any{}
		body["consequence"] = consequence
	}
	consequence["params"] = rawParams

	return body, nil
}

// saveRuleRaw performs the SaveRule PUT with a hand-built body instead of
// search.APIClient.SaveRule, which would re-encode the params through the typed
// struct and drop unmodelled keys.
func saveRuleRaw(ctx context.Context, client *search.APIClient, indexName, objectID string, body map[string]any) (int64, error) {
	res, resBody, err := client.CustomPutWithHTTPInfo(
		client.NewApiCustomPutRequest(rulePath(indexName, objectID)).WithBody(body),
		search.WithContext(ctx),
	)
	if err != nil {
		return 0, err
	}
	if res == nil {
		return 0, fmt.Errorf("no response from the Search API")
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode >= 300 {
		return 0, ruleAPIError(res.StatusCode, resBody)
	}

	var saved search.UpdatedAtResponse
	if err := json.Unmarshal(resBody, &saved); err != nil {
		return 0, fmt.Errorf("decode save rule response: %w", err)
	}

	return saved.TaskID, nil
}

// getRuleRaw reads the rule and returns both the typed representation and the
// untouched `consequence.params` document.
func getRuleRaw(ctx context.Context, client *search.APIClient, indexName, objectID string) (*search.Rule, json.RawMessage, error) {
	res, resBody, err := client.CustomGetWithHTTPInfo(
		client.NewApiCustomGetRequest(rulePath(indexName, objectID)),
		search.WithContext(ctx),
	)
	if err != nil {
		return nil, nil, err
	}
	if res == nil {
		return nil, nil, fmt.Errorf("no response from the Search API")
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode >= 300 {
		return nil, nil, ruleAPIError(res.StatusCode, resBody)
	}

	var rule search.Rule
	if err := json.Unmarshal(resBody, &rule); err != nil {
		return nil, nil, fmt.Errorf("decode rule: %w", err)
	}

	return &rule, extractRawParams(resBody), nil
}

// extractRawParams pulls `consequence.params` out of a rule payload without
// decoding it, so the bytes the API returned are the bytes that reach state.
func extractRawParams(payload []byte) json.RawMessage {
	var rule map[string]json.RawMessage
	if err := json.Unmarshal(payload, &rule); err != nil {
		return nil
	}

	var consequence map[string]json.RawMessage
	if err := json.Unmarshal(rule["consequence"], &consequence); err != nil {
		return nil
	}

	params, ok := consequence["params"]
	if !ok {
		return nil
	}

	return params
}

func rulePath(indexName, objectID string) string {
	return "1/indexes/" + url.PathEscape(indexName) + "/rules/" + url.PathEscape(objectID)
}

// ruleAPIError mirrors the vendored client's private error decoding so callers
// can keep matching on *search.APIError to detect a missing rule.
func ruleAPIError(status int, body []byte) error {
	message := http.StatusText(status)

	var decoded struct {
		Message *string `json:"message"`
	}
	if err := json.Unmarshal(body, &decoded); err == nil && decoded.Message != nil {
		message = summarizeErrorMessage(*decoded.Message)
	}

	return &search.APIError{Message: message, Status: status}
}

// maxErrorMessageLen bounds how much of an API-supplied message reaches a
// diagnostic. Response bodies are read unbounded by the client's transport and
// may come from an intermediary rather than Algolia, so they are neither size-
// nor content-trusted.
const maxErrorMessageLen = 2048

// summarizeErrorMessage makes an API-supplied message safe to put in a
// diagnostic: control characters collapse to spaces so it cannot forge extra log
// lines or emit terminal escapes, invalid UTF-8 is dropped, and truncation lands
// on a rune boundary.
func summarizeErrorMessage(message string) string {
	flattened := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}

		return r
	}, strings.ToValidUTF8(message, ""))

	summary := strings.TrimSpace(flattened)
	if len(summary) <= maxErrorMessageLen {
		return summary
	}

	cut := maxErrorMessageLen
	for cut > 0 && !utf8.RuneStart(summary[cut]) {
		cut--
	}

	return summary[:cut] + "... (truncated)"
}

// decodeJSONObject decodes into a generic object while keeping numbers as
// json.Number so re-encoding never reformats a value the user wrote.
func decodeJSONObject(data []byte) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var decoded map[string]any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}
	// A literal `null` decodes into a nil map without error, which callers would
	// then read as "no document at all".
	if decoded == nil {
		return nil, fmt.Errorf("expected a JSON object, got null")
	}

	return decoded, nil
}

func hydrateRuleModel(indexName string, ruleResp *search.Rule, rawParams json.RawMessage, model *RuleResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	priorConsequence := model.Consequence
	priorValidity := model.Validity

	model.ID = types.StringValue(ruleResourceID(indexName, ruleResp.GetObjectID()))
	model.IndexName = types.StringValue(indexName)
	model.ObjectID = types.StringValue(ruleResp.GetObjectID())

	if ruleResp.Description != nil {
		model.Description = types.StringValue(ruleResp.GetDescription())
	} else {
		model.Description = types.StringNull()
	}

	if ruleResp.Enabled != nil {
		model.Enabled = types.BoolValue(ruleResp.GetEnabled())
	} else {
		model.Enabled = types.BoolValue(true)
	}

	model.Scope = nullableString(ruleResp.Scope)
	model.Tags = flattenStringList(ruleResp.Tags)

	conditions, conditionDiags := flattenConditions(ruleResp.GetConditions())
	diags.Append(conditionDiags...)
	if diags.HasError() {
		return diags
	}
	model.Conditions = conditions

	consequence, consequenceDiags := flattenConsequence(ruleResp.GetConsequence(), rawParams, priorConsequence)
	diags.Append(consequenceDiags...)
	if diags.HasError() {
		return diags
	}
	model.Consequence = consequence

	validity, validityDiags := flattenValidity(ruleResp.GetValidity(), priorValidity)
	diags.Append(validityDiags...)
	if diags.HasError() {
		return diags
	}
	model.Validity = validity

	return diags
}

func parseRuleImportID(id string) (string, string, error) {
	index := strings.Index(id, "/")
	if index <= 0 || index == len(id)-1 {
		return "", "", fmt.Errorf("expected import ID in the form <index_name>/<object_id>")
	}

	return id[:index], id[index+1:], nil
}

func ruleResourceID(indexName, objectID string) string {
	return indexName + "/" + objectID
}

func expandConditions(list types.List) ([]search.Condition, diag.Diagnostics) {
	var diags diag.Diagnostics

	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}

	conditions := make([]search.Condition, 0, len(list.Elements()))
	for i, value := range list.Elements() {
		objValue, ok := value.(types.Object)
		if !ok {
			diags.AddError("Invalid condition value", fmt.Sprintf("Condition %d is not an object.", i))
			return nil, diags
		}

		attrs := objValue.Attributes()
		condition := search.NewEmptyCondition()

		if v, ok := attrs["pattern"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
			pattern := v.ValueString()
			condition.Pattern = &pattern
		}
		if v, ok := attrs["anchoring"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
			anchoring := search.Anchoring(v.ValueString())
			condition.Anchoring = &anchoring
		}
		if v, ok := attrs["alternatives"].(types.Bool); ok && !v.IsNull() && !v.IsUnknown() {
			alternatives := v.ValueBool()
			condition.Alternatives = &alternatives
		}
		if v, ok := attrs["context"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
			context := v.ValueString()
			condition.Context = &context
		}
		if v, ok := attrs["filters"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
			filters := v.ValueString()
			condition.Filters = &filters
		}

		conditions = append(conditions, *condition)
	}

	return conditions, diags
}

func flattenConditions(conditions []search.Condition) (types.List, diag.Diagnostics) {
	values := make([]attr.Value, 0, len(conditions))
	for _, condition := range conditions {
		value, diags := types.ObjectValue(conditionModelAttrTypes, map[string]attr.Value{
			"pattern":      nullableString(condition.Pattern),
			"anchoring":    nullableEnum(condition.Anchoring),
			"alternatives": nullableBool(condition.Alternatives),
			"context":      nullableString(condition.Context),
			"filters":      nullableString(condition.Filters),
		})
		if diags.HasError() {
			return types.ListNull(conditionModelType), diags
		}
		values = append(values, value)
	}

	return types.ListValue(conditionModelType, values)
}

// expandConsequence returns the typed consequence and, separately, the raw
// `params` document parsed straight out of params_json. Nothing decodes the
// params into search.ConsequenceParams: that struct has no
// AdditionalProperties, so any key Algolia ships ahead of a client regeneration
// would be dropped before the request is sent.
func expandConsequence(list types.List) (*search.Consequence, json.RawMessage, diag.Diagnostics) {
	var diags diag.Diagnostics

	if list.IsNull() || list.IsUnknown() || len(list.Elements()) == 0 {
		diags.AddError("Missing consequence", "A consequence block is required.")
		return nil, nil, diags
	}

	objValue, ok := list.Elements()[0].(types.Object)
	if !ok {
		diags.AddError("Invalid consequence value", "Consequence must be an object.")
		return nil, nil, diags
	}

	attrs := objValue.Attributes()
	consequence := search.NewConsequence()
	var rawParams json.RawMessage

	if v, ok := attrs["params_json"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
		// Decoded only to reject a document that is not a JSON object; the
		// original bytes are what gets sent.
		if _, err := decodeJSONObject([]byte(v.ValueString())); err != nil {
			diags.AddError("Invalid params_json", "Could not decode consequence params_json: "+err.Error())
			return nil, nil, diags
		}
		rawParams = json.RawMessage(v.ValueString())
	}

	if v, ok := attrs["filter_promotes"].(types.Bool); ok && !v.IsNull() && !v.IsUnknown() {
		filterPromotes := v.ValueBool()
		consequence.FilterPromotes = &filterPromotes
	}

	if v, ok := attrs["redirect_index_name"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
		consequence.Redirect = search.NewConsequenceRedirect(v.ValueString())
	}

	if v, ok := attrs["promote"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
		promote := make([]search.Promote, 0, len(v.Elements()))
		for i, entry := range v.Elements() {
			promoteObj, ok := entry.(types.Object)
			if !ok {
				diags.AddError("Invalid promote value", fmt.Sprintf("Promote entry %d is not an object.", i))
				return nil, nil, diags
			}
			promoteAttrs := promoteObj.Attributes()
			objectIDsValue, ok := promoteAttrs["object_ids"].(types.Set)
			if !ok || objectIDsValue.IsNull() || objectIDsValue.IsUnknown() {
				diags.AddError("Missing object_ids", "Each promote block requires object_ids.")
				return nil, nil, diags
			}
			positionValue, ok := promoteAttrs["position"].(types.Int64)
			if !ok || positionValue.IsNull() || positionValue.IsUnknown() {
				diags.AddError("Missing position", "Each promote block requires position.")
				return nil, nil, diags
			}

			objectIDs := setStrings(objectIDsValue)
			sort.Strings(objectIDs)
			promote = append(promote, *search.PromoteObjectIDsAsPromote(search.NewPromoteObjectIDs(objectIDs, int32(positionValue.ValueInt64()))))
		}
		consequence.Promote = promote
	}

	if v, ok := attrs["hide"].(types.Set); ok && !v.IsNull() && !v.IsUnknown() {
		objectIDs := setStrings(v)
		sort.Strings(objectIDs)
		hide := make([]search.ConsequenceHide, 0, len(objectIDs))
		for _, objectID := range objectIDs {
			hide = append(hide, *search.NewConsequenceHide(objectID))
		}
		consequence.Hide = hide
	}

	if v, ok := attrs["user_data"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
		userData, err := decodeJSONObject([]byte(v.ValueString()))
		if err != nil {
			diags.AddError("Invalid user_data", "Could not decode consequence user_data: "+err.Error())
			return nil, nil, diags
		}
		consequence.UserData = userData
	}

	return consequence, rawParams, diags
}

// flattenConsequence converts the API consequence into the single-element
// consequence block list. rawParams is the untouched `consequence.params`
// document from the API response, used instead of the typed params so
// unmodelled search parameters survive. prior is the model's existing
// consequence list; it is needed because `params_json`, `hide` and `user_data`
// are Optional and not Computed, so their planned values are the configuration
// verbatim and have to be carried over rather than replaced with a re-encoded
// equivalent.
func flattenConsequence(consequence search.Consequence, rawParams json.RawMessage, prior types.List) (types.List, diag.Diagnostics) {
	paramsValue := flattenConsequenceParams(rawParams, priorBlockString(prior, 0, "params_json"))

	promoteValues := make([]attr.Value, 0, len(consequence.GetPromote()))
	for _, promote := range consequence.GetPromote() {
		actual := promote.GetActualInstance()
		switch p := actual.(type) {
		case search.PromoteObjectIDs:
			objectIDs := stringSliceValues(p.GetObjectIDs())
			value, diags := types.ObjectValue(promoteModelAttrTypes, map[string]attr.Value{
				"object_ids": types.SetValueMust(types.StringType, objectIDs),
				"position":   types.Int64Value(int64(p.GetPosition())),
			})
			if diags.HasError() {
				return types.ListNull(consequenceModelType), diags
			}
			promoteValues = append(promoteValues, value)
		case search.PromoteObjectID:
			value, diags := types.ObjectValue(promoteModelAttrTypes, map[string]attr.Value{
				"object_ids": types.SetValueMust(types.StringType, []attr.Value{types.StringValue(p.GetObjectID())}),
				"position":   types.Int64Value(int64(p.GetPosition())),
			})
			if diags.HasError() {
				return types.ListNull(consequenceModelType), diags
			}
			promoteValues = append(promoteValues, value)
		}
	}
	promoteList, promoteDiags := types.ListValue(promoteModelType, promoteValues)
	if promoteDiags.HasError() {
		return types.ListNull(consequenceModelType), promoteDiags
	}

	hidden := make([]string, 0, len(consequence.GetHide()))
	for _, hide := range consequence.GetHide() {
		hidden = append(hidden, hide.GetObjectID())
	}
	hideSet := nullableStringSet(priorConsequenceSet(prior, "hide"), hidden)

	userDataValue, userDataDiags := flattenConsequenceUserData(consequence.UserData, priorBlockString(prior, 0, "user_data"))
	if userDataDiags.HasError() {
		return types.ListNull(consequenceModelType), userDataDiags
	}

	redirectValue := types.StringNull()
	if consequence.Redirect != nil {
		redirectValue = types.StringValue(consequence.Redirect.GetIndexName())
	}

	entry, diags := types.ObjectValue(consequenceModelAttrTypes, map[string]attr.Value{
		"params_json":         paramsValue,
		"promote":             promoteList,
		"hide":                hideSet,
		"user_data":           userDataValue,
		"filter_promotes":     nullableBool(consequence.FilterPromotes),
		"redirect_index_name": redirectValue,
	})
	if diags.HasError() {
		return types.ListNull(consequenceModelType), diags
	}

	return types.ListValue(consequenceModelType, []attr.Value{entry})
}

func expandValidity(list types.List) ([]search.TimeRange, diag.Diagnostics) {
	var diags diag.Diagnostics

	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}

	validity := make([]search.TimeRange, 0, len(list.Elements()))
	for i, value := range list.Elements() {
		objValue, ok := value.(types.Object)
		if !ok {
			diags.AddError("Invalid validity value", fmt.Sprintf("Validity entry %d is not an object.", i))
			return nil, diags
		}
		attrs := objValue.Attributes()
		rng := search.NewEmptyTimeRange()

		if v, ok := attrs["from"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
			parsed, err := time.Parse(time.RFC3339, v.ValueString())
			if err != nil {
				diags.AddError("Invalid validity.from", "Could not parse validity.from: "+err.Error())
				return nil, diags
			}
			value := parsed.UTC().Unix()
			rng.From = &value
		}
		if v, ok := attrs["until"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
			parsed, err := time.Parse(time.RFC3339, v.ValueString())
			if err != nil {
				diags.AddError("Invalid validity.until", "Could not parse validity.until: "+err.Error())
				return nil, diags
			}
			value := parsed.UTC().Unix()
			rng.Until = &value
		}

		validity = append(validity, *rng)
	}

	return validity, diags
}

// flattenValidity converts the API time ranges into the validity block list.
// prior is the model's existing validity list, matched element by element: the
// block is a list, so the API's Nth window is the configuration's Nth window.
func flattenValidity(validity []search.TimeRange, prior types.List) (types.List, diag.Diagnostics) {
	values := make([]attr.Value, 0, len(validity))
	for i, rng := range validity {
		value, diags := types.ObjectValue(validityModelAttrTypes, map[string]attr.Value{
			"from":  flattenTimestamp(rng.From, priorBlockString(prior, i, "from")),
			"until": flattenTimestamp(rng.Until, priorBlockString(prior, i, "until")),
		})
		if diags.HasError() {
			return types.ListNull(validityModelType), diags
		}
		values = append(values, value)
	}

	return types.ListValue(validityModelType, values)
}

func nullableString(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func nullableEnum[T ~string](value *T) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(string(*value))
}

func nullableBool(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}

// flattenTimestamp renders an API Unix second as an RFC3339 timestamp. `from`
// and `until` are Optional and not Computed, so the applied value has to equal
// the configured string, and rendering the second as UTC rewrites a configured
// zone offset and drops configured sub-second digits - Terraform rejects both as
// an inconsistent result even though the instant is unchanged. The configured
// string is therefore kept whenever it denotes the second the API returned.
func flattenTimestamp(value *int64, prior types.String) types.String {
	if value == nil {
		return types.StringNull()
	}

	if !prior.IsNull() && !prior.IsUnknown() {
		if parsed, err := time.Parse(time.RFC3339, prior.ValueString()); err == nil && parsed.Unix() == *value {
			return prior
		}
	}

	return types.StringValue(time.Unix(*value, 0).UTC().Format(time.RFC3339))
}

func setStrings(value types.Set) []string {
	stringsValue := make([]string, 0, len(value.Elements()))
	for _, element := range value.Elements() {
		if stringValue, ok := element.(types.String); ok && !stringValue.IsNull() && !stringValue.IsUnknown() {
			stringsValue = append(stringsValue, stringValue.ValueString())
		}
	}
	return stringsValue
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

	return types.SetValueMust(types.StringType, stringSliceValues(values))
}

// priorConsequenceSet reads a set attribute out of the model's existing
// single-element consequence block, falling back to null when there is no
// prior consequence (data source reads and imports).
func priorConsequenceSet(prior types.List, name string) types.Set {
	if prior.IsNull() || prior.IsUnknown() || len(prior.Elements()) == 0 {
		return types.SetNull(types.StringType)
	}

	objValue, ok := prior.Elements()[0].(types.Object)
	if !ok {
		return types.SetNull(types.StringType)
	}

	setValue, ok := objValue.Attributes()[name].(types.Set)
	if !ok {
		return types.SetNull(types.StringType)
	}

	return setValue
}

// priorBlockString reads a string attribute out of the model's existing value
// for a nested block, at the given element index. It falls back to null when
// there is no such element: data source reads and imports start from an empty
// model, and the API can return more blocks than the configuration has.
func priorBlockString(prior types.List, index int, name string) types.String {
	if prior.IsNull() || prior.IsUnknown() || index >= len(prior.Elements()) {
		return types.StringNull()
	}

	objValue, ok := prior.Elements()[index].(types.Object)
	if !ok {
		return types.StringNull()
	}

	stringValue, ok := objValue.Attributes()[name].(types.String)
	if !ok {
		return types.StringNull()
	}

	return stringValue
}

// flattenConsequenceParams renders the API's params document into params_json.
// The attribute is Optional and not Computed, so a document that differs from
// the configured one only in whitespace would make Terraform reject the apply as
// an inconsistent result. Whenever the configured document and the stored one
// carry the same data, the configured string is therefore kept verbatim.
func flattenConsequenceParams(rawParams json.RawMessage, prior types.String) types.String {
	if len(rawParams) == 0 {
		return types.StringNull()
	}

	if !prior.IsNull() && !prior.IsUnknown() && jsonEqual([]byte(prior.ValueString()), rawParams) {
		return prior
	}

	return types.StringValue(string(rawParams))
}

// flattenConsequenceUserData renders the API's userData document into user_data.
// The attribute is Optional and not Computed just like params_json, so the
// applied value has to equal the configured string, and re-encoding the decoded
// document does not reproduce it: keys come back in sorted order, the configured
// whitespace is gone, and - because decoding into search.Consequence turns every
// JSON number into a float64 - an integer above 2^53 comes back with different
// digits. Whenever the configured document and the stored one carry the same
// data, the configured string is therefore kept verbatim.
//
// The digit case depends on how jsonEqual compares: it decodes both sides into
// `any`, so the configured integer and the re-encoded one lose precision to the
// same float64 and compare equal, which is what lets the configured string win.
// Teaching jsonEqual to decode numbers as json.Number would make the two sides
// unequal here, and this function would go back to storing the re-encoded digits.
func flattenConsequenceUserData(userData map[string]any, prior types.String) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if userData == nil {
		return types.StringNull(), diags
	}

	encoded, err := json.Marshal(userData)
	if err != nil {
		diags.AddError("Error encoding consequence user_data", err.Error())
		return types.StringNull(), diags
	}

	if !prior.IsNull() && !prior.IsUnknown() && jsonEqual([]byte(prior.ValueString()), encoded) {
		return prior, diags
	}

	return types.StringValue(string(encoded)), diags
}

// jsonEqual reports whether two JSON documents carry the same data, ignoring
// key order and whitespace. Numbers are compared as float64, which
// flattenConsequenceUserData relies on; read its comment before making this
// stricter.
func jsonEqual(left, right []byte) bool {
	var leftDecoded, rightDecoded any
	if err := json.Unmarshal(left, &leftDecoded); err != nil {
		return false
	}
	if err := json.Unmarshal(right, &rightDecoded); err != nil {
		return false
	}

	return reflect.DeepEqual(leftDecoded, rightDecoded)
}

// expandStringList keeps the null/known-empty distinction: a null list becomes a
// nil slice so the field is omitted from the request, while a configured empty
// list becomes a non-nil empty slice so it is sent as `[]`.
func expandStringList(list types.List) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	values := make([]string, 0, len(list.Elements()))
	for _, element := range list.Elements() {
		if stringValue, ok := element.(types.String); ok && !stringValue.IsNull() && !stringValue.IsUnknown() {
			values = append(values, stringValue.ValueString())
		}
	}

	return values
}

// flattenStringList mirrors expandStringList: an absent field stays null, an
// empty array stays a known empty list.
func flattenStringList(values []string) types.List {
	if values == nil {
		return types.ListNull(types.StringType)
	}

	return types.ListValueMust(types.StringType, stringSliceValues(values))
}

func stringSliceValues(values []string) []attr.Value {
	result := make([]attr.Value, 0, len(values))
	for _, value := range values {
		result = append(result, types.StringValue(value))
	}
	return result
}
