package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	agentStudio "github.com/algolia/algoliasearch-client-go/v4/algolia/agent-studio"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/call"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/transport"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDisplayResultsWithoutName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Tools  []map[string]json.RawMessage `json:"tools"`
			Config map[string]json.RawMessage   `json:"config"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(body.Tools) != 1 {
			t.Errorf("tools = %v", body.Tools)
		} else if _, ok := body.Tools[0]["name"]; ok {
			t.Error("unset display tool name must be omitted so the server assigns it")
		}
		if string(body.Config["large"]) != "9007199254740993" {
			t.Errorf("user JSON lost precision: %s", body.Config["large"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"agent","name":"agent","instructions":"show results","status":"draft","createdAt":"t","tools":[{"type":"algolia_display_results","name":"algolia_display_results"}]}`))
	}))
	defer server.Close()
	client, err := agentStudio.NewClientWithConfig(agentStudio.AgentStudioConfiguration{
		Configuration: transport.Configuration{
			AppID: "test-app", ApiKey: "test-key",
			Hosts: []transport.StatefulHost{
				transport.NewStatefulHost("http", server.Listener.Addr().String(), func(call.Kind) bool { return true }),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	tool := expandAlgoliaDisplayResultsTool(&ToolAlgoliaDisplayResultsModel{
		Name: types.StringUnknown(), MinGroups: types.Int64Null(), MaxGroups: types.Int64Null(),
		MinResultsPerGroup: types.Int64Null(), MaxResultsPerGroup: types.Int64Null(),
	})
	config := map[string]any{"large": json.Number("9007199254740993")}
	for _, update := range []bool{false, true} {
		var doc *agentDocument
		if update {
			doc, err = updateAgent(context.Background(), client, "agent", &agentStudio.AgentConfigUpdate{
				Tools: []agentStudio.ToolConfigInput{*tool}, Config: config,
			})
		} else {
			doc, err = createAgent(context.Background(), client, &agentStudio.AgentConfigCreate{
				Name: "agent", Instructions: "show results", Tools: []agentStudio.ToolConfigInput{*tool}, Config: config,
			})
		}
		if err != nil {
			t.Fatal(err)
		}
		var model AgentResourceModel
		if diags := flattenAgentResponse(context.Background(), doc, &model); diags.HasError() {
			t.Fatal(diags)
		}
		var displayTools []ToolAlgoliaDisplayResultsModel
		if diags := model.ToolAlgoliaDisplayResults.ElementsAs(context.Background(), &displayTools, false); diags.HasError() {
			t.Fatal(diags)
		}
		if len(displayTools) != 1 || displayTools[0].Name.ValueString() != "algolia_display_results" {
			t.Fatalf("server-assigned tool name must reach state: %v", displayTools)
		}
	}
}
