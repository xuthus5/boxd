package main

import (
	"os"
	"path/filepath"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/core"
)

func TestDesktopFirstRunAutostartPreservesExplicitChoice(t *testing.T) {
	for _, tt := range []struct {
		name, body, setting string
		want                bool
	}{
		{name: "new defaults", want: true},
		{name: "restored empty defaults", body: "{}", want: true},
		{name: "explicit disabled", setting: "false"},
		{name: "custom config", body: `{"log":{"level":"warn"}}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			cfg := desktopConfig{Mode: "embedded", DataDir: dir, ConfigPath: filepath.Join(dir, "config.json"), Username: "admin"}
			if tt.body != "" {
				if err := os.WriteFile(cfg.ConfigPath, []byte(tt.body), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if tt.setting != "" {
				seedDesktopAutostart(t, cfg.DataDir, tt.setting)
			}
			rt, err := initRuntime(cfg)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = rt.close() })
			if rt.autostartKernel != tt.want {
				t.Fatalf("want autostart=%v, got %v", tt.want, rt.autostartKernel)
			}
		})
	}
}

func seedDesktopAutostart(t *testing.T, dir, value string) {
	t.Helper()
	db, err := bbolt.Open(filepath.Join(dir, "boxd.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	settings := core.NewSettingsManager(db)
	if err := settings.Set("kernel_autostart", value); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}
