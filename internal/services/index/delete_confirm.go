package index

import (
	"context"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/terraform-provider-algolia/internal/algoliaerr"
	"github.com/algolia/terraform-provider-algolia/internal/algoliawait"
)

// deleteConfirmBudget bounds the wait for a deleted index to actually disappear.
//
// Deliberately short. Measured against the live API, an index is gone the instant
// its delete task reports published - the confirming read 404s with no delay at
// all - so this budget is not there to absorb normal latency. It exists so that
// the abnormal case fails in a few seconds instead of hanging: an index observed
// surviving a published delete stayed for at least 206 seconds, so waiting longer
// would not have saved that destroy, it would only have delayed the news.
// A var rather than a const only so that a test driving Delete end to end can
// lower it; production code never assigns to it.
var deleteConfirmBudget = 20 * time.Second

// confirmIndexDeleted reports whether an index is really gone once its delete task
// has published.
//
// A published task is not proof the index went away. Algolia refuses to delete an
// index that is a destination of an A/B test, and while that refusal is usually a
// 403 on the delete itself, the association can outlive the test that created it -
// leaving a delete that is accepted, queued, published, and has no effect. The
// destroy then reports success, Terraform drops the resource from state, and an
// index nothing tracks keeps existing, keeps costing money, and keeps answering
// queries.
//
// A stray 404 is what success looks like here, so the read is inverted: anything
// other than "not found" means the index is still there.
func confirmIndexDeleted(ctx context.Context, client *search.APIClient, indexName string) error {
	return confirmIndexDeletedWithin(ctx, client, indexName, deleteConfirmBudget)
}

// confirmIndexDeletedWithin is confirmIndexDeleted with a caller-chosen budget, so
// a test can exercise the give-up path without sleeping out the real one.
func confirmIndexDeletedWithin(ctx context.Context, client *search.APIClient, indexName string, budget time.Duration) error {
	return algoliawait.Within(ctx, "deletion of index "+indexName, budget, func(ctx context.Context) (bool, error) {
		_, err := client.GetSettings(client.NewApiGetSettingsRequest(indexName), search.WithContext(ctx))
		if err != nil {
			if algoliaerr.IsNotFound(err) {
				return true, nil
			}

			// Any other failure is about the read, not the deletion, and says
			// nothing either way - keep waiting rather than claim the index
			// survived on the strength of a transient error.
			return false, nil
		}

		return false, nil
	})
}

// deleteNotConfirmedDetail explains an index that outlived its own delete task.
func deleteNotConfirmedDetail(indexName string, err error) string {
	return "Algolia accepted the request to delete index " + indexName + " and reported the task as " +
		"finished, but the index is still there: " + err.Error() + ".\n\n" +
		"The usual cause is an A/B test association that outlived the test itself - Algolia will not " +
		"delete an index that is a destination of one, and the association can linger after the test " +
		"is gone. Check for an A/B test referencing this index, and try the destroy again once it has " +
		"cleared.\n\n" +
		"The resource has been left in Terraform state, because removing it would leave an index " +
		"running that nothing manages."
}
