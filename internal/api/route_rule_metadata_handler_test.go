package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func newRouteMetadataHandler(t *testing.T, config map[string]any) (*ConfigHandler, *core.RouteRuleMetadataManager, string) {
	t.Helper()
	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, configPath, config)
	manager := core.NewRouteRuleMetadataManager(db)
	return NewConfigHandler(configPath, nil, nil, nil, core.NewDefaultRouteInstaller(), nil, manager), manager, configPath
}

func TestConfigHandlerRouteRuleMetadata(t *testing.T) {
	handler, _, _ := newRouteMetadataHandler(t, map[string]any{"route": map[string]any{
		"rules": []any{map[string]any{"action": "sniff"}},
	}})

	update := httptest.NewRecorder()
	handler.UpdateRouteRuleMetadata(update, jsonRequest(http.MethodPut, "/api/config/route/rule-metadata", `[{"name":"  协议嗅探  ","description":"识别协议"}]`))
	if update.Code != http.StatusOK {
		t.Fatalf("UpdateRouteRuleMetadata() status = %d, body = %s", update.Code, update.Body.String())
	}

	get := httptest.NewRecorder()
	handler.GetRouteRuleMetadata(get, httptest.NewRequest(http.MethodGet, "/api/config/route/rule-metadata", nil))
	if get.Code != http.StatusOK {
		t.Fatalf("GetRouteRuleMetadata() status = %d", get.Code)
	}
	metadata := decodeBody[[]model.RouteRuleMetadata](t, get)
	if len(metadata) != 1 || metadata[0].Name != "协议嗅探" || metadata[0].Description != "识别协议" {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func TestConfigHandlerRejectsInvalidRouteRuleMetadata(t *testing.T) {
	handler, _, _ := newRouteMetadataHandler(t, map[string]any{"route": map[string]any{
		"rules": []any{map[string]any{"action": "sniff"}},
	}})
	tests := []string{
		`[]`,
		`[{"name":"` + strings.Repeat("x", core.MaxRouteRuleNameLength+1) + `"}]`,
		`{`,
	}
	for _, body := range tests {
		rr := httptest.NewRecorder()
		handler.UpdateRouteRuleMetadata(rr, jsonRequest(http.MethodPut, "/api/config/route/rule-metadata", body))
		if rr.Code != http.StatusBadRequest {
			t.Errorf("body %q status = %d, want %d", body, rr.Code, http.StatusBadRequest)
		}
	}
}

func TestConfigHandlerRouteRuleMetadataErrors(t *testing.T) {
	t.Run("manager is not configured", func(t *testing.T) {
		handler := NewConfigHandler("unused", nil, nil, nil, nil, nil)
		get := httptest.NewRecorder()
		handler.GetRouteRuleMetadata(get, httptest.NewRequest(http.MethodGet, "/api/config/route/rule-metadata", nil))
		if get.Code != http.StatusNotImplemented {
			t.Fatalf("GET status = %d", get.Code)
		}
		put := httptest.NewRecorder()
		handler.UpdateRouteRuleMetadata(put, jsonRequest(http.MethodPut, "/api/config/route/rule-metadata", `[]`))
		if put.Code != http.StatusNotImplemented {
			t.Fatalf("PUT status = %d", put.Code)
		}
	})

	t.Run("config cannot be loaded", func(t *testing.T) {
		manager := core.NewRouteRuleMetadataManager(newTestDB(t))
		handler := NewConfigHandler(filepath.Join(t.TempDir(), "missing.json"), nil, nil, nil, nil, nil, manager)
		for _, method := range []string{http.MethodGet, http.MethodPut} {
			rr := httptest.NewRecorder()
			request := httptest.NewRequest(method, "/api/config/route/rule-metadata", strings.NewReader(`[]`))
			if method == http.MethodGet {
				handler.GetRouteRuleMetadata(rr, request)
			} else {
				handler.UpdateRouteRuleMetadata(rr, request)
			}
			if rr.Code != http.StatusInternalServerError {
				t.Errorf("%s status = %d", method, rr.Code)
			}
		}
	})

	t.Run("metadata database is unavailable", func(t *testing.T) {
		db := newTestDB(t)
		manager := core.NewRouteRuleMetadataManager(db)
		configPath := filepath.Join(t.TempDir(), "config.json")
		writeConfigFile(t, configPath, map[string]any{"route": map[string]any{"rules": []any{}}})
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		handler := NewConfigHandler(configPath, nil, nil, nil, nil, nil, manager)
		get := httptest.NewRecorder()
		handler.GetRouteRuleMetadata(get, httptest.NewRequest(http.MethodGet, "/api/config/route/rule-metadata", nil))
		if get.Code != http.StatusInternalServerError {
			t.Fatalf("GET status = %d", get.Code)
		}
		put := httptest.NewRecorder()
		handler.UpdateRouteRuleMetadata(put, jsonRequest(http.MethodPut, "/api/config/route/rule-metadata", `[]`))
		if put.Code != http.StatusInternalServerError {
			t.Fatalf("PUT status = %d", put.Code)
		}
	})
}

func TestInstallDefaultRouteRulesAddsMetadataNames(t *testing.T) {
	handler, manager, configPath := newRouteMetadataHandler(t, map[string]any{
		"outbounds": []any{
			map[string]any{"tag": "direct", "type": "direct"},
			map[string]any{"tag": "block", "type": "block"},
			map[string]any{"tag": "proxy", "type": "selector", "outbounds": []any{"direct"}},
		},
		"route": map[string]any{"rule_set": []any{
			map[string]any{"tag": "loyalsoldier-direct", "type": "local", "format": "source", "path": "rules/direct.json"},
			map[string]any{"tag": "geoip-cn", "type": "local", "format": "source", "path": "rules/geoip-cn.json"},
			map[string]any{"tag": "loyalsoldier-proxy", "type": "local", "format": "source", "path": "rules/proxy.json"},
			map[string]any{"tag": "loyalsoldier-reject", "type": "local", "format": "source", "path": "rules/reject.json"},
		}},
	})
	rr := httptest.NewRecorder()
	handler.InstallDefaultRouteRules(rr, httptest.NewRequest(http.MethodPost, "/api/config/route/defaults", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("InstallDefaultRouteRules() status = %d, body = %s", rr.Code, rr.Body.String())
	}
	config := decodeConfigFile(t, configPath)
	rules := config["route"].(map[string]any)["rules"].([]any)
	metadata, err := manager.List(rules)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(metadata) != 9 || metadata[0].Name != "协议嗅探" || metadata[8].Name != "代理列表流量走代理" {
		t.Fatalf("metadata = %#v", metadata)
	}
}
