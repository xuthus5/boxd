package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func TestGetConfigSnapshot(t *testing.T) {
	handler, history, _ := setupApplyHistoryHandler(t)
	body := []byte(`{"outbounds":[{"type":"direct","tag":"direct"}]}`)
	event := core.NewConfigApplyEvent("update", model.StatusOK, body, nil)
	if err := history.AppendSnapshot(event, body); err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/config/apply-history/snapshot/snapshot", nil), "id", event.ID)
	handler.GetConfigSnapshot(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	got := decodeBody[map[string]any](t, rr)
	if got["outbounds"] == nil {
		t.Fatalf("snapshot = %+v", got)
	}
}

func TestGetConfigSnapshotNotFound(t *testing.T) {
	handler, _, _ := setupApplyHistoryHandler(t)
	rr := httptest.NewRecorder()
	handler.GetConfigSnapshot(rr, withURLParam(httptest.NewRequest(http.MethodGet, "/snapshot", nil), "id", "missing"))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestGetConfigSnapshotRejectsInvalidData(t *testing.T) {
	handler, history, _ := setupApplyHistoryHandler(t)
	tests := []struct {
		name string
		body []byte
	}{
		{name: "malformed", body: []byte("not-json")},
		{name: "array", body: []byte(`["not","object"]`)},
		{name: "null", body: []byte("null")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := core.NewConfigApplyEvent("update", model.StatusOK, test.body, nil)
			if err := history.AppendSnapshot(event, test.body); err != nil {
				t.Fatal(err)
			}
			rr := httptest.NewRecorder()
			handler.GetConfigSnapshot(rr, withURLParam(httptest.NewRequest(http.MethodGet, "/snapshot", nil), "id", event.ID))
			if rr.Code != http.StatusInternalServerError {
				t.Fatalf("body = %q status = %d", test.body, rr.Code)
			}
		})
	}
}

func TestGetConfigSnapshotStoreError(t *testing.T) {
	db, err := bbolt.Open(filepath.Join(t.TempDir(), "closed.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	history := core.NewConfigApplyHistoryManager(db)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	handler := NewConfigHandlerWithHistory("config.json", nil, nil, nil, nil, nil, history)
	rr := httptest.NewRecorder()
	handler.GetConfigSnapshot(rr, withURLParam(httptest.NewRequest(http.MethodGet, "/snapshot", nil), "id", "event"))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestGetConfigSnapshotWithoutHistory(t *testing.T) {
	handler := NewConfigHandler(filepath.Join(t.TempDir(), "config.json"), nil, nil, nil, nil, nil)
	rr := httptest.NewRecorder()
	handler.GetConfigSnapshot(rr, withURLParam(httptest.NewRequest(http.MethodGet, "/snapshot", nil), "id", "missing"))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rr.Code)
	}
}
