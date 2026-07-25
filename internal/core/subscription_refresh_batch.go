package core

import (
	"context"
	"errors"
	"sync"

	"github.com/xuthus5/boxd/internal/model"
)

const subscriptionRefreshConcurrency = 4

type subscriptionRefreshJob struct {
	index        int
	subscription model.Subscription
}

type subscriptionRefreshResult struct {
	index   int
	failure SubscriptionRefreshFailure
}

func (m *SubscriptionManager) refreshSubscriptionsConcurrently(
	ctx context.Context,
	subs []model.Subscription,
) []SubscriptionRefreshFailure {
	jobs := make(chan subscriptionRefreshJob)
	results := make(chan subscriptionRefreshResult, len(subs))
	workerCount := min(len(subs), subscriptionRefreshConcurrency)
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			m.refreshSubscriptionWorker(ctx, jobs, results)
			workers.Done()
		}()
	}

dispatch:
	for index, subscription := range subs {
		if ctx.Err() != nil {
			break
		}
		select {
		case <-ctx.Done():
			break dispatch
		case jobs <- subscriptionRefreshJob{index: index, subscription: subscription}:
		}
	}
	close(jobs)
	workers.Wait()
	close(results)
	return collectSubscriptionRefreshFailures(len(subs), results)
}

func (m *SubscriptionManager) refreshSubscriptionWorker(
	ctx context.Context,
	jobs <-chan subscriptionRefreshJob,
	results chan<- subscriptionRefreshResult,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			if ctx.Err() != nil {
				continue
			}
			err := m.RefreshContext(ctx, job.subscription.ID)
			if err == nil {
				continue
			}
			if ctxErr := ctx.Err(); ctxErr != nil && errors.Is(err, ctxErr) {
				continue
			}
			classified := classifySubscriptionRefreshError(err)
			results <- subscriptionRefreshResult{
				index: job.index,
				failure: SubscriptionRefreshFailure{
					ID:      job.subscription.ID,
					Name:    job.subscription.Name,
					Code:    classified.Code,
					Message: err.Error(),
				},
			}
		}
	}
}

func collectSubscriptionRefreshFailures(
	subscriptionCount int,
	results <-chan subscriptionRefreshResult,
) []SubscriptionRefreshFailure {
	ordered := make([]SubscriptionRefreshFailure, subscriptionCount)
	for result := range results {
		ordered[result.index] = result.failure
	}
	failures := make([]SubscriptionRefreshFailure, 0, subscriptionCount)
	for _, failure := range ordered {
		if failure.Code != "" {
			failures = append(failures, failure)
		}
	}
	return failures
}
