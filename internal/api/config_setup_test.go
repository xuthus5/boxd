package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

type setupKernel struct {
	running bool
	reloads int
	fail    bool
}

func (s *setupKernel) Status() model.ServiceStatus { return model.ServiceStatus{Running: s.running} }

func (s *setupKernel) Restart() error { s.running = true; return nil }

func (s *setupKernel) Reload() error {
	if !s.running {
		return nil
	}
	s.reloads++
	if s.fail {
		return &core.InboundProfileError{Code: "test_reload_failed", Message: "test reload failed"}
	}
	return nil
}

func setupHandler(t *testing.T, body string) (*ConfigHandler, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	return NewConfigHandler(path, nil, core.NewLoyalsoldierRuleSetInstaller(filepath.Join(dir, "data")),
		core.NewDefaultOutboundsInstaller(), core.NewDefaultRouteInstaller(), core.NewDefaultDNSInstaller()), path
}

func TestSetupPreviewAndApplyPreserveStoppedKernel(t *testing.T) {
	handler, path := setupHandler(t, `{}`)
	kernel := &setupKernel{}
	handler.instance = kernel
	preview := httptest.NewRecorder()
	handler.PreviewSetup(preview, jsonRequest(http.MethodPost, "/", `{"inbound_mode":"proxy"}`))
	if preview.Code != http.StatusOK {
		t.Fatalf("preview: %d %s", preview.Code, preview.Body.String())
	}
	var envelope struct {
		Data core.SetupPlan `json:"data"`
	}
	if err := json.Unmarshal(preview.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil || string(before) != `{}` {
		t.Fatalf("preview wrote config: %v", err)
	}
	request, err := json.Marshal(core.SetupRequest{InboundMode: "proxy", SourceHash: envelope.Data.SourceHash})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ApplySetup(response, jsonRequest(http.MethodPost, "/", string(request)))
	if response.Code != http.StatusOK || kernel.running || kernel.reloads != 0 {
		t.Fatalf("apply changed stopped state: %d %s, %#v", response.Code, response.Body.String(), kernel)
	}
	after := decodeConfigFile(t, path)
	if len(after["inbounds"].([]any)) != 1 || after["dns"].(map[string]any)["final"] != "dns-remote" {
		t.Fatal("required modules were not applied together")
	}
}

func TestSetupRejectsStalePreviewBeforeWritingAssets(t *testing.T) {
	handler, path := setupHandler(t, `{"log":{"level":"debug"}}`)
	request, err := json.Marshal(core.SetupRequest{SourceHash: core.ConfigContentHash([]byte(`{}`))})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ApplySetup(response, jsonRequest(http.MethodPost, "/", string(request)))
	if response.Code != http.StatusConflict {
		t.Fatalf("stale preview: %d %s", response.Code, response.Body.String())
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != `{"log":{"level":"debug"}}` {
		t.Fatal("stale preview overwrote a newer config")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "data")); !os.IsNotExist(err) {
		t.Fatal("stale preview created assets")
	}
}

func TestSetupResetBacksUpOriginalBytes(t *testing.T) {
	const original = "broken JSON with user content\n"
	handler, path := setupHandler(t, original)
	request, err := json.Marshal(core.SetupRequest{ResetInvalid: true, SourceHash: core.ConfigContentHash([]byte(original))})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ApplySetup(response, jsonRequest(http.MethodPost, "/", string(request)))
	if response.Code != http.StatusOK {
		t.Fatalf("explicit reset: %d %s", response.Code, response.Body.String())
	}
	backup := filepath.Join(filepath.Dir(path), "data", "config-backups", "before-setup-"+core.ConfigContentHash([]byte(original))+".json")
	body, err := os.ReadFile(backup)
	if err != nil || !bytes.Equal(body, []byte(original)) {
		t.Fatalf("original config backup: %v", err)
	}
}

func TestSetupFailedReloadRestoresConfig(t *testing.T) {
	handler, path := setupHandler(t, `{}`)
	kernel := &setupKernel{running: true, fail: true}
	handler.instance = kernel
	request, err := json.Marshal(core.SetupRequest{SourceHash: core.ConfigContentHash([]byte(`{}`))})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ApplySetup(response, jsonRequest(http.MethodPost, "/", string(request)))
	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || result.Status != "rolled_back" || !kernel.running {
		t.Fatalf("reload failure: %d %s", response.Code, response.Body.String())
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != `{}` {
		t.Fatalf("original config was not restored: %v", err)
	}
}
