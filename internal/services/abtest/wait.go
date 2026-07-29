package abtest

import (
	"context"
	"fmt"

	abtestingapi "github.com/algolia/algoliasearch-client-go/v4/algolia/abtesting-v3"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/terraform-provider-algolia/internal/algoliawait"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// waitForABTestTask blocks until the search task an A/B test write queued has
// been published.
//
// Every write on this API returns `{index, abTestID, taskID}`, and the client's
// own model documents what that means: "A successful API response means that a
// task was added to a queue. It might not run immediately." Returning as soon as
// the call succeeds therefore reports work that has not happened yet.
//
// That is not a theoretical race. Deleting an A/B test and its indexes in one
// `terraform destroy` failed reproducibly: the test was gone from the API while
// Algolia still rejected deleting the indexes it had referenced, with
//
//	API error [403] cannot delete with an index under AB testing index as destination
//
// leaving the indexes behind for the operator to clean up. The very same deletes
// succeeded once the queued task had run. The task is an ordinary search index
// task on the index named in the response, so it is waited on with the search
// client rather than the A/B Testing one.
//
// A nil searchClient means the provider was configured without search
// credentials, which cannot happen for a configured provider; it is tolerated
// here rather than panicking, since skipping the wait only restores the previous
// behaviour.
//
// A response carrying an index but no task ID is a different matter. TaskID is a
// plain int64 in the generated client, so a zero is indistinguishable from a
// field that failed to decode, and skipping the wait in that case would quietly
// reintroduce the race this function exists to close. There is nothing useful to
// fail on - the write itself already succeeded - so it is logged loudly instead of
// passing silently.
func waitForABTestTask(ctx context.Context, searchClient *search.APIClient, resp *abtestingapi.ABTestResponse) error {
	if searchClient == nil || resp == nil {
		return nil
	}

	if resp.TaskID == 0 || resp.Index == "" {
		if resp.Index != "" || resp.TaskID != 0 {
			tflog.Warn(ctx, "A/B test response carried no task to wait for; skipping the wait", map[string]any{
				"index":   resp.Index,
				"task_id": resp.TaskID,
			})
		}

		return nil
	}

	indexName := resp.Index
	taskID := resp.TaskID
	subject := fmt.Sprintf("A/B test task %d on index %q", taskID, indexName)

	return algoliawait.Until(ctx, subject, func(ctx context.Context) (bool, error) {
		task, err := searchClient.GetTask(searchClient.NewApiGetTaskRequest(indexName, taskID), search.WithContext(ctx))
		if err != nil {
			return false, err
		}

		return task.Status == search.TASK_STATUS_PUBLISHED, nil
	})
}
