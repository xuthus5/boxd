package core

import (
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

func TestDefaultUIPreferences(t *testing.T) {
	prefs := DefaultUIPreferences()
	if prefs.Theme != "system" || prefs.Language != "zh" || prefs.MinimumLogLevel != "all" {
		t.Fatalf("unexpected defaults: %+v", prefs)
	}
}

func TestSettingsManagerUIPreferences(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	defer cleanup()

	settings := NewSettingsManager(db)
	prefs, err := settings.UIPreferences()
	if err != nil {
		t.Fatal(err)
	}
	if prefs != DefaultUIPreferences() {
		t.Fatalf("default prefs = %+v", prefs)
	}

	custom := model.UIPreferences{Theme: "dark", Language: "en", MinimumLogLevel: "warn"}
	if err := settings.SetUIPreferences(custom); err != nil {
		t.Fatal(err)
	}
	stored, err := settings.UIPreferences()
	if err != nil {
		t.Fatal(err)
	}
	if stored != custom {
		t.Fatalf("stored prefs = %+v, want %+v", stored, custom)
	}
}

func TestSettingsManagerRejectsInvalidUIPreferences(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	defer cleanup()
	settings := NewSettingsManager(db)

	invalids := []model.UIPreferences{
		{Theme: "blue", Language: "zh", MinimumLogLevel: "all"},
		{Theme: "dark", Language: "fr", MinimumLogLevel: "all"},
		{Theme: "dark", Language: "en", MinimumLogLevel: "fatal"},
	}
	for _, invalid := range invalids {
		if err := settings.SetUIPreferences(invalid); err == nil {
			t.Fatalf("expected rejection for %+v", invalid)
		}
	}

	if err := settings.Set(uiPreferencesKey, "{"); err != nil {
		t.Fatal(err)
	}
	if _, err := settings.UIPreferences(); err == nil {
		t.Fatal("expected decode error")
	}

	if err := settings.Set(uiPreferencesKey, `{"theme":"dark","language":"en","minimumLogLevel":"fatal"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := settings.UIPreferences(); err == nil {
		t.Fatal("expected validation error for stored invalid prefs")
	}
}

func TestSettingsManagerUIPreferencesDatabaseErrors(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	settings := NewSettingsManager(db)
	cleanup()

	if _, err := settings.UIPreferences(); err == nil {
		t.Fatal("expected read error after db close")
	}
	if err := settings.SetUIPreferences(DefaultUIPreferences()); err == nil {
		t.Fatal("expected write error after db close")
	}
}
