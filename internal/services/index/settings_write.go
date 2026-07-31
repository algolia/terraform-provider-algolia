package index

import (
	"context"
	"fmt"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/terraform-provider-algolia/internal/algoliawait"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// writeReissueGrace is how long a settings write waits on one task ID before
// sending the write again, and writeReissueLimit caps how many times it does so.
//
// Settings tasks against the live API publish within a few seconds, so the grace
// is generous for a healthy queue while keeping a void task ID to seconds rather
// than the whole wait budget. Both are vars only so a unit test can shorten them.
var (
	writeReissueGrace = 30 * time.Second
	writeReissueLimit = 2
)

// waitForSettingsWrite waits for a settings write to indexName to land, sending
// it again if the wait stops making progress.
//
// A task ID is only meaningful while the index's task queue is the one that
// accepted it. Algolia restarts that queue when another write turns the index
// into a replica, and the ID from before the conversion then never reaches
// published: `getTask` keeps answering notPublished for a write that has, in
// fact, already been applied. Waiting on it alone burns the full budget and
// fails a create that actually succeeded, which is what happens whenever a
// configuration has an algolia_index resource for an index that another index's
// `replicas` list also creates, with no dependency ordering the two.
//
// So a wait that stops progressing re-sends the same settings and follows the new
// task ID instead. Settings writes are idempotent, which is what makes this safe:
// being wrong about the cause costs one redundant write, while being right turns a
// 30-minute hang into a pause of the grace period. A genuinely slow index is
// unaffected beyond those few extra writes, because the re-sent task queues behind
// the work already in flight and the overall budget is unchanged.
func waitForSettingsWrite(ctx context.Context, client *search.APIClient, indexName string, settings *search.IndexSettings, taskID int64) error {
	current := taskID
	reissues := 0
	waitingSince := time.Now()

	return algoliawait.Until(ctx, fmt.Sprintf("settings write to index %q", indexName), func(ctx context.Context) (bool, error) {
		resp, err := client.GetTask(client.NewApiGetTaskRequest(indexName, current), search.WithContext(ctx))
		if err != nil {
			return false, err
		}
		if resp.Status == search.TASK_STATUS_PUBLISHED {
			return true, nil
		}

		if reissues >= writeReissueLimit || time.Since(waitingSince) < writeReissueGrace {
			return false, nil
		}

		tflog.Debug(ctx, "Re-sending a settings write whose task is not publishing", map[string]any{
			"name":   indexName,
			"task":   current,
			"waited": time.Since(waitingSince).String(),
		})

		setResp, err := client.SetSettings(client.NewApiSetSettingsRequest(indexName, settings), search.WithContext(ctx))
		if err != nil {
			return false, err
		}

		current = setResp.TaskID
		reissues++
		waitingSince = time.Now()

		return false, nil
	})
}

// setIndexSettings writes settings to an index and waits for the write to land.
//
// Callers that have to persist state between the write and the wait - index
// Create, which records the index's identity before waiting so a failed wait
// cannot orphan it - call SetSettings and waitForSettingsWrite themselves.
func setIndexSettings(ctx context.Context, client *search.APIClient, indexName string, settings *search.IndexSettings) error {
	setResp, err := client.SetSettings(client.NewApiSetSettingsRequest(indexName, settings), search.WithContext(ctx))
	if err != nil {
		return err
	}

	return waitForSettingsWrite(ctx, client, indexName, settings, setResp.TaskID)
}
