package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
}
