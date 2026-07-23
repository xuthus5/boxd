package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func setupApplyHistoryHandler(t *testing.T) (*ConfigHandler, *core.ConfigApplyHistoryManager, string) {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"log":{"level":"info"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	db, err := bbolt.Open(filepath.Join(dir, "boxd.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	history := core.NewConfigApplyHistoryManager(db)
	handler := NewConfigHandlerWithHistory(configPath, nil, nil, nil, nil, nil, history)
	return handler, history, configPath
}

func decodeApplyHistoryEvents(t *testing.T, body []byte) []model.ConfigApplyEvent {
	t.Helper()
	var envelope struct {
		Data struct {
			Events []model.ConfigApplyEvent `json:"events"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode envelope: %v body=%s", err, body)
	}
	return envelope.Data.Events
}

func TestListConfigApplyHistoryEmpty(t *testing.T) {
	handler, _, _ := setupApplyHistoryHandler(t)
	rr := httptest.NewRecorder()
	handler.ListConfigApplyHistory(rr, httptest.NewRequest(http.MethodGet, "/api/config/apply-history", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	events := decodeApplyHistoryEvents(t, rr.Body.Bytes())
	if len(events) != 0 {
		t.Fatalf("events = %+v", events)
	}
}

func TestListConfigApplyHistoryNilManager(t *testing.T) {
	handler := NewConfigHandler(filepath.Join(t.TempDir(), "missing.json"), nil, nil, nil, nil, nil)
	rr := httptest.NewRecorder()
	handler.ListConfigApplyHistory(rr, httptest.NewRequest(http.MethodGet, "/api/config/apply-history", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestUpdateConfigRecordsApplyHistory(t *testing.T) {
	handler, history, _ := setupApplyHistoryHandler(t)
	body := []byte(`{
  "log": {"level": "warn"},
  "inbounds": [],
  "outbounds": [{"type": "direct", "tag": "direct"}],
  "route": {"final": "direct"}
}`)
	req := httptest.NewRequest(http.MethodPut, "/api/config/", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	handler.UpdateConfig(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	events, err := history.List(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %+v", events)
	}
	if events[0].Source != "update" || events[0].Status != "applied" {
		t.Fatalf("event = %+v", events[0])
	}

	listRR := httptest.NewRecorder()
	handler.ListConfigApplyHistory(listRR, httptest.NewRequest(http.MethodGet, "/api/config/apply-history", nil))
	if listRR.Code != http.StatusOK {
		t.Fatalf("list status = %d", listRR.Code)
	}
	listed := decodeApplyHistoryEvents(t, listRR.Body.Bytes())
	if len(listed) != 1 || listed[0].Source != "update" {
		t.Fatalf("list payload = %+v", listed)
	}
}

func TestUpdateRawConfigRecordsApplyHistory(t *testing.T) {
	handler, history, _ := setupApplyHistoryHandler(t)
	body := []byte(`{
  "log": {"level": "info"},
  "outbounds": [{"type": "direct", "tag": "direct"}],
  "route": {"final": "direct"}
}`)
	req := httptest.NewRequest(http.MethodPut, "/api/config/raw", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	handler.UpdateRawConfig(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	events, err := history.List(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Source != "raw" {
		t.Fatalf("events = %+v", events)
	}
}
