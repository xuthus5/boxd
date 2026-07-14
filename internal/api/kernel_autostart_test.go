package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xuthus5/boxui/internal/core"
)

type enabledResp struct {
	Enabled bool `json:"enabled"`
}

func TestKernelAutostartDefaultsToFalse(t *testing.T) {
	db := newTestDB(t)
	h := NewSettingsHandler(core.NewSettingsManager(db))

	rr := httptest.NewRecorder()
	h.GetKernelAutostart(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	resp := decodeBody[enabledResp](t, rr)
	if resp.Enabled {
		t.Fatal("expected default enabled=false")
	}
}

func TestKernelAutostartSetAndGet(t *testing.T) {
	db := newTestDB(t)
	h := NewSettingsHandler(core.NewSettingsManager(db))

	putRR := httptest.NewRecorder()
	body := []byte(`{"enabled":true}`)
	h.SetKernelAutostart(putRR, httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(body)))
	if putRR.Code != http.StatusOK {
		t.Fatalf("set status = %d", putRR.Code)
	}
	resp := decodeBody[enabledResp](t, putRR)
	if !resp.Enabled {
		t.Fatal("expected enabled=true in set response")
	}

	getRR := httptest.NewRecorder()
	h.GetKernelAutostart(getRR, httptest.NewRequest(http.MethodGet, "/", nil))
	if getRR.Code != http.StatusOK {
		t.Fatalf("get status = %d", getRR.Code)
	}
	resp = decodeBody[enabledResp](t, getRR)
	if !resp.Enabled {
		t.Fatal("expected persisted enabled=true")
	}
}

func TestKernelAutostartSetRejectsInvalidBody(t *testing.T) {
	db := newTestDB(t)
	h := NewSettingsHandler(core.NewSettingsManager(db))

	rr := httptest.NewRecorder()
	h.SetKernelAutostart(rr, httptest.NewRequest(http.MethodPut, "/", bytes.NewReader([]byte("not-json"))))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}
