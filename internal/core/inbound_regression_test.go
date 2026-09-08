package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultInstallDoesNotEnableTUN(t *testing.T) {
	result, err := NewDefaultInboundsInstaller().Install(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Inbounds) != 1 {
		t.Fatalf("default installation must only enable mixed, got %#v", result.Inbounds)
	}
	if result.Inbounds[0].(map[string]any)["type"] != "mixed" {
		t.Fatalf("expected usable mixed inbound, got %#v", result.Inbounds)
	}
}

func TestContainerBootstrapUsesReachableMixed(t *testing.T) {
	t.Setenv("BOXD_CONTAINER", "true")
	path := filepath.Join(t.TempDir(), "config.json")
	if created, err := EnsureDefaultConfig(t.Context(), path, filepath.Dir(path)); err != nil || !created {
		t.Fatalf("bootstrap: created=%v, error=%v", created, err)
	}
	cfg := readInboundRegressionConfig(t, path)
	inbounds := cfg["inbounds"].([]any)
	if len(inbounds) != 1 || inbounds[0].(map[string]any)["listen"] != "0.0.0.0" {
		t.Fatalf("container defaults need a reachable mixed listener only: %#v", inbounds)
	}
}

func readInboundRegressionConfig(t *testing.T, path string) map[string]any {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(body, &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}
