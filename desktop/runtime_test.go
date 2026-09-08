package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultDesktopConfig(t *testing.T) {
	cfg := defaultDesktopConfig()
	if cfg.Mode != "embedded" {
		t.Fatalf("mode = %q", cfg.Mode)
	}
	if cfg.RemoteURL != "http://127.0.0.1:9091" {
		t.Fatalf("remote url = %q", cfg.RemoteURL)
	}
	if cfg.Username != "admin" {
		t.Fatalf("username = %q", cfg.Username)
	}
	if cfg.RefreshInterval != 60 {
		t.Fatalf("refresh interval = %d", cfg.RefreshInterval)
	}
	if cfg.DataDir == "" || cfg.ConfigPath == "" {
		t.Fatal("data dir and config path must not be empty")
	}
}

func TestParseDesktopConfigDefaults(t *testing.T) {
	for _, key := range []string{
		"BOXD_DESKTOP_MODE", "BOXD_REMOTE_URL", "BOXD_DATA_DIR",
		"BOXD_CONFIG", "BOXD_USERNAME", "BOXD_PASSWORD",
	} {
		t.Setenv(key, "")
	}
	cfg := parseDesktopConfig()
	if cfg.Mode != "embedded" {
		t.Fatalf("mode = %q", cfg.Mode)
	}
}

func TestParseDesktopConfigRemoteMode(t *testing.T) {
	t.Setenv("BOXD_DESKTOP_MODE", "remote")
	t.Setenv("BOXD_REMOTE_URL", "http://example.com:9091")
	cfg := parseDesktopConfig()
	if cfg.Mode != "remote" {
		t.Fatalf("mode = %q", cfg.Mode)
	}
	if cfg.RemoteURL != "http://example.com:9091" {
		t.Fatalf("remote url = %q", cfg.RemoteURL)
	}
}

func TestParseDesktopConfigDataDir(t *testing.T) {
	t.Setenv("BOXD_DATA_DIR", "/custom/data")
	t.Setenv("BOXD_CONFIG", "/custom/config.json")
	t.Setenv("BOXD_USERNAME", "user1")
	t.Setenv("BOXD_PASSWORD", "pass1")
	cfg := parseDesktopConfig()
	if cfg.DataDir != "/custom/data" {
		t.Fatalf("data dir = %q", cfg.DataDir)
	}
	if cfg.ConfigPath != "/custom/config.json" {
		t.Fatalf("config path = %q", cfg.ConfigPath)
	}
	if cfg.Username != "user1" {
		t.Fatalf("username = %q", cfg.Username)
	}
	if cfg.Password != "pass1" {
		t.Fatalf("password = %q", cfg.Password)
	}
}

func TestInitRuntimeEmbedded(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	cfg := desktopConfig{
		Mode:            "embedded",
		DataDir:         filepath.Join(dir, "data"),
		ConfigPath:      filepath.Join(dir, "config", "config.json"),
		Username:        "admin",
		Password:        "",
		RefreshInterval: 60,
	}
	rt, err := initRuntime(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rt.close() }()
	if rt.svc == nil {
		t.Fatal("service set is nil")
	}
	if rt.instance == nil {
		t.Fatal("instance is nil")
	}
	if _, err := os.Stat(filepath.Join(dir, "data", "boxd.db")); err != nil {
		t.Fatalf("db not created: %v", err)
	}
	if _, err := os.Stat(cfg.ConfigPath); err != nil {
		t.Fatalf("default config not created: %v", err)
	}
	assertInitialDesktopPolicy(t, cfg)
}

func TestInitRuntimeEmbeddedPreservesExistingConfig(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	configPath := filepath.Join(dir, "config", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"log":{"level":"debug"}}`)
	if err := os.WriteFile(configPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	cfg := desktopConfig{
		Mode:            "embedded",
		DataDir:         filepath.Join(dir, "data"),
		ConfigPath:      configPath,
		Username:        "admin",
		Password:        "",
		RefreshInterval: 60,
	}
	rt, err := initRuntime(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rt.close() }()
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config after init: %v", err)
	}
	if string(after) != string(body) {
		t.Fatalf("existing config was modified: got %q", string(after))
	}
}

func assertInitialDesktopPolicy(t *testing.T, cfg desktopConfig) {
	t.Helper()
	body, err := os.ReadFile(cfg.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	var generated map[string]any
	if err := json.Unmarshal(body, &generated); err != nil {
		t.Fatal(err)
	}
	dns, _ := generated["dns"].(map[string]any)
	route, _ := generated["route"].(map[string]any)
	if dns["final"] != "dns-remote" || route["final"] != "proxy" {
		t.Fatalf("initial DNS/routing policy is incomplete: %#v", generated)
	}
	experimental, _ := generated["experimental"].(map[string]any)
	cache, _ := experimental["cache_file"].(map[string]any)
	if cache["path"] != filepath.Join(cfg.DataDir, "cache.db") {
		t.Fatalf("initial cache must use the configured data directory: %#v", cache)
	}
	assertInitialDesktopRuleSets(t, route)
}

func assertInitialDesktopRuleSets(t *testing.T, route map[string]any) {
	t.Helper()
	rules, _ := route["rule_set"].([]any)
	want := map[string]bool{
		"loyalsoldier-direct": true, "loyalsoldier-proxy": true,
		"loyalsoldier-reject": true, "geoip-cn": true,
	}
	for _, item := range rules {
		entry, _ := item.(map[string]any)
		tag, _ := entry["tag"].(string)
		if !want[tag] || entry["type"] != "local" {
			t.Fatalf("unexpected initial rule set: %#v", entry)
		}
		delete(want, tag)
		path, _ := entry["path"].(string)
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			t.Fatalf("initial rule set was not installed: %q, %v", path, err)
		}
	}
	if len(want) != 0 {
		t.Fatalf("initial rule sets are missing: %#v", want)
	}
}

func TestInitRuntimeRemoteMode(t *testing.T) {
	cfg := desktopConfig{Mode: "remote", RemoteURL: "http://127.0.0.1:9091"}
	rt, err := initRuntime(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if rt.svc != nil {
		t.Fatal("remote mode should not init svc")
	}
	if err := rt.close(); err != nil {
		t.Fatal(err)
	}
}

func TestStartStopKernelNilRuntime(t *testing.T) {
	if err := startKernel(nil); err != nil {
		t.Fatal(err)
	}
	if err := stopKernel(nil); err != nil {
		t.Fatal(err)
	}
	if err := restartKernel(nil); err != nil {
		t.Fatal(err)
	}
}

func TestErrNotReady(t *testing.T) {
	err := errNotReady()
	if err == nil {
		t.Fatal("err not ready should return error")
	}
	if err.Error() != "desktop runtime is not ready" {
		t.Fatalf("err = %q", err.Error())
	}
}
