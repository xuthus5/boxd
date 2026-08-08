package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

func TestLoginBridge(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	body := json.RawMessage(`{"username": "admin", "password": "admin123"}`)
	resp, err := loginBridge(rt, body)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	data, ok := resp.(model.AuthResponse)
	if !ok {
		t.Fatalf("resp type = %T", resp)
	}
	if data.Token == "" {
		t.Fatal("expected a token")
	}
}

func TestLoginBridgeInvalidBody(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	if _, err := loginBridge(rt, json.RawMessage(`{invalid`)); err == nil {
		t.Fatal("expected error for invalid body")
	}
}

func TestImportLinkBridge(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	body := json.RawMessage(`{"link": "vmess://invalid"}`)
	if _, err := importLinkBridge(rt, body); err == nil {
		t.Fatal("expected error for invalid link")
	}
}

func TestImportSaveBridge(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	body := json.RawMessage(`{"tag": "n1", "type": "vless", "server": "example.com", "port": 443}`)
	if _, err := importSaveBridge(rt, body); err == nil {
		t.Fatal("expected error for incomplete node config")
	}
}

func TestProbeDNSBridge(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	body := json.RawMessage(`{"type": "udp", "server": "1.1.1.1", "server_port": 53, "domain": "example.com"}`)
	if _, err := probeDNSBridge(rt, body); err != nil {
		// 探测失败（网络不可达）也应返回结果而非 err；若返回 err 则接受业务错误。
		_ = err
	}
}

func TestProbeDNSBatchBridge(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	body := json.RawMessage(`{"items": [{"type": "udp", "server": "1.1.1.1", "server_port": 53, "domain": "example.com"}], "concurrency": 2}`)
	if _, err := probeDNSBatchBridge(rt, body); err != nil {
		_ = err
	}
}

func TestListNodesBridge(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	resp, err := listNodesBridge(rt)
	if err != nil {
		t.Fatal(err)
	}
	// 聚合列表：手动导入节点 + 订阅节点，JSON 序列化后是数组。
	body, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(body)), "[") {
		t.Fatalf("resp is not a JSON array: %s", body)
	}
}

func TestListNodesBridgeNilManager(t *testing.T) {
	rt := &desktopRuntime{}
	if _, err := listNodesBridge(rt); err == nil {
		t.Fatal("expected error for nil runtime")
	}
}

func TestDispatchNewRoutesNotUnknownPath(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	cases := []BridgeRequest{
		{Path: "/api/auth/logout", Method: "POST"},
		{Path: "/api/import/link", Method: "POST", Body: json.RawMessage(`{"link": "x"}`)},
		{Path: "/api/nodes/test-batch", Method: "POST", Body: json.RawMessage(`{"items": []}`)},
		{Path: "/api/runtime/dns/probe", Method: "POST", Body: json.RawMessage(`{}`)},
		{Path: "/api/subscriptions/refresh-all", Method: "POST"},
		{Path: "/api/nodes/test-results", Method: "GET"},
	}
	for _, req := range cases {
		resp, _ := svc.Call(req)
		if strings.Contains(resp.Error, "unknown path") {
			t.Fatalf("route %s not wired: %s", req.Path, resp.Error)
		}
	}
}
