package api

import (
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
	writeJSON(w, http.StatusOK, probeDNSServer(req))
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
	if req.Concurrency <= 0 {
		req.Concurrency = defaultBatchConcurrency
	}

	results := make([]DNSProbeResult, len(req.Items))
	sem := make(chan struct{}, req.Concurrency)
	var wg sync.WaitGroup
	for i, item := range req.Items {
		wg.Add(1)
		go func(idx int, it DNSProbeRequest) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[idx] = probeDNSServer(it)
		}(i, item)
	}
	wg.Wait()
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}
