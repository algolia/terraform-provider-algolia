package agent

import (
	"encoding/json"

	agentStudio "github.com/algolia/algoliasearch-client-go/v4/algolia/agent-studio"
)

// displayResultsToolConfig models the display-results wire format, which the
// client represents as UnknownToolConfig.
type displayResultsToolConfig struct {
	Name               *string `json:"name,omitempty"`
	MinGroups          *int32  `json:"minGroups,omitempty"`
	MaxGroups          *int32  `json:"maxGroups,omitempty"`
	MinResultsPerGroup *int32  `json:"minResultsPerGroup,omitempty"`
	MaxResultsPerGroup *int32  `json:"maxResultsPerGroup,omitempty"`
}

// UnknownToolConfig requires a name, but the legacy display tool lets the server
// assign one. Remove the carrier's empty name before sending that tool. Override
// all top-level fields as RawMessages: the client's body-parameter merge otherwise
// decodes the typed body through float64 and loses precision in user JSON.
func displayResultsWriteOptions(config any, tools []agentStudio.ToolConfigInput) ([]agentStudio.RequestOption, error) {
	var unnamed []int
	for i, tool := range tools {
		if t := tool.UnknownToolConfig; t != nil && t.Type == "algolia_display_results" && t.Name == "" {
			unnamed = append(unnamed, i)
		}
	}
	if len(unnamed) == 0 {
		return nil, nil
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &body); err != nil {
		return nil, err
	}
	var rawTools []json.RawMessage
	if err := json.Unmarshal(body["tools"], &rawTools); err != nil {
		return nil, err
	}
	for _, i := range unnamed {
		var tool map[string]json.RawMessage
		if err := json.Unmarshal(rawTools[i], &tool); err != nil {
			return nil, err
		}
		delete(tool, "name")
		rawTools[i], err = json.Marshal(tool)
		if err != nil {
			return nil, err
		}
	}
	body["tools"], err = json.Marshal(rawTools)
	if err != nil {
		return nil, err
	}
	params := make(map[string]any, len(body))
	for key, value := range body {
		params[key] = value
	}
	return []agentStudio.RequestOption{agentStudio.WithBodyParams(params)}, nil
}
