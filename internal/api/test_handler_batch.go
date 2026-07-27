package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/xuthus5/boxd/internal/model"
)

const (
	maxBatchItems       = 128
	maxBatchConcurrency = 32
)

var defaultBatchConcurrency = 8

type testBatchRequest struct {
	Items       []TestRequest `json:"items"`
	Concurrency int           `json:"concurrency"`
}

type testBatchJob struct {
	index int
	item  TestRequest
}

// RunBatch POST /api/nodes/test-batch：固定数量 worker 并发测速。
func (h *TestHandler) RunBatch(w http.ResponseWriter, r *http.Request) {
	var req testBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid request")
		return
	}
	if err := normalizeTestBatchRequest(&req); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, err.Error())
		return
	}

	results, completed := h.runTestBatch(r.Context(), req.Items, req.Concurrency)
	if !completed {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func normalizeTestBatchRequest(req *testBatchRequest) error {
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

func (h *TestHandler) runTestBatch(
	ctx context.Context,
	items []TestRequest,
	concurrency int,
) ([]model.TestResult, bool) {
	results := make([]model.TestResult, len(items))
	jobs := make(chan testBatchJob)
	var workers sync.WaitGroup
	workerCount := min(max(concurrency, 1), len(items))
	for range workerCount {
		workers.Go(func() { h.runTestBatchWorker(ctx, jobs, results) })
	}

	completed := true
dispatch:
	for index, item := range items {
		select {
		case <-ctx.Done():
			completed = false
			break dispatch
		case jobs <- testBatchJob{index: index, item: item}:
		}
	}
	close(jobs)
	workers.Wait()
	return results, completed && ctx.Err() == nil
}

func (h *TestHandler) runTestBatchWorker(
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
			result, completed := h.runTestBatchItem(ctx, job.item)
			if !completed {
				return
			}
			results[job.index] = result
		}
	}
}

func (h *TestHandler) runTestBatchItem(ctx context.Context, item TestRequest) (model.TestResult, bool) {
	item.Tag = firstNonEmpty(item.Tag, item.Server)
	result, err := h.dispatchTest(ctx, item)
	if err != nil {
		result = failedTestResult(err.Error(), err)
		result.Tag = item.Tag
		result.TestType = item.TestType
	}
	if ctx.Err() != nil {
		return result, false
	}
	if h.nodeManager != nil {
		_ = h.persistTestResult(result)
	}
	return result, ctx.Err() == nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func nonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
