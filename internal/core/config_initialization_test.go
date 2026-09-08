package core

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDefaultConfigUsesDataDirAndPreservesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "etc", "config.json")
	dataDir := filepath.Join(dir, "data")
	if created, err := EnsureDefaultConfig(t.Context(), path, dataDir); err != nil || !created {
		t.Fatalf("initialize: created=%v, err=%v", created, err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(body, &cfg); err != nil {
		t.Fatal(err)
	}
	experimental := cfg["experimental"].(map[string]any)
	cache := experimental["cache_file"].(map[string]any)
	if cache["path"] != filepath.Join(dataDir, "cache.db") {
		t.Fatalf("cache must use the configured data directory: %#v", cache)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if created, err := EnsureDefaultConfig(t.Context(), path, filepath.Join(dir, "unused")); err != nil || created {
		t.Fatalf("repeat: created=%v, err=%v", created, err)
	}
	after, err := os.Stat(path)
	if err != nil || !before.ModTime().Equal(after.ModTime()) {
		t.Fatalf("existing configuration was rewritten: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "unused")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("repeat initialization touched another data directory: %v", err)
	}
}

func TestEnsureDefaultConfigCancellationAndFailureLeaveNoConfig(t *testing.T) {
	for _, name := range []string{"cancelled", "invalid data directory"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.json")
			dataDir := filepath.Join(dir, "data")
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)
			if name == "cancelled" {
				cancel()
			} else if err := os.WriteFile(dataDir, nil, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := EnsureDefaultConfig(ctx, path, dataDir); err == nil {
				t.Fatal("expected initialization failure")
			}
			if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failed initialization published a config: %v", err)
			}
		})
	}
}

func TestEncodeInitialConfigRejectsInvalidConfiguration(t *testing.T) {
	for _, tt := range []struct {
		name string
		cfg  map[string]any
	}{
		{name: "unencodable", cfg: map[string]any{"bad": func() {}}},
		{name: "unknown sing-box field", cfg: map[string]any{"unknown_field": true}},
		{name: "dangling outbound", cfg: map[string]any{"route": map[string]any{"final": "missing"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := encodeInitialConfig(tt.cfg); err == nil {
				t.Fatal("expected validation error before publishing configuration")
			}
		})
	}
}

func TestEnsureConfigFileInstallsCompleteDefaultPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "config.json")
	created, err := EnsureConfigFile(path)
	if err != nil || !created {
		t.Fatalf("initialization: created=%v, err=%v", created, err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(body, &cfg); err != nil {
		t.Fatal(err)
	}
	dns, _ := cfg["dns"].(map[string]any)
	if dns["final"] != "dns-remote" {
		t.Fatalf("default DNS must use the proxy resolver: %#v", dns)
	}
	route, _ := cfg["route"].(map[string]any)
	if route["final"] != "proxy" {
		t.Fatalf("default route must use proxy: %#v", route["final"])
	}
	ruleSets, _ := route["rule_set"].([]any)
	if len(ruleSets) != 4 {
		t.Fatalf("want four usable rule sets, got %d", len(ruleSets))
	}
	for _, value := range ruleSets {
		entry, _ := value.(map[string]any)
		file, _ := entry["path"].(string)
		if entry["type"] != "local" || file == "" {
			t.Fatalf("rule set requires local data: %#v", entry)
		}
		if _, err := os.Stat(file); err != nil {
			t.Fatalf("missing rule set: %v", err)
		}
	}
}

func TestEnsureConfigFileCreatesPrivateDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "config.json")
	if _, err := EnsureConfigFile(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0700 {
		t.Fatalf("directory permissions: want 0700, got %o", info.Mode().Perm())
	}
}

func TestEnsureConfigFileRejectsDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.Symlink(filepath.Join(dir, "missing.json"), path); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := EnsureConfigFile(path); err == nil {
		t.Fatal("initialization must not replace a dangling configuration symlink")
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("configuration symlink was replaced: %v", err)
	}
}
