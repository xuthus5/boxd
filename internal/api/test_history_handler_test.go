package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/xuthus5/boxd/internal/model"
)

func TestTestHandlerListHistory(t *testing.T) {
	nodeMgr, _, _, _ := newAPIManagers(t)
	handler := NewTestHandler(func() string { return "" }, nodeMgr, nil)

	rr := httptest.NewRecorder()
	handler.ListHistory(rr, httptest.NewRequest(http.MethodGet, "/api/nodes/test-history", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("empty history status = %d", rr.Code)
	}
	empty := decodeBody[map[string]any](t, rr)
	if empty["history"] == nil {
		t.Fatalf("history key missing: %#v", empty)
	}

	result := model.TestResult{Tag: "hk-01", TestType: "tcp", Success: true, LatencyMs: 18, Timestamp: time.Now().UTC()}
	if err := nodeMgr.SaveTestResult("hk-01_tcp", result); err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler.ListHistory(rr, httptest.NewRequest(http.MethodGet, "/api/nodes/test-history?tag=hk-01", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("tag history status = %d body=%s", rr.Code, rr.Body.String())
	}
	one := decodeBody[struct {
		Tag     string                          `json:"tag"`
		History map[string][]model.LatencyPoint `json:"history"`
	}](t, rr)
	if one.Tag != "hk-01" || len(one.History["tcp"]) != 1 {
		t.Fatalf("tag history = %+v", one)
	}
}
