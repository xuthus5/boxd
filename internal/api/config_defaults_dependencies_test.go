package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
)

type defaultsRestartRecorder struct{ calls int }

func (r *defaultsRestartRecorder) Restart() error {
	r.calls++
	return nil
}

func TestDefaultDNSInstallsDependenciesInOneApply(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, path, map[string]any{})
	restarts := &defaultsRestartRecorder{}
	handler := NewConfigHandler(path, restarts, nil, nil, nil, core.NewDefaultDNSInstaller())
	response := httptest.NewRecorder()
	handler.InstallDefaultDNS(response, httptest.NewRequest(http.MethodPost, "/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("install DNS: %d %s", response.Code, response.Body.String())
	}
	cfg := decodeConfigFile(t, path)
	if cfg["route"].(map[string]any)["final"] != "proxy" {
		t.Fatal("missing proxy final after installing DNS dependencies")
	}
	if restarts.calls != 1 {
		t.Fatalf("DNS and outbounds must be applied together, restarted %d times", restarts.calls)
	}
	if cfg["dns"].(map[string]any)["final"] != "dns-remote" {
		t.Fatal("default DNS must fail closed through the proxy")
	}
}

func TestDefaultDNSRejectsUnsafePolicyWithoutWriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, path, map[string]any{"outbounds": []any{
		map[string]any{"type": "direct", "tag": "proxy"},
	}})
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewConfigHandler(path, nil, nil, nil, nil, core.NewDefaultDNSInstaller())
	response := httptest.NewRecorder()
	handler.InstallDefaultDNS(response, httptest.NewRequest(http.MethodPost, "/", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("want actionable invalid-policy response: %d %s", response.Code, response.Body.String())
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("unsafe installation modified existing configuration: %v", err)
	}
}

func TestDefaultExperimentalUsesApplicationDataDir(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	path := filepath.Join(dir, "config.json")
	writeConfigFile(t, path, map[string]any{})
	handler := NewConfigHandler(path, nil, core.NewLoyalsoldierRuleSetInstaller(dataDir), nil, nil, nil)
	response := httptest.NewRecorder()
	handler.InstallDefaultExperimental(response, httptest.NewRequest(http.MethodPost, "/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("install experimental: %d %s", response.Code, response.Body.String())
	}
	cfg := decodeConfigFile(t, path)
	experimental := cfg["experimental"].(map[string]any)
	cache := experimental["cache_file"].(map[string]any)
	if cache["path"] != filepath.Join(dataDir, "cache.db") {
		t.Fatalf("cache was placed outside the application data directory: %#v", cache)
	}
	clash := experimental["clash_api"].(map[string]any)
	if _, exists := clash["external_controller"]; exists {
		t.Fatal("internal mode control must not open another listener")
	}
}
