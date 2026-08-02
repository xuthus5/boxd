package service

import (
	"context"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func newTestSettings(t *testing.T) *SettingsService {
	t.Helper()
	db := newTestDB(t)
	settings := core.NewSettingsManager(db)
	if _, err := settings.EnsureAdminCredential("admin", ""); err != nil {
		t.Fatal(err)
	}
	return NewSettingsService(settings, "admin")
}

func TestSettingsPasswordStatus(t *testing.T) {
	svc := newTestSettings(t)
	status, err := svc.GetPasswordStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !status["defaultPassword"] {
		t.Fatal("expected default password")
	}
}

func TestSettingsChangePassword(t *testing.T) {
	svc := newTestSettings(t)
	result, err := svc.ChangePassword(context.Background(), "admin123", "new-password")
	if err != nil {
		t.Fatal(err)
	}
	if !result["changed"] {
		t.Fatal("expected changed")
	}
}

func TestSettingsChangePasswordWrongCurrent(t *testing.T) {
	svc := newTestSettings(t)
	_, err := svc.ChangePassword(context.Background(), "wrong", "new-password")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSettingsChangePasswordWeak(t *testing.T) {
	svc := newTestSettings(t)
	_, err := svc.ChangePassword(context.Background(), "admin123", "x")
	if err == nil {
		t.Fatal("expected weak password error")
	}
}

func TestSettingsTestURL(t *testing.T) {
	svc := newTestSettings(t)
	result, err := svc.GetTestURL(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result["url"] != defaultTestURL {
		t.Fatalf("url = %q", result["url"])
	}
	if err := svc.settings.Set("url_test", "invalid url"); err != nil {
		t.Fatal(err)
	}
	result, err = svc.GetTestURL(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result["url"] != defaultTestURL {
		t.Fatalf("url = %q", result["url"])
	}
}

func TestSettingsSetTestURL(t *testing.T) {
	svc := newTestSettings(t)
	result, err := svc.SetTestURL(context.Background(), "https://example.com/")
	if err != nil {
		t.Fatal(err)
	}
	if result["url"] != "https://example.com/" {
		t.Fatalf("url = %q", result["url"])
	}
	result, err = svc.SetTestURL(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if result["url"] != defaultTestURL {
		t.Fatalf("url = %q", result["url"])
	}
}

func TestSettingsSetTestURLInvalid(t *testing.T) {
	svc := newTestSettings(t)
	_, err := svc.SetTestURL(context.Background(), "not a url")
	if err == nil {
		t.Fatal("expected error for invalid url")
	}
}

func TestSettingsURLTestDefaults(t *testing.T) {
	svc := newTestSettings(t)
	config, err := svc.GetURLTestDefaults(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_ = config

	enabled := true
	url := "https://cp.cloudflare.com/"
	interval := "3m"
	tolerance := uint16(50)
	updated, err := svc.SetURLTestDefaults(context.Background(), URLTestDefaultsInput{
		Enabled: &enabled, URL: &url, Interval: &interval, Tolerance: &tolerance,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !updated.Enabled {
		t.Fatal("expected enabled")
	}
}

func TestSettingsSetURLTestDefaultsMissing(t *testing.T) {
	svc := newTestSettings(t)
	_, err := svc.SetURLTestDefaults(context.Background(), URLTestDefaultsInput{})
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
}

func TestSettingsSetURLTestDefaultsInvalid(t *testing.T) {
	svc := newTestSettings(t)
	enabled := true
	url := "not a url"
	interval := "3m"
	tolerance := uint16(50)
	_, err := svc.SetURLTestDefaults(context.Background(), URLTestDefaultsInput{
		Enabled: &enabled, URL: &url, Interval: &interval, Tolerance: &tolerance,
	})
	if err == nil {
		t.Fatal("expected error for invalid defaults")
	}
}

func TestSettingsKernelAutostart(t *testing.T) {
	svc := newTestSettings(t)
	result, err := svc.SetKernelAutostart(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if !result["enabled"] {
		t.Fatal("expected enabled")
	}
	got, err := svc.GetKernelAutostart(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !got["enabled"] {
		t.Fatal("expected enabled")
	}
}

func TestSettingsUIPreferences(t *testing.T) {
	svc := newTestSettings(t)
	prefs := model.UIPreferences{Theme: "dark", Language: "zh", MinimumLogLevel: "info"}
	saved, err := svc.SetUIPreferences(context.Background(), prefs)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Theme != "dark" {
		t.Fatalf("theme = %q", saved.Theme)
	}
	got, err := svc.GetUIPreferences(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Language != "zh" {
		t.Fatalf("language = %q", got.Language)
	}
}

func TestSettingsSetUIPreferencesInvalid(t *testing.T) {
	svc := newTestSettings(t)
	_, err := svc.SetUIPreferences(context.Background(), model.UIPreferences{Theme: "neon"})
	if err == nil {
		t.Fatal("expected error for invalid theme")
	}
}

func TestSettingsJWTSecret(t *testing.T) {
	svc := newTestSettings(t)
	info, err := svc.GetJWTSecret(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if info["present"].(bool) {
		t.Fatal("secret should not be present initially")
	}
	updated, err := svc.SetJWTSecret(context.Background(), "my-secret-key-12345")
	if err != nil {
		t.Fatal(err)
	}
	if updated["length"] != 19 {
		t.Fatalf("length = %v", updated["length"])
	}
	info, err = svc.GetJWTSecret(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !info["present"].(bool) {
		t.Fatal("secret should be present")
	}
}

func TestSettingsSetJWTSecretEmpty(t *testing.T) {
	svc := newTestSettings(t)
	_, err := svc.SetJWTSecret(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty secret")
	}
}

func TestMaskJWTSecret(t *testing.T) {
	if got := maskJWTSecret(""); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := maskJWTSecret("ab"); got != "********" {
		t.Fatalf("got %q", got)
	}
	if got := maskJWTSecret("abcdef"); !strings.Contains(got, "****") {
		t.Fatalf("got %q", got)
	}
}
