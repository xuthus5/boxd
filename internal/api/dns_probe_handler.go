package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/xuthus5/boxd/internal/model"
)

// ProbeDNS POST /api/runtime/dns/probe — 探测单个 DNS 服务器。
func (h *RuntimeHandler) ProbeDNS(w http.ResponseWriter, r *http.Request) {
	var req DNSProbeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid request")
		return
	}
	result := probeDNSServer(r.Context(), req)
	if r.Context().Err() != nil {
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ProbeDNSBatch POST /api/runtime/dns/probe-batch — 并发探测多个 DNS 服务器。
func (h *RuntimeHandler) ProbeDNSBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Items       []DNSProbeRequest `json:"items"`
		Concurrency int               `json:"concurrency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid request")
		return
	}
	if len(req.Items) == 0 {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "items is empty")
		return
	}
	if len(req.Items) > maxDNSProbeItems {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "too many dns probe items")
		return
	}
	if req.Concurrency <= 0 {
		req.Concurrency = defaultBatchConcurrency
	}
	if req.Concurrency > maxDNSProbeConcurrency {
		req.Concurrency = maxDNSProbeConcurrency
	}

	results, completed := runDNSProbeBatch(r.Context(), req.Items, req.Concurrency)
	if !completed {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func runDNSProbeBatch(ctx context.Context, items []DNSProbeRequest, concurrency int) ([]DNSProbeResult, bool) {
	results := make([]DNSProbeResult, len(items))
	jobs := make(chan int)
	var wg sync.WaitGroup
	workerCount := min(max(concurrency, 1), len(items))
	for range workerCount {
		wg.Go(func() {
			for {
				if ctx.Err() != nil {
					return
				}
				select {
				case <-ctx.Done():
					return
				case index, ok := <-jobs:
					if !ok {
						return
					}
					results[index] = probeDNSServer(ctx, items[index])
				}
			}
		})
	}
	completed := true
sendJobs:
	for index := range items {
		select {
		case <-ctx.Done():
			completed = false
			break sendJobs
		case jobs <- index:
		}
	}
	close(jobs)
	wg.Wait()
	return results, completed && ctx.Err() == nil
}
