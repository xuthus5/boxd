package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func TestUIPreferencesHandlers(t *testing.T) {
	db := newTestDB(t)
	settings := core.NewSettingsManager(db)
	handler := NewSettingsHandler(settings)

	rr := httptest.NewRecorder()
	handler.GetUIPreferences(rr, httptest.NewRequest(http.MethodGet, "/api/settings/preferences", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("get defaults status = %d body = %s", rr.Code, rr.Body.String())
	}
	got := decodeBody[model.UIPreferences](t, rr)
	if got != core.DefaultUIPreferences() {
		t.Fatalf("defaults = %+v", got)
	}

	rr = httptest.NewRecorder()
	handler.SetUIPreferences(rr, jsonRequest(http.MethodPut, "/api/settings/preferences", `{`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid json status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.SetUIPreferences(rr, jsonRequest(http.MethodPut, "/api/settings/preferences",
		`{"theme":"blue","language":"zh","minimumLogLevel":"all"}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid theme status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.SetUIPreferences(rr, jsonRequest(http.MethodPut, "/api/settings/preferences",
		`{"theme":"dark","language":"en","minimumLogLevel":"warn"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("set status = %d body = %s", rr.Code, rr.Body.String())
	}
	saved := decodeBody[model.UIPreferences](t, rr)
	want := model.UIPreferences{Theme: "dark", Language: "en", MinimumLogLevel: "warn"}
	if saved != want {
		t.Fatalf("saved = %+v, want %+v", saved, want)
	}

	rr = httptest.NewRecorder()
	handler.GetUIPreferences(rr, httptest.NewRequest(http.MethodGet, "/api/settings/preferences", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("get saved status = %d", rr.Code)
	}
	loaded := decodeBody[model.UIPreferences](t, rr)
	if loaded != want {
		t.Fatalf("loaded = %+v, want %+v", loaded, want)
	}
}

func TestUIPreferencesHandlerDatabaseErrors(t *testing.T) {
	db := newTestDB(t)
	settings := core.NewSettingsManager(db)
	handler := NewSettingsHandler(settings)
	_ = db.Close()

	rr := httptest.NewRecorder()
	handler.GetUIPreferences(rr, httptest.NewRequest(http.MethodGet, "/api/settings/preferences", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("get closed db status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.SetUIPreferences(rr, jsonRequest(http.MethodPut, "/api/settings/preferences",
		`{"theme":"dark","language":"en","minimumLogLevel":"info"}`))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("set closed db status = %d", rr.Code)
	}
}

func TestUIPreferencesRoundTripJSON(t *testing.T) {
	// ensure response envelope carries camelCase fields expected by the UI
	prefs := model.UIPreferences{Theme: "light", Language: "zh", MinimumLogLevel: "debug"}
	raw, err := json.Marshal(prefs)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"theme":"light","language":"zh","minimumLogLevel":"debug"}` {
		t.Fatalf("json = %s", raw)
	}
}
