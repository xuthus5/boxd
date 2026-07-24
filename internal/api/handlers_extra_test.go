package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func TestNodesListWithSubscriptions(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)

	// 添加一个导入节点
	if err := nodeMgr.Add(model.Outbound{Tag: "imported", Type: "vless", Server: "1.1.1.1", Port: 443}); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.List(rr, httptest.NewRequest(http.MethodGet, "/api/nodes", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	nodes := decodeBody[[]map[string]any](t, rr)
	if len(nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0]["tag"] != "imported" {
		t.Errorf("tag = %v, want 'imported'", nodes[0]["tag"])
	}
}

func TestNodesGetMissingTag(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/nodes/", nil)
	handler.Get(rr, withURLParam(req, "tag", ""))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestNodesUpdateTagChanged(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)

	if err := nodeMgr.Add(model.Outbound{Tag: "old-tag", Type: "vless", Server: "1.1.1.1", Port: 443}); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	req := withURLParam(jsonRequest(http.MethodPut, "/api/nodes/old-tag", `{"tag":"new-tag","type":"vmess","server":"2.2.2.2","port":80}`), "tag", "old-tag")
	handler.Update(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestNodesUpdateRejectsTagConflict(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)
	if err := nodeMgr.Add(model.Outbound{Tag: "old-tag", Type: "vless"}); err != nil {
		t.Fatal(err)
	}
	if err := nodeMgr.Add(model.Outbound{Tag: "taken", Type: "trojan"}); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	req := withURLParam(jsonRequest(http.MethodPut, "/api/nodes/old-tag", `{"tag":"taken","type":"vless"}`), "tag", "old-tag")
	handler.Update(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	envelope := decodeEnvelope(t, rr)
	if envelope.Error == nil || envelope.Error.Code != model.ErrorConflict {
		t.Fatalf("error = %#v", envelope.Error)
	}
	if nodeMgr.Get("old-tag") == nil || nodeMgr.Get("taken").Type != "trojan" {
		t.Fatal("tag conflict changed existing nodes")
	}
}

func TestNodesUpdateMissingNode(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)

	rr := httptest.NewRecorder()
	req := withURLParam(jsonRequest(http.MethodPut, "/api/nodes/missing", `{"tag":"new","type":"vless"}`), "tag", "missing")
	handler.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	envelope := decodeEnvelope(t, rr)
	if envelope.Error == nil || envelope.Error.Code != model.ErrorNodeNotFound {
		t.Fatalf("error = %#v", envelope.Error)
	}
}

func TestNodesUpdateMissingTag(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/nodes/", nil)
	handler.Update(rr, withURLParam(req, "tag", ""))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestNodesUpdateInvalidJSON(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)

	if err := nodeMgr.Add(model.Outbound{Tag: "test", Type: "vless", Server: "1.1.1.1", Port: 443}); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	req := withURLParam(jsonRequest(http.MethodPut, "/api/nodes/test", "not json"), "tag", "test")
	handler.Update(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestNodesUpdateMissingFields(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)

	if err := nodeMgr.Add(model.Outbound{Tag: "test", Type: "vless", Server: "1.1.1.1", Port: 443}); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	req := withURLParam(jsonRequest(http.MethodPut, "/api/nodes/test", `{"tag":"","type":""}`), "tag", "test")
	handler.Update(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestNodesDeleteMissingTag(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/nodes/", nil)
	handler.Delete(rr, withURLParam(req, "tag", ""))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestNodesDeleteNotFound(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/nodes/missing", nil)
	handler.Delete(rr, withURLParam(req, "tag", "missing"))

	// NodeManager.Delete 不报错即使节点不存在
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestNodesSyncToConfig(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)

	rr := httptest.NewRecorder()
	handler.SyncToConfig(rr, httptest.NewRequest(http.MethodPost, "/api/nodes/sync-config", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestNodesSyncToConfigError(t *testing.T) {
	nodeMgr, subMgr, _, _ := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, "/nonexistent/config.json")

	rr := httptest.NewRecorder()
	handler.SyncToConfig(rr, httptest.NewRequest(http.MethodPost, "/api/nodes/sync-config", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestNodesSyncToConfigRestarts(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	restarter := &fakeRestartable{}
	handler := NewNodesHandler(nodeMgr, subMgr, configPath, restarter)

	rr := httptest.NewRecorder()
	handler.SyncToConfig(rr, httptest.NewRequest(http.MethodPost, "/api/nodes/sync-config", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if restarter.calls != 1 {
		t.Fatalf("restart calls = %d, want 1", restarter.calls)
	}
	if !strings.Contains(rr.Body.String(), `"restarted":true`) {
		t.Fatalf("body = %s", rr.Body.String())
	}
}

func TestNodesSyncToConfigRestartError(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	restarter := &fakeRestartable{errs: []error{errors.New("restart failed")}}
	handler := NewNodesHandler(nodeMgr, subMgr, configPath, restarter)

	rr := httptest.NewRecorder()
	handler.SyncToConfig(rr, httptest.NewRequest(http.MethodPost, "/api/nodes/sync-config", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if restarter.calls != 2 {
		t.Fatalf("restart calls = %d, want 2", restarter.calls)
	}
}

func TestImportSaveNodeSyncRestarts(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	restarter := &fakeRestartable{}
	handler := NewImportHandler(nodeMgr, subMgr, configPath, restarter)

	rr := httptest.NewRecorder()
	handler.SaveNode(rr, jsonRequest(http.MethodPost, "/api/import/save", `{"tag":"n1","type":"vless","server":"1.1.1.1","port":443,"config":{"uuid":"u"}}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if restarter.calls != 1 {
		t.Fatalf("restart calls = %d, want 1", restarter.calls)
	}
}

func TestSubscriptionRefreshFailedSkipsRestart(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	restarter := &fakeRestartable{}
	handler := NewSubscriptionHandler(subMgr, nodeMgr, configPath, restarter)

	rr := httptest.NewRecorder()
	req := withURLParam(httptest.NewRequest(http.MethodPost, "/api/subscriptions/missing/refresh", nil), "id", "missing")
	handler.Refresh(rr, req)
	if rr.Code == http.StatusOK {
		t.Fatal("expected refresh failure for missing subscription")
	}
	if restarter.calls != 0 {
		t.Fatalf("restart calls = %d, want 0", restarter.calls)
	}
}

func TestSubscriptionCreateInvalidInterval(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewSubscriptionHandler(subMgr, nodeMgr, configPath)

	rr := httptest.NewRecorder()
	handler.Create(rr, jsonRequest(http.MethodPost, "/api/subscriptions", `{"name":"test","url":"https://example.com","interval_min":-1}`))

	// interval <= 0 应该被设为默认值或接受
	if rr.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusCreated)
	}
}

func TestSubscriptionRefreshNotFound(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewSubscriptionHandler(subMgr, nodeMgr, configPath)

	rr := httptest.NewRecorder()
	req := withURLParam(httptest.NewRequest(http.MethodPost, "/api/subscriptions/999/refresh", nil), "id", "999")
	handler.Refresh(rr, req)

	// Refresh 返回 error 时 handler 返回 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestSubscriptionRefreshAllSuccess(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewSubscriptionHandler(subMgr, nodeMgr, configPath)

	rr := httptest.NewRecorder()
	handler.RefreshAll(rr, httptest.NewRequest(http.MethodPost, "/api/subscriptions/refresh-all", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestSettingsSetTestURLInvalidJSON(t *testing.T) {
	_, _, settingsMgr, _ := newAPIManagers(t)
	handler := NewSettingsHandler(settingsMgr)

	rr := httptest.NewRecorder()
	handler.SetTestURL(rr, jsonRequest(http.MethodPut, "/api/settings/url-test", "not json"))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestSettingsSetTestURLRejectsUnsupportedURL(t *testing.T) {
	_, _, settingsMgr, _ := newAPIManagers(t)
	handler := NewSettingsHandler(settingsMgr)

	for _, rawURL := range []string{"file:///tmp/probe", "/relative", "https://example.com:bad/probe"} {
		rr := httptest.NewRecorder()
		handler.SetTestURL(rr, jsonRequest(http.MethodPut, "/api/settings/url-test", `{"url":"`+rawURL+`"}`))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("url %q status = %d body=%s", rawURL, rr.Code, rr.Body.String())
		}
	}
	if err := settingsMgr.Set("url_test", "file:///tmp/legacy"); err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler.GetTestURL(rr, httptest.NewRequest(http.MethodGet, "/api/settings/url-test", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), defaultTestURL) {
		t.Fatalf("invalid stored URL response = %d %s", rr.Code, rr.Body.String())
	}
}

func TestImportSaveNodeInvalidJSON(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewImportHandler(nodeMgr, subMgr, configPath)

	rr := httptest.NewRecorder()
	handler.SaveNode(rr, jsonRequest(http.MethodPost, "/api/import/save", "not json"))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestImportSaveNodeSuccess(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewImportHandler(nodeMgr, subMgr, configPath)

	rr := httptest.NewRecorder()
	handler.SaveNode(rr, jsonRequest(http.MethodPost, "/api/import/save", `{"tag":"test-node","type":"vless","server":"1.2.3.4","port":443,"config":{}}`))

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestSyncOutboundsToConfigEmptyConfig(t *testing.T) {
	nodeMgr, subMgr, _, _ := newAPIManagers(t)
	configPath := filepath.Join(t.TempDir(), "empty.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}

	err := syncOutboundsToConfig(nodeMgr, subMgr, configPath)
	if err != nil {
		t.Fatal(err)
	}

	cfg := decodeConfigFile(t, configPath)
	if _, ok := cfg["outbounds"]; !ok {
		t.Error("expected outbounds in config")
	}
}

func TestSyncOutboundsToConfigInvalidJSON(t *testing.T) {
	nodeMgr, subMgr, _, _ := newAPIManagers(t)
	configPath := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(configPath, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}

	err := syncOutboundsToConfig(nodeMgr, subMgr, configPath)
	if err == nil {
		t.Fatal("expected error for invalid JSON config")
	}
}

func TestSyncOutboundsToConfigReadError(t *testing.T) {
	nodeMgr, subMgr, _, _ := newAPIManagers(t)

	err := syncOutboundsToConfig(nodeMgr, subMgr, "/nonexistent/config.json")
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestNewNetworkHandler(t *testing.T) {
	h := NewNetworkHandler()
	if h == nil {
		t.Fatal("expected non-nil NetworkHandler")
	}
}

func TestRuntimeHandlerWithURLTest(t *testing.T) {
	instance := core.NewSBInstance("/nonexistent/config.json", core.NewLogWriter(5))
	_ = instance
}
