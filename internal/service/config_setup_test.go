package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
)

func TestConfigSetupPreviewIsReadOnly(t *testing.T) {
	svc := newTestService(t)
	const source = `{"log":{"level":"warn"}}`
	if err := os.WriteFile(svc.Deps.ConfigPath, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	status, err := svc.Config().GetSetup(t.Context())
	if err != nil || status.SourceHash != core.ConfigContentHash([]byte(source)) {
		t.Fatalf("setup status: %+v, %v", status, err)
	}
	plan, err := svc.Config().PreviewSetup(t.Context(), core.SetupRequest{InboundMode: "proxy"})
	if err != nil || plan.SourceHash != status.SourceHash || len(plan.Modules) != 6 {
		t.Fatalf("setup preview: %+v, %v", plan, err)
	}
	assertServiceSetupSource(t, svc.Config().path, source)
	for _, name := range []string{"rule-sets", "config-backups"} {
		if _, err := os.Stat(filepath.Join(svc.Deps.DataDir, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("preview wrote %s: %v", name, err)
		}
	}
}

func TestConfigSetupApplyBacksUpAndInstallsDependencies(t *testing.T) {
	svc := newTestService(t)
	const source = `{"log":{"level":"warn"}}`
	if err := os.WriteFile(svc.Deps.ConfigPath, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	request := core.SetupRequest{InboundMode: "proxy", SourceHash: core.ConfigContentHash([]byte(source))}
	result, err := svc.Config().ApplySetup(t.Context(), request)
	if err != nil || result.Status != "ok" || result.RolledBack || len(result.Modules) != 6 {
		t.Fatalf("apply setup: %+v, %v", result, err)
	}
	body, err := os.ReadFile(svc.Deps.ConfigPath)
	if err != nil || core.ConfigContentHash(body) != result.ConfigHash {
		t.Fatalf("applied hash does not match persisted config: %v", err)
	}
	backup := filepath.Join(svc.Deps.DataDir, "config-backups", "before-setup-"+request.SourceHash+".json")
	assertServiceSetupSource(t, backup, source)
	entries, err := os.ReadDir(filepath.Join(svc.Deps.DataDir, "rule-sets"))
	if err != nil || len(entries) < 4 {
		t.Fatalf("setup dependencies were not installed: %v, %v", entries, err)
	}
}

func TestConfigSetupRequiresPreviewAndRejectsStaleInvalidSource(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.Config().ApplySetup(t.Context(), core.SetupRequest{})
	assertSetupDomainError(t, err, "400/preview_required")
	request := core.SetupRequest{SourceHash: core.ConfigContentHash(nil)}
	if err := os.WriteFile(svc.Deps.ConfigPath, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Config().ApplySetup(t.Context(), request)
	assertSetupDomainError(t, err, "409/config_changed")
	assertServiceSetupSource(t, svc.Deps.ConfigPath, "{")
}

func TestConfigSetupInvalidJSONNeedsExplicitReset(t *testing.T) {
	svc := newTestService(t)
	const source = "{ damaged JSON"
	if err := os.WriteFile(svc.Deps.ConfigPath, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	request := core.SetupRequest{InboundMode: "proxy", SourceHash: core.ConfigContentHash([]byte(source))}
	_, err := svc.Config().PreviewSetup(t.Context(), request)
	assertSetupDomainError(t, err, "400/config_invalid")
	_, err = svc.Config().ApplySetup(t.Context(), request)
	assertSetupDomainError(t, err, "400/config_invalid")
	request.ResetInvalid = true
	result, err := svc.Config().ApplySetup(t.Context(), request)
	if err != nil || result.Status != "ok" {
		t.Fatalf("explicit reset failed: %+v, %v", result, err)
	}
	backup := filepath.Join(svc.Deps.DataDir, "config-backups", "before-setup-"+request.SourceHash+".json")
	assertServiceSetupSource(t, backup, source)
}

func TestConfigSetupFailsBeforeWriteWhenPreparationFails(t *testing.T) {
	for _, blocked := range []string{"config-backups", "rule-sets"} {
		t.Run(blocked, func(t *testing.T) {
			svc := newTestService(t)
			writeTestConfig(t, svc.Deps.ConfigPath, map[string]any{})
			if err := os.WriteFile(filepath.Join(svc.Deps.DataDir, blocked), []byte("blocked"), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := svc.Config().ApplySetup(t.Context(), core.SetupRequest{InboundMode: "proxy", SourceHash: core.ConfigContentHash([]byte("{}"))})
			assertSetupDomainError(t, err, "500/internal_error")
			assertServiceSetupSource(t, svc.Deps.ConfigPath, "{}")
		})
	}
}

func TestConfigSetupReportsInputReadAndCancellationErrors(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.Config().PreviewSetup(t.Context(), core.SetupRequest{Modules: []string{"unknown"}})
	assertSetupDomainError(t, err, "400/invalid_setup_module")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := svc.Config().GetSetup(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("status ignored cancellation: %v", err)
	}
	if _, err := svc.Config().PreviewSetup(ctx, core.SetupRequest{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("preview ignored cancellation: %v", err)
	}
	svc.Config().path = svc.Deps.DataDir
	_, err = svc.Config().GetSetup(t.Context())
	assertSetupDomainError(t, err, "500/internal_error")
	_, err = svc.Config().PreviewSetup(t.Context(), core.SetupRequest{})
	assertSetupDomainError(t, err, "500/internal_error")
	_, err = svc.Config().ApplySetup(t.Context(), core.SetupRequest{SourceHash: "old"})
	assertSetupDomainError(t, err, "500/internal_error")
}

func TestConfigSetupEnvironmentAndDomainErrors(t *testing.T) {
	probe := &configReloadProbe{running: true}
	cfg := newConfig(filepath.Join(t.TempDir(), "config.json"), "", probe, ConfigInstaller{})
	if env := cfg.setupEnvironment(); !env.KernelRunning || env.DataDir != filepath.Dir(cfg.path) {
		t.Fatalf("unexpected setup environment: %+v", env)
	}
	assertSetupDomainError(t, setupDomainError(&core.InboundProfileError{Code: "tun_unavailable", Message: "no device"}), "400/tun_unavailable")
	assertSetupDomainError(t, setupDomainError(&ErrInvalidRuntime{msg: "invalid config"}), "400/config_invalid_runtime")
	_, err := cfg.applySetupPlan(t.Context(), &core.SetupPlan{Config: map[string]any{"invalid": func() {}}}, filepath.Dir(cfg.path))
	assertSetupDomainError(t, err, "500/internal_error")
}

func assertSetupDomainError(t *testing.T, err error, expected string) {
	t.Helper()
	var domain *DomainError
	if !errors.As(err, &domain) || fmt.Sprintf("%d/%s", domain.Status, domain.Code) != expected {
		t.Fatalf("want %s, got %v", expected, err)
	}
}

func assertServiceSetupSource(t *testing.T, path, expected string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil || string(body) != expected {
		t.Fatalf("expected source %q, got %q: %v", expected, body, err)
	}
}
