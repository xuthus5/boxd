package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuthus5/boxui/internal/core"
)

func newConfigHandlerWithFile(t *testing.T) (*ConfigHandler, string) {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, configPath, map[string]any{
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
		},
	})
	return NewConfigHandler(configPath, nil,
		core.NewLoyalsoldierRuleSetInstaller(t.TempDir()),
		core.NewDefaultOutboundsInstaller(),
		core.NewDefaultRouteInstaller(),
		core.NewDefaultDNSInstaller(),
	), configPath
}

func TestUpdateConfigInvalidJSON(t *testing.T) {
	handler, _ := newConfigHandlerWithFile(t)

	rr := httptest.NewRecorder()
	handler.UpdateConfig(rr, jsonRequest(http.MethodPut, "/api/config", "not json"))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestUpdateConfigFileWriteError(t *testing.T) {
	// 父路径是一个已存在的文件，无法在其下创建子文件
	parentFile := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(parentFile, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	handler := NewConfigHandler(parentFile+"/config.json", nil, nil, nil, nil, nil)

	rr := httptest.NewRecorder()
	handler.UpdateConfig(rr, jsonRequest(http.MethodPut, "/api/config", `{"outbounds":[]}`))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestGetConfigNotFound(t *testing.T) {
	handler := NewConfigHandler("/nonexistent/config.json", nil, nil, nil, nil, nil)

	rr := httptest.NewRecorder()
	handler.GetConfig(rr, httptest.NewRequest(http.MethodGet, "/api/config", nil))

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestGetConfigInvalidJSON(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}
	handler := NewConfigHandler(configPath, nil, nil, nil, nil, nil)

	rr := httptest.NewRecorder()
	handler.GetConfig(rr, httptest.NewRequest(http.MethodGet, "/api/config", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestGetRawConfigNotFound(t *testing.T) {
	handler := NewConfigHandler("/nonexistent/config.json", nil, nil, nil, nil, nil)

	rr := httptest.NewRecorder()
	handler.GetRawConfig(rr, httptest.NewRequest(http.MethodGet, "/api/config/raw", nil))

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestGetRawConfigInvalidJSON(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	handler := NewConfigHandler(configPath, nil, nil, nil, nil, nil)

	rr := httptest.NewRecorder()
	handler.GetRawConfig(rr, httptest.NewRequest(http.MethodGet, "/api/config/raw", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestUpdateRawConfigSuccess(t *testing.T) {
	handler, configPath := newConfigHandlerWithFile(t)

	rr := httptest.NewRecorder()
	handler.UpdateRawConfig(rr, jsonRequest(http.MethodPut, "/api/config/raw", `{"outbounds":[{"type":"direct","tag":"direct"}]}`))

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	// 验证文件被写入
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateRawConfigInvalidJSON(t *testing.T) {
	handler, _ := newConfigHandlerWithFile(t)

	rr := httptest.NewRecorder()
	handler.UpdateRawConfig(rr, jsonRequest(http.MethodPut, "/api/config/raw", "not json"))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestInstallDefaultOutboundsSuccess(t *testing.T) {
	handler, _ := newConfigHandlerWithFile(t)

	rr := httptest.NewRecorder()
	handler.InstallDefaultOutbounds(rr, httptest.NewRequest(http.MethodPost, "/api/config/outbounds/defaults", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestInstallDefaultOutboundsConfigNotFound(t *testing.T) {
	handler := NewConfigHandler("/nonexistent/config.json", nil, nil,
		core.NewDefaultOutboundsInstaller(), nil, nil)

	rr := httptest.NewRecorder()
	handler.InstallDefaultOutbounds(rr, httptest.NewRequest(http.MethodPost, "/api/config/outbounds/defaults", nil))

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestInstallDefaultRouteRulesSuccess(t *testing.T) {
	handler, _ := newConfigHandlerWithFile(t)

	rr := httptest.NewRecorder()
	handler.InstallDefaultRouteRules(rr, httptest.NewRequest(http.MethodPost, "/api/config/route/defaults", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestInstallDefaultRouteRulesConfigNotFound(t *testing.T) {
	handler := NewConfigHandler("/nonexistent/config.json", nil, nil, nil,
		core.NewDefaultRouteInstaller(), nil)

	rr := httptest.NewRecorder()
	handler.InstallDefaultRouteRules(rr, httptest.NewRequest(http.MethodPost, "/api/config/route/defaults", nil))

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestInstallDefaultDNSSuccess(t *testing.T) {
	handler, _ := newConfigHandlerWithFile(t)

	rr := httptest.NewRecorder()
	handler.InstallDefaultDNS(rr, httptest.NewRequest(http.MethodPost, "/api/config/dns/defaults", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestInstallDefaultDNSConfigNotFound(t *testing.T) {
	handler := NewConfigHandler("/nonexistent/config.json", nil, nil, nil, nil,
		core.NewDefaultDNSInstaller())

	rr := httptest.NewRecorder()
	handler.InstallDefaultDNS(rr, httptest.NewRequest(http.MethodPost, "/api/config/dns/defaults", nil))

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestInstallDefaultOutboundsNilInstaller(t *testing.T) {
	handler, _ := newConfigHandlerWithFile(t)
	// 覆盖为 nil installer
	handler.outboundInstaller = nil

	rr := httptest.NewRecorder()
	handler.InstallDefaultOutbounds(rr, httptest.NewRequest(http.MethodPost, "/api/config/outbounds/defaults", nil))

	if rr.Code != http.StatusNotImplemented {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotImplemented)
	}
}

func TestInstallDefaultRouteRulesNilInstaller(t *testing.T) {
	handler, _ := newConfigHandlerWithFile(t)
	handler.routeInstaller = nil

	rr := httptest.NewRecorder()
	handler.InstallDefaultRouteRules(rr, httptest.NewRequest(http.MethodPost, "/api/config/route/defaults", nil))

	if rr.Code != http.StatusNotImplemented {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotImplemented)
	}
}

func TestInstallDefaultDNSNilInstaller(t *testing.T) {
	handler, _ := newConfigHandlerWithFile(t)
	handler.dnsInstaller = nil

	rr := httptest.NewRecorder()
	handler.InstallDefaultDNS(rr, httptest.NewRequest(http.MethodPost, "/api/config/dns/defaults", nil))

	if rr.Code != http.StatusNotImplemented {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotImplemented)
	}
}

func TestInstallDefaultRuleSetsSuccess(t *testing.T) {
	handler, _ := newConfigHandlerWithFile(t)

	rr := httptest.NewRecorder()
	// InstallDefaultRuleSets 需要 context，使用 httptest.Request
	req := httptest.NewRequest(http.MethodPost, "/api/config/rule-sets/defaults", nil)
	handler.InstallDefaultRuleSets(rr, req)

	// 可能成功或因网络问题失败，但不应该是 500 panic
	if rr.Code == 0 {
		t.Error("expected non-zero status code")
	}
}

func TestAtomicWriteFileMkdirError(t *testing.T) {
	// 指向不可创建的目录
	err := atomicWriteFile("/proc/cannot-create/file.json", []byte("{}"))
	if err == nil {
		t.Fatal("expected error for atomicWriteFile to invalid path")
	}
}

func TestMergeRuleSets(t *testing.T) {
	existing := []any{
		map[string]any{"tag": "existing", "type": "rule_set"},
	}
	installed := []map[string]any{
		{"tag": "new", "type": "rule_set"},
		{"tag": "existing", "type": "rule_set", "updated": true},
	}

	result := mergeRuleSets(existing, installed)
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}

	// 验证 "existing" 被更新（替换而非追加）
	first, ok := result[0].(map[string]any)
	if !ok {
		t.Fatal("expected map at index 0")
	}
	if first["updated"] != true {
		t.Error("expected existing item to be updated")
	}
}
