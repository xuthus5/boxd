package api

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

type TestRequest struct {
	Tag      string `json:"tag"`
	TestType string `json:"test_type"`
	Server   string `json:"server"`
	Port     int    `json:"port"`
}

type TestHandler struct {
	settingsURL func() string
	nodeManager *core.NodeManager
	instance    outboundDialer
}

type outboundDialer interface {
	DialOutbound(ctx context.Context, tag, network, addr string) (net.Conn, error)
	OutboundDelay(ctx context.Context, tag, link string, timeout time.Duration) (uint16, error)
}

func NewTestHandler(settingsURLFn func() string, nodeManager *core.NodeManager, instance outboundDialer) *TestHandler {
	return &TestHandler{settingsURL: settingsURLFn, nodeManager: nodeManager, instance: instance}
}

func (h *TestHandler) Run(w http.ResponseWriter, r *http.Request) {
	var req TestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid request")
		return
	}
	if req.Tag == "" || req.TestType == "" {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "tag and test_type are required")
		return
	}

	result, err := h.dispatchTest(r.Context(), req)
	if r.Context().Err() != nil {
		return
	}
	if err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, err.Error())
		return
	}
	if err := h.persistTestResult(result); err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to save test result")
		return
	}
	if r.Context().Err() != nil {
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *TestHandler) persistTestResult(result model.TestResult) error {
	if h.nodeManager == nil {
		return errors.New("node manager not available")
	}
	key := result.Tag + "_" + nonEmpty(result.TestType, "test")
	return h.nodeManager.SaveTestResult(key, result)
}

func (h *TestHandler) ListResults(w http.ResponseWriter, _ *http.Request) {
	all := h.nodeManager.GetAllTestResults()
	writeJSON(w, http.StatusOK, all)
}

// ListHistory GET /api/nodes/test-history[?tag=]
func (h *TestHandler) ListHistory(w http.ResponseWriter, r *http.Request) {
	tag := strings.TrimSpace(r.URL.Query().Get("tag"))
	if tag != "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"tag":     tag,
			"history": h.nodeManager.GetTestHistory(tag),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"history": h.nodeManager.GetAllTestHistory(),
	})
}
