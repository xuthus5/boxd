package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxui/internal/model"
)

func TestNodesHandlerListGetUpdateDeleteAndSync(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)

	if err := nodeMgr.Add(model.Outbound{Tag: "old", Type: "vless", Server: "1.2.3.4", Port: 443}); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.List(rr, httptest.NewRequest(http.MethodGet, "/api/nodes", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("list status = %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"old"`) {
		t.Fatalf("list body = %s", rr.Body.String())
	}

	rr = httptest.NewRecorder()
	req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/nodes/old", nil), "tag", "old")
	handler.Get(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = withURLParam(
		jsonRequest(http.MethodPut, "/api/nodes/old", `{"tag":"new","type":"vmess","server":"5.6.7.8","port":80,"config":{"uuid":"u"}}`),
		"tag",
		"old",
	)
	handler.Update(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("update status = %d body=%s", rr.Code, rr.Body.String())
	}
	if nodeMgr.Get("old") != nil || nodeMgr.Get("new") == nil {
		t.Fatal("node rename did not persist")
	}

	rr = httptest.NewRecorder()
	handler.SyncToConfig(rr, httptest.NewRequest(http.MethodPost, "/api/nodes/sync-config", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("sync status = %d", rr.Code)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"proxy"`) || !strings.Contains(string(data), `"new"`) {
		t.Fatalf("synced config = %s", string(data))
	}

	rr = httptest.NewRecorder()
	req = withURLParam(httptest.NewRequest(http.MethodDelete, "/api/nodes/new", nil), "tag", "new")
	handler.Delete(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete status = %d", rr.Code)
	}
}

func TestNodesHandlerErrors(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)

	tests := []struct {
		name   string
		method func(*httptest.ResponseRecorder)
		want   int
	}{
		{
			name: "get missing tag",
			method: func(rr *httptest.ResponseRecorder) {
				handler.Get(rr, httptest.NewRequest(http.MethodGet, "/api/nodes/", nil))
			},
			want: http.StatusBadRequest,
		},
		{
			name: "get not found",
			method: func(rr *httptest.ResponseRecorder) {
				req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/nodes/nope", nil), "tag", "nope")
				handler.Get(rr, req)
			},
			want: http.StatusNotFound,
		},
		{
			name: "update invalid json",
			method: func(rr *httptest.ResponseRecorder) {
				if err := nodeMgr.Add(model.Outbound{Tag: "bad-json", Type: "vless"}); err != nil {
					t.Fatal(err)
				}
				req := withURLParam(jsonRequest(http.MethodPut, "/api/nodes/bad-json", `{`), "tag", "bad-json")
				handler.Update(rr, req)
			},
			want: http.StatusBadRequest,
		},
		{
			name: "update missing required fields",
			method: func(rr *httptest.ResponseRecorder) {
				if err := nodeMgr.Add(model.Outbound{Tag: "missing-fields", Type: "vless"}); err != nil {
					t.Fatal(err)
				}
				req := withURLParam(jsonRequest(http.MethodPut, "/api/nodes/missing-fields", `{"tag":"","type":""}`), "tag", "missing-fields")
				handler.Update(rr, req)
			},
			want: http.StatusBadRequest,
		},
		{
			name: "delete missing tag",
			method: func(rr *httptest.ResponseRecorder) {
				handler.Delete(rr, httptest.NewRequest(http.MethodDelete, "/api/nodes/", nil))
			},
			want: http.StatusBadRequest,
		},
		{
			name: "sync config error",
			method: func(rr *httptest.ResponseRecorder) {
				badHandler := NewNodesHandler(nodeMgr, subMgr, t.TempDir()+"/missing/config.json")
				badHandler.SyncToConfig(rr, httptest.NewRequest(http.MethodPost, "/api/nodes/sync-config", nil))
			},
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tt.method(rr)
			if rr.Code != tt.want {
				t.Fatalf("status = %d, want %d", rr.Code, tt.want)
			}
		})
	}
}

func TestSyncOutboundsToConfigPreservesNonProxyOutbounds(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	writeConfigFile(t, configPath, map[string]any{
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
			map[string]any{"type": "wireguard", "tag": "wg"},
		},
		"route": map[string]any{},
	})
	if err := nodeMgr.Add(model.Outbound{
		Tag: "node-a", Type: "trojan", Server: "example.com", Port: 443,
		Raw: map[string]any{"password": "secret"},
	}); err != nil {
		t.Fatal(err)
	}

	if err := syncOutboundsToConfig(nodeMgr, subMgr, configPath); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	outbounds := cfg["outbounds"].([]any)
	if len(outbounds) < 4 {
		t.Fatalf("outbounds length = %d, want at least 4", len(outbounds))
	}
	if !strings.Contains(string(data), `"wg"`) || !strings.Contains(string(data), `"password"`) {
		t.Fatalf("config missing preserved/raw fields: %s", string(data))
	}
	if !strings.Contains(string(data), `"tag": "direct"`) {
		t.Fatalf("config missing direct builtin: %s", string(data))
	}
}

func TestSyncOutboundsToConfigKeepsExistingProxySelector(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	writeConfigFile(t, configPath, map[string]any{
		"outbounds": []any{
			map[string]any{"type": "wireguard", "tag": "wg"},
		},
		"route": map[string]any{"final": "direct"},
	})
	if err := nodeMgr.Add(model.Outbound{Tag: "proxy", Type: "selector", Raw: map[string]any{"outbounds": []string{"wg"}}}); err != nil {
		t.Fatal(err)
	}

	if err := syncOutboundsToConfig(nodeMgr, subMgr, configPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), `"proxy"`) != 1 {
		t.Fatalf("config should contain one proxy selector: %s", string(data))
	}
	if !strings.Contains(string(data), `"final": "direct"`) {
		t.Fatalf("route final should be preserved: %s", string(data))
	}
}

func TestSyncOutboundsToConfigPreservesCustomGroupsAndBuiltins(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	writeConfigFile(t, configPath, map[string]any{
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
			map[string]any{"type": "block", "tag": "block"},
			map[string]any{"type": "selector", "tag": "whitelist", "outbounds": []string{"direct", "proxy"}},
			map[string]any{"type": "urltest", "tag": "auto", "outbounds": []string{"node-a"}},
		},
		"route": map[string]any{"final": "proxy"},
	})
	if err := nodeMgr.Add(model.Outbound{Tag: "node-a", Type: "trojan", Server: "example.com", Port: 443}); err != nil {
		t.Fatal(err)
	}

	if err := syncOutboundsToConfig(nodeMgr, subMgr, configPath); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, needle := range []string{`"tag": "block"`, `"tag": "whitelist"`, `"tag": "auto"`} {
		if !strings.Contains(text, needle) {
			t.Fatalf("config missing preserved outbound %s: %s", needle, text)
		}
	}
}

func TestSyncOutboundsToConfigPreservesExistingManagedOutboundFields(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	writeConfigFile(t, configPath, map[string]any{
		"outbounds": []any{
			map[string]any{
				"type": "vless", "tag": "node-a", "server": "old.example", "server_port": 443,
				"routing_mark": 128, "domain_strategy": "prefer_ipv4", "packet_encoding": "",
			},
		},
		"route": map[string]any{},
	})
	if err := nodeMgr.Add(model.Outbound{
		Tag: "node-a", Type: "vless", Server: "example.com", Port: 8443,
		Raw: map[string]any{"uuid": "u", "tls": map[string]any{"enabled": true}},
	}); err != nil {
		t.Fatal(err)
	}

	if err := syncOutboundsToConfig(nodeMgr, subMgr, configPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, needle := range []string{`"routing_mark": 128`, `"domain_strategy": "prefer_ipv4"`, `"packet_encoding": ""`} {
		if !strings.Contains(text, needle) {
			t.Fatalf("config missing preserved field %s: %s", needle, text)
		}
	}
}

func TestSubscriptionHandlerCRUDAndRefresh(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewSubscriptionHandler(subMgr, nodeMgr, configPath)

	rr := httptest.NewRecorder()
	handler.Create(rr, jsonRequest(http.MethodPost, "/api/subscriptions", `{"name":"sub","url":"https://example.test/sub","interval_min":0}`))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", rr.Code, rr.Body.String())
	}
	created := decodeBody[model.Subscription](t, rr)
	if created.IntervalMin != 60 {
		t.Fatalf("default interval = %d", created.IntervalMin)
	}

	rr = httptest.NewRecorder()
	req := withURLParam(jsonRequest(http.MethodPut, "/api/subscriptions/"+created.ID, `{"name":"renamed","url":"","interval_min":30}`), "id", created.ID)
	handler.Update(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("update status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = withURLParam(httptest.NewRequest(http.MethodGet, "/api/subscriptions/"+created.ID, nil), "id", created.ID)
	handler.Get(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.List(rr, httptest.NewRequest(http.MethodGet, "/api/subscriptions", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("list status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = withURLParam(httptest.NewRequest(http.MethodDelete, "/api/subscriptions/"+created.ID, nil), "id", created.ID)
	handler.Delete(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete status = %d", rr.Code)
	}
}

func TestSubscriptionHandlerErrorsAndRefreshAll(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewSubscriptionHandler(subMgr, nodeMgr, configPath)

	rr := httptest.NewRecorder()
	handler.RefreshAll(rr, httptest.NewRequest(http.MethodPost, "/api/subscriptions/refresh-all", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("empty refresh all status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.Create(rr, jsonRequest(http.MethodPost, "/api/subscriptions", `{`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid create status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.Create(rr, jsonRequest(http.MethodPost, "/api/subscriptions", `{"name":"","url":""}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("missing fields status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req := withURLParam(jsonRequest(http.MethodPut, "/api/subscriptions/missing", `{`), "id", "missing")
	handler.Update(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid update status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = withURLParam(jsonRequest(http.MethodPut, "/api/subscriptions/missing", `{"name":"x"}`), "id", "missing")
	handler.Update(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("missing update status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = withURLParam(httptest.NewRequest(http.MethodDelete, "/api/subscriptions/missing", nil), "id", "missing")
	handler.Delete(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("missing delete status = %d", rr.Code)
	}

	if _, err := subMgr.Create("bad", "://bad-url", 60); err != nil {
		t.Fatal(err)
	}
	bad := subMgr.List()[0]
	rr = httptest.NewRecorder()
	req = withURLParam(httptest.NewRequest(http.MethodPost, "/api/subscriptions/"+bad.ID+"/refresh", nil), "id", bad.ID)
	handler.Refresh(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("refresh status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.RefreshAll(rr, httptest.NewRequest(http.MethodPost, "/api/subscriptions/refresh-all", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("refresh all status = %d", rr.Code)
	}
	envelope := decodeEnvelope(t, rr)
	if envelope.Status != model.StatusPartial {
		t.Fatalf("expected partial status, got %#v", envelope.Status)
	}

	rr = httptest.NewRecorder()
	req = withURLParam(httptest.NewRequest(http.MethodGet, "/api/subscriptions/missing", nil), "id", "missing")
	handler.Get(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("missing get status = %d", rr.Code)
	}
}

func TestSyncOutboundsCreatesSubscriptionGroups(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	writeConfigFile(t, configPath, map[string]any{
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
		},
		"route": map[string]any{},
	})

	sub, err := subMgr.Create("my-sub", "https://example.com/sub", 60)
	if err != nil {
		t.Fatal(err)
	}
	updated := model.Subscription{
		ID: sub.ID, Name: sub.Name, URL: sub.URL, IntervalMin: sub.IntervalMin,
		Outbounds: []model.Outbound{
			{Tag: "hk-1", Type: "vless", Server: "1.1.1.1", Port: 443},
			{Tag: "us-1", Type: "vless", Server: "2.2.2.2", Port: 443},
		},
	}
	newData, _ := json.Marshal(updated)
	_ = subMgr.DB().Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("subscriptions"))
		return b.Put([]byte(sub.ID), newData)
	})

	if err := syncOutboundsToConfig(nodeMgr, subMgr, configPath); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	outbounds := cfg["outbounds"].([]any)

	var subGroup any
	for _, ob := range outbounds {
		obm := ob.(map[string]any)
		if obm["type"] == "urltest" && obm["tag"] == "my-sub" {
			subGroup = obm
		}
	}
	if subGroup == nil {
		t.Fatalf("urltest group for subscription not found: %s", string(data))
	}
	members := subGroup.(map[string]any)["outbounds"].([]any)
	if len(members) != 2 {
		t.Fatalf("group members = %d, want 2", len(members))
	}

	var proxySel any
	for _, ob := range outbounds {
		obm := ob.(map[string]any)
		if obm["type"] == "selector" && obm["tag"] == "proxy" {
			proxySel = obm
		}
	}
	if proxySel == nil {
		t.Fatalf("proxy selector not found: %s", string(data))
	}
	proxyMembers := proxySel.(map[string]any)["outbounds"].([]any)
	foundGroup := false
	for _, m := range proxyMembers {
		if m.(string) == "my-sub" {
			foundGroup = true
		}
	}
	if !foundGroup {
		t.Fatalf("proxy selector should include subscription group tag: %v", proxyMembers)
	}
	if proxySel.(map[string]any)["default"] != "my-sub" {
		t.Fatalf("proxy default should be 'my-sub', got %v", proxySel.(map[string]any)["default"])
	}
}
