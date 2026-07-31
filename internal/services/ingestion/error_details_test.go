package ingestion

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The Ingestion API answers an invalid request with a summary plus a list of what
// exactly was wrong. Reporting only the summary - "Invalid payload, see
// error.details" - names details the operator cannot see, which is how the
// `criticalThreshold` cap and the no-code step shape each cost a round of manual
// API probing to identify.
//
// This drives a real Create against a fake host returning that shape, so it guards
// the wiring rather than the helper: reverting a call site to err.Error() fails
// here. The helper's own behaviour, including the shapes it has to tolerate, is
// covered in internal/algoliaerr.
const invalidPayloadResponse = `{
  "message": "Invalid payload, see error.details",
  "status": 400,
  "error": {
    "code": "invalid_payload",
    "details": [
      {"label": "policies.criticalThreshold", "message": "'criticalThreshold' must be lower or equal to '10'"}
    ]
  }
}`

func invalidPayloadHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(invalidPayloadResponse))
	}
}

func TestCreateTask_diagnosticNamesTheRejectedField(t *testing.T) {
	ctx := context.Background()
	client := newTestIngestionClient(t, invalidPayloadHandler())

	plan := TaskResourceModel{
		ID:            types.StringUnknown(),
		TaskID:        types.StringUnknown(),
		SourceID:      types.StringValue("11111111-1111-1111-1111-111111111111"),
		DestinationID: types.StringValue("22222222-2222-2222-2222-222222222222"),
		Action:        types.StringValue("replace"),
		Enabled:       types.BoolValue(true),
		Policies:      types.StringValue(`{"criticalThreshold":50}`),
	}

	resp := &resource.CreateResponse{State: emptyState(t, taskResourceSchema())}
	createTask(ctx, client, &plan, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("createTask succeeded against a host that rejects everything, want an error")
	}

	detail := resp.Diagnostics.Errors()[0].Detail()
	for _, want := range []string{
		// The summary the client produced, which carries the status.
		"Invalid payload, see error.details",
		// And the part that was missing: which field, and why.
		"policies.criticalThreshold",
		"must be lower or equal to '10'",
	} {
		if !strings.Contains(detail, want) {
			t.Errorf("diagnostic does not mention %q:\n%s", want, detail)
		}
	}
}
