package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xuthus5/boxd/internal/config"
	"github.com/xuthus5/boxd/internal/core"
)

func TestInitializeServerConfigSetsOnlyFirstRunAutostart(t *testing.T) {
	for _, tt := range []struct {
		name, body, setting, want string
	}{
		{name: "new defaults", want: "true"},
		{name: "recovered defaults", body: "{}", want: "true"},
		{name: "custom config", body: `{"log":{"level":"warn"}}`},
		{name: "explicit disabled", setting: "false", want: "false"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{DataDir: t.TempDir(), ConfigPath: filepath.Join(t.TempDir(), "config.json")}
			db, err := openDatabase(cfg.DataDir)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = db.Close() })
			settings := core.NewSettingsManager(db)
			if tt.body != "" {
				if err := os.WriteFile(cfg.ConfigPath, []byte(tt.body), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if tt.setting != "" {
				if err := settings.Set("kernel_autostart", tt.setting); err != nil {
					t.Fatal(err)
				}
			}
			if err := initializeServerConfig(cfg, settings); err != nil {
				t.Fatal(err)
			}
			if got := settings.Get("kernel_autostart"); got != tt.want {
				t.Fatalf("want autostart %q, got %q", tt.want, got)
			}
		})
	}
}

func TestInitializeServerConfigAndSettingsReportErrors(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatal(err)
	}
	settings := core.NewSettingsManager(db)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := initializeServerSettings(&config.Config{Username: "admin"}, settings); err == nil {
		t.Fatal("closed settings database must fail startup")
	}
	if err := initializeServerConfig(&config.Config{DataDir: dir, ConfigPath: dir}, settings); err == nil {
		t.Fatal("invalid config path must fail startup")
	}
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"log":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := initializeServerConfig(&config.Config{DataDir: dir, ConfigPath: path}, settings); err == nil {
		t.Fatal("autostart transaction failure must fail startup")
	}
}
