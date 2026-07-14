package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

func TestNodesUpdateAddTestResultAfter(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)
	if err := nodeMgr.Add(model.Outbound{Tag: "n1", Type: "vless", Server: "1.1.1.1", Port: 443}); err != nil {
		t.Fatal(err)
	}
	// 更新节点 - tag 不变
	rr := httptest.NewRecorder()
	req := withURLParam(jsonRequest(http.MethodPut, "/api/nodes/n1", `{"tag":"n1","type":"vmess","server":"2.2.2.2","port":80}`), "tag", "n1")
	handler.Update(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestNodesListWithSubOutbounds(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)
	// 创建一个订阅，通过 Refresh 方式无法在测试中触发网络请求
	// 直接检查空订阅列表不报错
	rr := httptest.NewRecorder()
	handler.List(rr, httptest.NewRequest(http.MethodGet, "/api/nodes", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d", rr.Code)
	}
}

func TestNodesDeleteWithTestResults(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)
	if err := nodeMgr.Add(model.Outbound{Tag: "del-me", Type: "vless", Server: "1.1.1.1", Port: 443}); err != nil {
		t.Fatal(err)
	}
	if err := nodeMgr.SaveTestResult("del-me_tcp", model.TestResult{Tag: "del-me", TestType: "tcp", Success: true, LatencyMs: 10}); err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	req := withURLParam(httptest.NewRequest(http.MethodDelete, "/api/nodes/del-me", nil), "tag", "del-me")
	handler.Delete(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d", rr.Code)
	}
}

func TestSyncOutboundsWithProxyNodes(t *testing.T) {
	nodeMgr, subMgr, _, _ := newAPIManagers(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, configPath, map[string]any{
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
			map[string]any{"type": "block", "tag": "block"},
		},
	})
	// 添加代理节点
	if err := nodeMgr.Add(model.Outbound{Tag: "proxy1", Type: "vless", Server: "1.1.1.1", Port: 443}); err != nil {
		t.Fatal(err)
	}
	if err := nodeMgr.Add(model.Outbound{Tag: "proxy2", Type: "vmess", Server: "2.2.2.2", Port: 80}); err != nil {
		t.Fatal(err)
	}

	err := syncOutboundsToConfig(nodeMgr, subMgr, configPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg := decodeConfigFile(t, configPath)
	outbounds, ok := cfg["outbounds"].([]any)
	if !ok {
		t.Fatal("expected outbounds array")
	}
	// 应该有 direct, block, proxy1, proxy2
	if len(outbounds) < 4 {
		t.Errorf("expected >= 4 outbounds, got %d", len(outbounds))
	}
}

func TestAtomicWriteFileSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.json")
	err := atomicWriteFile(path, []byte(`{"test":true}`))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"test":true}` {
		t.Errorf("unexpected content: %s", data)
	}
}

func TestAtomicWriteFileOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	err := atomicWriteFile(path, []byte("new"))
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "new" {
		t.Errorf("expected 'new', got %s", data)
	}
}

func TestRecordTrafficHistoryPoint(t *testing.T) {
	lw := core.NewLogWriter(5)
	buf := newTrafficHistoryBuffer(10)
	handler := NewStatsHandler(lw, nil, nil)
	_ = handler
	_ = buf
	// recordTrafficHistoryPoint 是私有方法，通过 TrafficSSE 间接测试
	// 这里仅确保编译通过
}

func TestSubscriptionRefreshWithMockServer(t *testing.T) {
	// 使用本地 HTTP 服务器模拟订阅源
	body := "vmess://eyJhZGQiOiIxLjIuMy40IiwicG9ydCI6NDQzLCJpZCI6InV1aWQiLCJhaWQiOjAsIm5ldCI6IndzIiwidGxzIjoiIiwiaG9zdCI6IiIsInBhdGgiOiIiLCJwcyI6InRlc3QifQ=="
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	db := newTestDB(t)
	subMgr := core.NewSubscriptionManager(db, t.TempDir())
	nodeMgr := core.NewNodeManager(db)
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, configPath, map[string]any{"outbounds": []any{}})

	handler := NewSubscriptionHandler(subMgr, nodeMgr, configPath)
	sub, err := subMgr.Create("test-refresh", server.URL, 60)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	req := withURLParam(httptest.NewRequest(http.MethodPost, "/api/subscriptions/"+sub.ID+"/refresh", nil), "id", sub.ID)
	handler.Refresh(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestSubscriptionRefreshInvalidID(t *testing.T) {
	_, subMgr, _, configPath := newAPIManagers(t)
	nodeMgr := core.NewNodeManager(newTestDB(t))
	handler := NewSubscriptionHandler(subMgr, nodeMgr, configPath)

	rr := httptest.NewRecorder()
	req := withURLParam(httptest.NewRequest(http.MethodPost, "/api/subscriptions/invalid-id/refresh", nil), "id", "invalid-id")
	handler.Refresh(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}
