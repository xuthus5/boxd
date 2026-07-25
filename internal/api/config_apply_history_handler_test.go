package api

import (
	"bytes"
	"encoding/json"
	"errors"
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

func TestMarkCurrentConfigApplyEvent(t *testing.T) {
	body := []byte(`{"log":{"level":"info"}}`)
	events := []model.ConfigApplyEvent{
		{Status: model.ConfigApplyStatusApplied, Hash: core.ConfigBodyHash(body)},
		{Status: model.ConfigApplyStatusRolledBack, Hash: core.ConfigBodyHash(body)},
	}
	missing := filepath.Join(t.TempDir(), "missing.json")
	markCurrentConfigApplyEvent(events, missing)
	if events[0].Current || events[1].Current {
		t.Fatalf("missing config marked current: %+v", events)
	}
	if configSnapshotIsCurrent(missing, body) {
		t.Fatal("missing config should not match snapshot")
	}
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	markCurrentConfigApplyEvent(events, configPath)
	if !events[0].Current || events[1].Current {
		t.Fatalf("current flags = %+v", events)
	}
	if !configSnapshotIsCurrent(configPath, body) || configSnapshotIsCurrent(configPath, []byte(`{}`)) {
		t.Fatal("snapshot current detection mismatch")
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
	if len(listed) != 1 || listed[0].Source != "update" || !listed[0].Current {
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

func TestRestoreConfigSnapshot(t *testing.T) {
	handler, history, configPath := setupApplyHistoryHandler(t)
	body := []byte(`{"log":{"level":"warn"},"outbounds":[{"type":"direct","tag":"direct"}],"route":{"final":"direct"}}`)
	update := httptest.NewRecorder()
	handler.UpdateConfig(update, jsonRequest(http.MethodPut, "/api/config", string(body)))
	if update.Code != http.StatusOK {
		t.Fatalf("update status = %d body=%s", update.Code, update.Body.String())
	}
	events, err := history.List(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || !events[0].Restorable {
		t.Fatalf("events = %+v", events)
	}
	if err := os.WriteFile(configPath, []byte(`{"log":{"level":"info"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	restore := httptest.NewRecorder()
	req := withURLParam(httptest.NewRequest(http.MethodPost, "/api/config/apply-history/restore/restore", nil), "id", events[0].ID)
	handler.RestoreConfig(restore, req)
	if restore.Code != http.StatusOK {
		t.Fatalf("restore status = %d body=%s", restore.Code, restore.Body.String())
	}
	envelope := decodeEnvelope(t, restore)
	if envelope.Status != model.StatusOK {
		t.Fatalf("restore envelope = %+v", envelope)
	}
	if got, err := os.ReadFile(configPath); err != nil {
		t.Fatal(err)
	} else if string(got) != string(body) {
		t.Fatalf("restored config = %s, want %s", got, body)
	}
	events, err = history.List(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Source != "restore" {
		t.Fatalf("restored history = %+v", events)
	}
}

func TestRestoreConfigSnapshotAlreadyCurrent(t *testing.T) {
	handler, history, _ := setupApplyHistoryHandler(t)
	body := `{"log":{"level":"warn"},"outbounds":[{"type":"direct","tag":"direct"}],"route":{"final":"direct"}}`
	update := httptest.NewRecorder()
	handler.UpdateConfig(update, jsonRequest(http.MethodPut, "/api/config", body))
	if update.Code != http.StatusOK {
		t.Fatalf("update status = %d body=%s", update.Code, update.Body.String())
	}
	events, err := history.List(5)
	if err != nil || len(events) != 1 {
		t.Fatalf("events = %+v err=%v", events, err)
	}
	rr := httptest.NewRecorder()
	req := withURLParam(httptest.NewRequest(http.MethodPost, "/api/config/apply-history/current/restore", nil), "id", events[0].ID)
	handler.RestoreConfig(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	data := decodeBody[map[string]any](t, rr)
	if data["already_current"] != true || data["restored"] != false {
		t.Fatalf("data = %+v", data)
	}
	events, err = history.List(5)
	if err != nil || len(events) != 1 {
		t.Fatalf("history after no-op = %+v err=%v", events, err)
	}
}

func TestRestoreConfigSnapshotNotFound(t *testing.T) {
	handler, _, _ := setupApplyHistoryHandler(t)
	rr := httptest.NewRecorder()
	req := withURLParam(httptest.NewRequest(http.MethodPost, "/api/config/apply-history/missing/restore", nil), "id", "missing")
	handler.RestoreConfig(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRestoreConfigSnapshotWithoutHistory(t *testing.T) {
	handler := NewConfigHandler(filepath.Join(t.TempDir(), "config.json"), nil, nil, nil, nil, nil)
	rr := httptest.NewRecorder()
	req := withURLParam(httptest.NewRequest(http.MethodPost, "/api/config/apply-history/missing/restore", nil), "id", "missing")
	handler.RestoreConfig(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRestoreConfigSnapshotRejectsInvalidBody(t *testing.T) {
	handler, history, _ := setupApplyHistoryHandler(t)
	event := core.NewConfigApplyEvent("update", model.StatusOK, []byte("not-json"), nil)
	if err := history.AppendSnapshot(event, []byte("not-json")); err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	req := withURLParam(httptest.NewRequest(http.MethodPost, "/api/config/apply-history/invalid/restore", nil), "id", event.ID)
	handler.RestoreConfig(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRestoreConfigSnapshotRollsBackFailedRestart(t *testing.T) {
	handler, history, configPath := setupApplyHistoryHandler(t)
	handler.instance = &fakeRestartable{errs: []error{errors.New("restart failed"), nil}}
	body := []byte(`{"log":{"level":"warn"},"outbounds":[{"type":"direct","tag":"direct"}],"route":{"final":"direct"}}`)
	event := core.NewConfigApplyEvent("update", model.StatusOK, body, nil)
	if err := history.AppendSnapshot(event, body); err != nil {
		t.Fatal(err)
	}
	previous, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	req := withURLParam(httptest.NewRequest(http.MethodPost, "/api/config/apply-history/failed/restore", nil), "id", event.ID)
	handler.RestoreConfig(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	envelope := decodeEnvelope(t, rr)
	if envelope.Status != model.StatusRolledBack {
		t.Fatalf("envelope = %+v", envelope)
	}
	current, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(current, previous) {
		t.Fatalf("current config = %s, want previous %s", current, previous)
	}
}
