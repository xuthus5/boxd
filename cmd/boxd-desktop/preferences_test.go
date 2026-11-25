package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDesktopPreferencesRoundTrip(t *testing.T) {
	directory := t.TempDir()
	missing, err := loadDesktopPreferences(directory)
	if err != nil || missing.isDark() || missing.Theme != "light" {
		t.Fatalf("missing preferences = %+v, %v", missing, err)
	}

	if err := saveDesktopPreferences(directory, desktopPreferences{Theme: preferenceTheme(true)}); err != nil {
		t.Fatalf("saveDesktopPreferences() error = %v", err)
	}
	preferences, err := loadDesktopPreferences(directory)
	if err != nil || !preferences.isDark() || preferences.Theme != "dark" {
		t.Fatalf("loaded preferences = %+v, %v", preferences, err)
	}
	info, err := os.Stat(filepath.Join(directory, desktopPreferencesFile))
	if err != nil {
		t.Fatalf("stat preferences: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("preferences mode = %o, want 600", mode)
	}
}

func TestDesktopPreferencesRejectMalformedFile(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, desktopPreferencesFile), []byte("{"), 0o600); err != nil {
		t.Fatalf("write malformed preferences: %v", err)
	}
	preferences, err := loadDesktopPreferences(directory)
	if err == nil || preferences.Theme != "light" || preferences.isDark() {
		t.Fatalf("malformed preferences = %+v, %v", preferences, err)
	}
}
