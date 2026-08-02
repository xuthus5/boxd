package service

import (
	"context"
	"errors"
	"sync"

	"github.com/xuthus5/boxd/internal/model"
)

// defaultTestURL 默认测速 URL。
var defaultTestURL = "https://cp.cloudflare.com/"

// TestBatchRequest 批量测速请求。
type TestBatchRequest struct {
	Items       []TestRequest `json:"items"`
	Concurrency int           `json:"concurrency"`
}

const (
	maxBatchItems       = 128
	maxBatchConcurrency = 32
)

func normalizeTestBatchRequest(req *TestBatchRequest) error {
	if len(req.Items) == 0 {
		return errors.New("items is empty")
	}
	if len(req.Items) > maxBatchItems {
		return errors.New("too many test items")
	}
	if req.Concurrency <= 0 {
		req.Concurrency = defaultBatchConcurrency
	}
	if req.Concurrency > maxBatchConcurrency {
		req.Concurrency = maxBatchConcurrency
	}
	return nil
}

type testBatchJob struct {
	index int
	item  TestRequest
}

func (s *TestService) runTestBatch(
	ctx context.Context,
	items []TestRequest,
	concurrency int,
) []model.TestResult {
	results := make([]model.TestResult, len(items))
	jobs := make(chan testBatchJob)
	var workers sync.WaitGroup
	workerCount := min(max(concurrency, 1), len(items))
	for range workerCount {
		workers.Go(func() { s.runTestBatchWorker(ctx, jobs, results) })
	}

dispatch:
	for index, item := range items {
		select {
		case <-ctx.Done():
			break dispatch
		case jobs <- testBatchJob{index: index, item: item}:
		}
	}
	close(jobs)
	workers.Wait()
	return results
}

func (s *TestService) runTestBatchWorker(
	ctx context.Context,
	jobs <-chan testBatchJob,
	results []model.TestResult,
) {
	for {
		if ctx.Err() != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok || ctx.Err() != nil {
				return
			}
			result, completed := s.runTestBatchItem(ctx, job.item)
			if !completed {
				return
			}
			results[job.index] = result
		}
	}
}

func (s *TestService) runTestBatchItem(ctx context.Context, item TestRequest) (model.TestResult, bool) {
	item.Tag = firstNonEmpty(item.Tag, item.Server)
	result, err := s.dispatchTest(ctx, item)
	if err != nil {
		result = failedTestResult(err.Error(), err)
		result.Tag = item.Tag
		result.TestType = item.TestType
	}
	if ctx.Err() != nil {
		return result, false
	}
	if s.nodeManager != nil {
		_ = s.persistTestResult(result)
	}
	return result, ctx.Err() == nil
}
