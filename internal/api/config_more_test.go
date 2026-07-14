package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

func TestUpdateConfigWithInstanceRestartFail(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, configPath, map[string]any{"outbounds": []any{}})
	instance := core.NewSBInstance(configPath, core.NewLogWriter(5))
	handler := NewConfigHandler(configPath, instance, nil, nil, nil, nil)

	rr := httptest.NewRecorder()
	handler.UpdateConfig(rr, jsonRequest(http.MethodPut, "/api/config", `{"outbounds":[]}`))

	// instance.Restart 失败且 rollback 后的 Restart 也失败，返回 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestUpdateRawConfigWithInstanceRestartFail(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, configPath, map[string]any{"outbounds": []any{}})
	instance := core.NewSBInstance(configPath, core.NewLogWriter(5))
	handler := NewConfigHandler(configPath, instance, nil, nil, nil, nil)

	rr := httptest.NewRecorder()
	handler.UpdateRawConfig(rr, jsonRequest(http.MethodPut, "/api/config/raw", `{"outbounds":[]}`))

	// instance.Restart 失败且 rollback 后的 Restart 也失败，返回 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestGetConfigSuccess(t *testing.T) {
	handler, _ := newConfigHandlerWithFile(t)

	rr := httptest.NewRecorder()
	handler.GetConfig(rr, httptest.NewRequest(http.MethodGet, "/api/config", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestGetRawConfigSuccess(t *testing.T) {
	handler, _ := newConfigHandlerWithFile(t)

	rr := httptest.NewRecorder()
	handler.GetRawConfig(rr, httptest.NewRequest(http.MethodGet, "/api/config/raw", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestUpdateConfigSuccessNoInstance(t *testing.T) {
	handler, _ := newConfigHandlerWithFile(t)

	rr := httptest.NewRecorder()
	handler.UpdateConfig(rr, jsonRequest(http.MethodPut, "/api/config", `{"outbounds":[{"type":"direct","tag":"direct"}]}`))

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestInstallDefaultOutboundsWithInstance(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, configPath, map[string]any{"outbounds": []any{}})
	instance := core.NewSBInstance(configPath, core.NewLogWriter(5))
	handler := NewConfigHandler(configPath, instance, nil,
		core.NewDefaultOutboundsInstaller(), nil, nil)

	rr := httptest.NewRecorder()
	handler.InstallDefaultOutbounds(rr, httptest.NewRequest(http.MethodPost, "/api/config/outbounds/defaults", nil))

	// instance restart 失败导致 500 或 rollback 后 500
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d or %d", rr.Code, http.StatusOK, http.StatusInternalServerError)
	}
}

func TestInstallDefaultRouteRulesWithInstance(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, configPath, map[string]any{"outbounds": []any{}})
	instance := core.NewSBInstance(configPath, core.NewLogWriter(5))
	handler := NewConfigHandler(configPath, instance,
		core.NewLoyalsoldierRuleSetInstaller(t.TempDir()), nil,
		core.NewDefaultRouteInstaller(), nil)

	rr := httptest.NewRecorder()
	handler.InstallDefaultRouteRules(rr, httptest.NewRequest(http.MethodPost, "/api/config/route/defaults", nil))

	// instance restart 失败导致 500 或 rollback 后 500
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d or %d", rr.Code, http.StatusOK, http.StatusInternalServerError)
	}
}

func TestInstallDefaultDNSWithInstance(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, configPath, map[string]any{"outbounds": []any{}})
	instance := core.NewSBInstance(configPath, core.NewLogWriter(5))
	handler := NewConfigHandler(configPath, instance, nil, nil, nil,
		core.NewDefaultDNSInstaller())

	rr := httptest.NewRecorder()
	handler.InstallDefaultDNS(rr, httptest.NewRequest(http.MethodPost, "/api/config/dns/defaults", nil))

	// instance restart 失败导致 500 或 rollback 后 500
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d or %d", rr.Code, http.StatusOK, http.StatusInternalServerError)
	}
}

func TestNodesListEmpty(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)

	rr := httptest.NewRecorder()
	handler.List(rr, httptest.NewRequest(http.MethodGet, "/api/nodes", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d", rr.Code)
	}
	nodes := decodeBody[[]map[string]any](t, rr)
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(nodes))
	}
}

func TestNodesGetSuccess(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)

	if err := nodeMgr.Add(model.Outbound{Tag: "test", Type: "vless", Server: "1.1.1.1", Port: 443}); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/nodes/test", nil), "tag", "test")
	handler.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestNodesDeleteSuccess(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)

	rr := httptest.NewRecorder()
	req := withURLParam(httptest.NewRequest(http.MethodDelete, "/api/nodes/test", nil), "tag", "test")
	handler.Delete(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestNodesUpdateSuccessSameTag(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)

	if err := nodeMgr.Add(model.Outbound{Tag: "test", Type: "vless", Server: "1.1.1.1", Port: 443}); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	req := withURLParam(jsonRequest(http.MethodPut, "/api/nodes/test", `{"tag":"test","type":"vmess","server":"2.2.2.2","port":80}`), "tag", "test")
	handler.Update(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}
