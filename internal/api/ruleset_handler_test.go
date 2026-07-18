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

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

func TestRuleSetHandlerStatusAndAutoUpdate(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"route":{"rule_set":[{"tag":"geo","type":"remote","url":"https://example.com/a.srs"}]}}`), 0600); err != nil {
		t.Fatal(err)
	}
	db, err := bbolt.Open(filepath.Join(dir, "boxui.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	settings := core.NewSettingsManager(db)
	updater := core.NewRuleSetUpdater(configPath, dir, core.NewLoyalsoldierRuleSetInstaller(dir), nil, nil)
	handler := NewRuleSetHandler(updater, settings)

	rr := httptest.NewRecorder()
	handler.Status(rr, httptest.NewRequest(http.MethodGet, "/api/config/rule-sets/status", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d body=%s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	handler.GetAutoUpdate(rr, httptest.NewRequest(http.MethodGet, "/api/config/rule-sets/auto-update", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("get auto code = %d", rr.Code)
	}

	body, _ := json.Marshal(model.RuleSetAutoUpdate{Enabled: true, Interval: "3h"})
	rr = httptest.NewRecorder()
	handler.SetAutoUpdate(rr, httptest.NewRequest(http.MethodPut, "/api/config/rule-sets/auto-update", bytes.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("set auto code = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRuleSetHandlerNilUpdater(t *testing.T) {
	handler := NewRuleSetHandler(nil, nil)
	rr := httptest.NewRecorder()
	handler.Status(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("code = %d", rr.Code)
	}
}
