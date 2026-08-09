package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/core"
)

func newTestService(t *testing.T) *ServiceSet {
	t.Helper()
	dir := t.TempDir()
	db, err := bbolt.Open(filepath.Join(dir, "test.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	deps := Deps{
		DB:            db,
		ConfigPath:    filepath.Join(dir, "config.json"),
		DataDir:       dir,
		Version:       "test",
		Settings:      core.NewSettingsManager(db),
		ApplyHistory:  core.NewConfigApplyHistoryManager(db),
		RouteMetadata: core.NewRouteRuleMetadataManager(db),
	}
	return New(deps)
}

func writeTestConfig(t *testing.T, path string, value any) {
	t.Helper()
	data := []byte(`{"outbounds":[]}`)
	if value != nil {
		var err error
		if err = func() error { return nil }(); err != nil {
			t.Fatal(err)
		}
		data = mustJSON(t, value)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	return mustMarshalJSON(t, value)
}

func TestServiceSetAccessors(t *testing.T) {
	svc := newTestService(t)
	if svc.Config() == nil {
		t.Fatal("Config() returned nil")
	}
	if svc.Network() == nil {
		t.Fatal("Network() returned nil")
	}
	if svc.Kernel() == nil {
		t.Fatal("Kernel() returned nil")
	}
	if svc.Service() == nil {
		t.Fatal("Service() returned nil")
	}
}

func TestServiceSetTestSettingsURLFallback(t *testing.T) {
	svc := newTestService(t)
	ts := svc.Test()
	if ts.settingsURL == nil {
		t.Fatal("Test() settingsURL function not set from settings manager")
	}
	if got := ts.settingsURL(); got != defaultTestURL {
		t.Fatalf("settingsURL() = %q, want default %q", got, defaultTestURL)
	}
}

func TestServiceSetTestSettingsURLConfigured(t *testing.T) {
	svc := newTestService(t)
	if err := svc.Deps.Settings.Set("url_test", "https://example.test/probe"); err != nil {
		t.Fatal(err)
	}
	ts := svc.Test()
	if got := ts.settingsURL(); got != "https://example.test/probe" {
		t.Fatalf("settingsURL() = %q, want configured URL", got)
	}
}

func TestConfigGetConfigMissing(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.Config().GetConfig(context.Background())
	var de *DomainError
	if !errors.As(err, &de) {
		t.Fatalf("expected DomainError, got %v", err)
	}
	if de.Code != "not_found" {
		t.Fatalf("code = %q", de.Code)
	}
}

func TestConfigGetConfigInvalidJSON(t *testing.T) {
	svc := newTestService(t)
	if err := os.WriteFile(svc.Deps.ConfigPath, []byte("{invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Config().GetConfig(context.Background())
	var de *DomainError
	if !errors.As(err, &de) {
		t.Fatalf("expected DomainError, got %v", err)
	}
	if de.Code != "internal_error" {
		t.Fatalf("code = %q", de.Code)
	}
}

func TestConfigGetConfigSuccess(t *testing.T) {
	svc := newTestService(t)
	writeTestConfig(t, svc.Deps.ConfigPath, map[string]any{"outbounds": []any{}})
	cfg, err := svc.Config().GetConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	m, ok := cfg.(map[string]any)
	if !ok {
		t.Fatalf("cfg type = %T", cfg)
	}
	if _, ok := m["outbounds"]; !ok {
		t.Fatalf("outbounds missing: %v", m)
	}
}

func TestConfigValidateConfigEmpty(t *testing.T) {
	svc := newTestService(t)
	err := svc.Config().ValidateConfig(context.Background(), nil, "validate")
	var de *DomainError
	if !errors.As(err, &de) {
		t.Fatalf("expected DomainError, got %v", err)
	}
	if de.Code != "invalid_request" {
		t.Fatalf("code = %q", de.Code)
	}
}

func TestConfigValidateConfigInvalidJSON(t *testing.T) {
	svc := newTestService(t)
	err := svc.Config().ValidateConfig(context.Background(), []byte("{"), "validate")
	var de *DomainError
	if !errors.As(err, &de) {
		t.Fatalf("expected DomainError, got %v", err)
	}
	if de.Code != "invalid_request" {
		t.Fatalf("code = %q", de.Code)
	}
}

func TestConfigValidateConfigValid(t *testing.T) {
	svc := newTestService(t)
	err := svc.Config().ValidateConfig(context.Background(), []byte(`{"log":{"level":"info"}}`), "validate")
	if err != nil {
		t.Fatal(err)
	}
}

func TestConfigApplyConfigNoInstance(t *testing.T) {
	svc := newTestService(t)
	writeTestConfig(t, svc.Deps.ConfigPath, map[string]any{"outbounds": []any{}})
	result, err := svc.Config().ApplyConfig(context.Background(), []byte(`{"log":{"level":"debug"}}`), "update")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" {
		t.Fatalf("status = %q", result.Status)
	}
	data, readErr := os.ReadFile(svc.Deps.ConfigPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != `{"log":{"level":"debug"}}` {
		t.Fatalf("config = %s", data)
	}
}

type fakeRestart struct {
	errs []error
}

func (f *fakeRestart) Restart() error {
	if len(f.errs) == 0 {
		return nil
	}
	err := f.errs[0]
	f.errs = f.errs[1:]
	return err
}

func TestConfigApplyConfigRollback(t *testing.T) {
	svc := newTestService(t)
	writeTestConfig(t, svc.Deps.ConfigPath, map[string]any{"outbounds": []any{}})
	cfg := newConfig(svc.Deps.ConfigPath, &fakeRestart{errs: []error{errors.New("restart failed"), nil}}, ConfigInstaller{})
	result, err := cfg.ApplyConfig(context.Background(), []byte(`{"log":{"level":"debug"}}`), "update")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "rolled_back" {
		t.Fatalf("status = %q", result.Status)
	}
	if result.APIError == nil || result.APIError.Code != "config_restart_failed" {
		t.Fatalf("apiError = %+v", result.APIError)
	}
	data, readErr := os.ReadFile(svc.Deps.ConfigPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != `{"outbounds":[]}` {
		t.Fatalf("config not rolled back: %s", data)
	}
}

func TestConfigApplyConfigWriteFailure(t *testing.T) {
	dir := t.TempDir()
	cfg := newConfig(filepath.Join(dir, "sub", "config.json"), nil, ConfigInstaller{})
	_, err := cfg.ApplyConfig(context.Background(), []byte(`{"log":{"level":"debug"}}`), "update")
	if err != nil {
		t.Fatal(err)
	}
	blocked := newConfig(dir, nil, ConfigInstaller{})
	_, err = blocked.ApplyConfig(context.Background(), []byte(`{"log":{"level":"debug"}}`), "update")
	if err == nil {
		t.Fatal("expected error when config path is a directory")
	}
}

func TestConfigApplyConfigInvalidRuntime(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.Config().ApplyConfig(context.Background(), []byte(`{"inbounds":[{"type":"http","tag":"http-in","listen":"127.0.0.1","listen_port":8080},{"type":"http","tag":"http-in","listen":"127.0.0.1","listen_port":8081}]}`), "update")
	if err == nil {
		t.Fatal("expected error for duplicate inbound tag")
	}
}
