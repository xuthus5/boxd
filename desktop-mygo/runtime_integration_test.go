package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/internal/core"

	"github.com/xuthus5/boxd/desktop-mygo/internal/runtime"
)

func TestRuntimeBackendSnapshotWithTemporaryData(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	cfg := runtime.Config{
		Mode: "embedded", DataDir: filepath.Join(dir, "data"),
		ConfigPath: filepath.Join(dir, "config", "config.json"),
		Username:   "admin", RefreshInterval: 60,
	}
	rt, err := runtime.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := rt.Close(); err != nil {
			t.Errorf("close runtime: %v", err)
		}
	})
	for path, wantMode := range map[string]os.FileMode{
		cfg.DataDir:                           0o700,
		filepath.Join(cfg.DataDir, "boxd.db"): 0o600,
		cfg.ConfigPath:                        0o600,
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if info.Mode().Perm() != wantMode {
			t.Errorf("%s mode = %o, want %o", path, info.Mode().Perm(), wantMode)
		}
	}
	backend := newRuntimeBackend(rt)
	snap, err := backend.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snap.Status == nil || !snap.LogsLoaded || !snap.PreferencesLoaded {
		t.Fatalf("snapshot = %+v", snap)
	}
	text, err := backend.LoadConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if text == "" || !strings.Contains(text, `"log"`) {
		t.Fatalf("loaded config = %q", text)
	}
	if err := backend.ValidateConfig(context.Background(), text); err != nil {
		t.Fatalf("validate config: %v", err)
	}
	bundle, err := backend.SupportBundle(context.Background())
	if err != nil {
		t.Fatalf("collect support bundle: %v", err)
	}
	for _, source := range []string{"version", "service", "memory", "config_diagnostics", "rule_sets", "subscriptions", "network"} {
		if bundle.Requests[source].Status != "ok" {
			t.Errorf("support bundle source %s = %+v", source, bundle.Requests[source])
		}
	}
}

func TestRuntimeBackendSetupPreviewAndApply(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	cfg := runtime.Config{
		Mode: "embedded", DataDir: filepath.Join(dir, "data"),
		ConfigPath: filepath.Join(dir, "config", "config.json"),
		Username:   "admin", RefreshInterval: 60,
	}
	rt, err := runtime.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := rt.Close(); err != nil {
			t.Errorf("close runtime: %v", err)
		}
	})
	backend := newRuntimeBackend(rt)
	ctx := context.Background()
	source, err := os.ReadFile(cfg.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	sourceHash := core.ConfigContentHash(source)
	status, err := backend.SetupStatus(ctx)
	if err != nil || status.SourceHash != sourceHash {
		t.Fatalf("setup status = %+v err=%v", status, err)
	}
	request := core.SetupRequest{Modules: []string{"outbounds"}, InboundMode: "preserve"}
	plan, err := backend.PreviewSetup(ctx, request)
	if err != nil || plan.SourceHash != sourceHash || len(plan.Modules) == 0 {
		t.Fatalf("setup preview = %+v err=%v", plan, err)
	}
	afterPreview, err := os.ReadFile(cfg.ConfigPath)
	if err != nil || core.ConfigContentHash(afterPreview) != sourceHash {
		t.Fatalf("preview modified configuration: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.DataDir, "config-backups")); !os.IsNotExist(err) {
		t.Fatalf("preview created backups: %v", err)
	}
	request.SourceHash = plan.SourceHash
	result, err := backend.ApplySetup(ctx, request)
	if err != nil || result.Status != "ok" || result.RolledBack {
		t.Fatalf("setup apply = %+v err=%v", result, err)
	}
	applied, err := os.ReadFile(cfg.ConfigPath)
	if err != nil || core.ConfigContentHash(applied) != result.ConfigHash {
		t.Fatalf("applied configuration hash = %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.DataDir, "config-backups", "before-setup-"+sourceHash+".json")); err != nil {
		t.Fatalf("setup backup missing: %v", err)
	}
	if _, err := backend.ApplySetup(ctx, request); err == nil {
		t.Fatal("stale setup preview was accepted")
	}
}
