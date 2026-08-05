package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureConfigFileCreatesWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "config.json")

	created, err := EnsureConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected created=true")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("generated config is not valid JSON: %v", err)
	}
	if cfg["inbounds"] == nil || cfg["outbounds"] == nil || cfg["route"] == nil {
		t.Fatalf("generated config missing required sections: %#v", cfg)
	}
	// 权限应为 0600（安全要求）。
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Fatalf("config perm = %v, want 0600", perm)
	}
}

func TestEnsureConfigFileKeepsExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"existing": true}`), 0600); err != nil {
		t.Fatal(err)
	}

	created, err := EnsureConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("expected created=false for existing file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"existing": true}` {
		t.Fatalf("existing config was modified: %s", data)
	}
}

func TestEnsureConfigFileRejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	if _, err := EnsureConfigFile(dir); err == nil {
		t.Fatal("expected error when path is a directory")
	}
}

func TestMinimalConfigTemplate(t *testing.T) {
	cfg := minimalConfigTemplate()
	if _, err := json.Marshal(cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["log"] == nil {
		t.Fatal("expected log section")
	}
	inbounds, _ := cfg["inbounds"].([]any)
	outbounds, _ := cfg["outbounds"].([]any)
	if len(inbounds) == 0 || len(outbounds) == 0 {
		t.Fatal("expected inbounds and outbounds entries")
	}
}
