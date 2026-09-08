package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xuthus5/boxd/internal/core"
)

func TestPreparedConfigCASRejectsStaleBeforePreparation(t *testing.T) {
	svc := newTestService(t)
	writeTestConfig(t, svc.Deps.ConfigPath, map[string]any{})
	prepared := false
	_, err := svc.Config().writePreparedConfig(t.Context(), []byte(`{"log":{}}`), configWriteOptions{
		ExpectedHash: "stale", Prepare: func() error { prepared = true; return nil },
	})
	if !errors.Is(err, core.ErrSetupStale) || prepared {
		t.Fatalf("stale source reached preparation: prepared=%v, error=%v", prepared, err)
	}
	assertServiceSetupSource(t, svc.Deps.ConfigPath, "{}")
}

func TestPreparedConfigUsesTheSharedPathLock(t *testing.T) {
	svc := newTestService(t)
	unlock := core.LockConfig(svc.Deps.ConfigPath)
	defer func() { unlock() }()
	finished := make(chan error, 1)
	go func() {
		_, err := svc.Config().ApplyConfig(t.Context(), []byte("{}"), "test")
		finished <- err
	}()
	select {
	case err := <-finished:
		t.Fatalf("config write bypassed the path lock: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	unlock()
	unlock = func() {}
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("config writer did not release its locks")
	}
}

func TestPreparedConfigCancellationPreventsWrites(t *testing.T) {
	svc := newTestService(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := svc.Config().writePreparedConfig(ctx, []byte("{}"), configWriteOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled write: %v", err)
	}
	if _, err := svc.Config().prepareConfigWrite(ctx, configWriteOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled preparation: %v", err)
	}
	ctx, cancel = context.WithCancel(t.Context())
	defer cancel()
	_, err := svc.Config().writePreparedConfig(ctx, []byte("{}"), configWriteOptions{Prepare: func() error { cancel(); return nil }})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled after preparation: %v", err)
	}
	if _, err := os.Stat(svc.Deps.ConfigPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled preparation wrote config: %v", err)
	}
}

func TestConfigReloadFailureRestoresBeforeExplicitRestart(t *testing.T) {
	svc := newTestService(t)
	writeTestConfig(t, svc.Deps.ConfigPath, map[string]any{})
	probe := &configReloadProbe{running: true, reloadErr: errors.New("new config failed")}
	cfg := newConfig(svc.Deps.ConfigPath, svc.Deps.DataDir, probe, ConfigInstaller{})
	result, err := cfg.ApplyConfig(t.Context(), []byte(`{"log":{}}`), "test")
	if err != nil || !result.RolledBack || probe.reloads != 1 || probe.restarts != 1 {
		t.Fatalf("rollback did not explicitly recover the kernel: %+v, %+v, %v", result, probe, err)
	}
	assertServiceSetupSource(t, cfg.path, "{}")
	probe.restartErr = errors.New("recovery failed")
	_, err = cfg.ApplyConfig(t.Context(), []byte(`{"log":{}}`), "test")
	assertSetupDomainError(t, err, "500/config_restart_failed")
	if !strings.Contains(err.Error(), "new config failed") || !strings.Contains(err.Error(), "recovery failed") {
		t.Fatalf("reload diagnostics were lost: %v", err)
	}
}

func TestPreparedConfigReportsFilesystemAndPreparationFailures(t *testing.T) {
	svc := newTestService(t)
	cause := errors.New("backup failed")
	_, err := svc.Config().writePreparedConfig(t.Context(), []byte("{}"), configWriteOptions{Prepare: func() error { return cause }})
	if !errors.Is(err, cause) {
		t.Fatalf("prepare failure was lost: %v", err)
	}
	cfg := newConfig(filepath.Join(svc.Deps.DataDir, "sub", "config.json"), "", nil, ConfigInstaller{})
	_, err = cfg.writePreparedConfig(t.Context(), []byte("{}"), configWriteOptions{Prepare: func() error {
		return os.WriteFile(filepath.Dir(cfg.path), []byte("block directory creation"), 0600)
	}})
	assertSetupDomainError(t, err, "500/internal_error")
	bad := newConfig(svc.Deps.DataDir, "", nil, ConfigInstaller{})
	_, err = bad.ApplyConfig(t.Context(), []byte("{}"), "test")
	assertSetupDomainError(t, err, "500/internal_error")
}
