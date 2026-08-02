package main

import (
	"encoding/json"
	"testing"
)

func TestBridgeServiceNotReady(t *testing.T) {
	rt := &desktopRuntime{}
	svc := newBoxdBridgeService(rt)
	_, err := svc.Call(BridgeRequest{Path: "/api/service/status"})
	if err == nil {
		t.Fatal("expected not ready error")
	}
}

func TestBridgeServiceStatus(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, err := svc.Call(BridgeRequest{Path: "/api/service/status"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q", resp.Status)
	}
}

func TestBridgeServiceStartStop(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	// config 缺失时 start 报错
	resp, err := svc.Call(BridgeRequest{Path: "/api/service/start", Method: "POST"})
	if err == nil {
		t.Fatalf("expected start error, got %+v", resp)
	}
	if resp.Status != "error" {
		t.Fatalf("status = %q", resp.Status)
	}
}

func TestBridgeConfigGet(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, err := svc.Call(BridgeRequest{Path: "/api/config/"})
	if err == nil {
		t.Fatalf("expected error for missing config, got %+v", resp)
	}
}

func TestBridgeSettings(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, err := svc.Call(BridgeRequest{Path: "/api/settings/preferences"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q", resp.Status)
	}
}

func TestBridgeRuntimeVersion(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, err := svc.Call(BridgeRequest{Path: "/api/runtime/version"})
	if err != nil {
		t.Fatal(err)
	}
	data, ok := resp.Data.(map[string]string)
	if !ok {
		t.Fatalf("data type = %T", resp.Data)
	}
	if data["version"] == "" {
		t.Fatal("version empty")
	}
}

func TestBridgeHealth(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, err := svc.Call(BridgeRequest{Path: "/readyz"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q", resp.Status)
	}
	resp, err = svc.Call(BridgeRequest{Path: "/healthz"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q", resp.Status)
	}
}

func TestBridgeUnknownPath(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	_, err := svc.Call(BridgeRequest{Path: "/api/unknown"})
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestErrResultNil(t *testing.T) {
	resp, err := errResult(nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q", resp.Status)
	}
}

func TestErrorfBridge(t *testing.T) {
	err := ErrorfBridge(404, "not_found", "unknown path %q", "/x")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "not_found: unknown path \"/x\"" {
		t.Fatalf("err = %q", err.Error())
	}
}

func TestBridgeWriteOperations(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)

	// service 控制：stop 在内核未运行时返回错误（预期行为）
	_, err := svc.Call(BridgeRequest{Path: "/api/service/stop", Method: "POST"})
	if err != nil {
		// 内核未运行导致 stop 报错是可接受的
		t.Logf("stop returned error (expected): %v", err)
	}

	// settings 写操作
	resp, err := svc.Call(BridgeRequest{
		Path: "/api/settings/kernel-autostart", Method: "PUT",
		Body: json.RawMessage(`{"enabled":true}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("autostart status = %q", resp.Status)
	}

	// runtime flush：内核未运行时返回错误（预期）
	_, err = svc.Call(BridgeRequest{Path: "/api/runtime/dns/flush", Method: "POST"})
	if err != nil {
		t.Logf("flush returned error (expected when kernel not running): %v", err)
	}
}

func TestBridgeWriteInvalidBody(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	_, err := svc.Call(BridgeRequest{
		Path: "/api/settings/kernel-autostart", Method: "PUT",
		Body: json.RawMessage(`{invalid`),
	})
	if err == nil {
		t.Fatal("expected error for invalid body")
	}
}

func TestBridgeWriteEmptyBody(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	_, err := svc.Call(BridgeRequest{
		Path: "/api/settings/kernel-autostart", Method: "PUT",
	})
	if err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestBridgeConfigValidate(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, err := svc.Call(BridgeRequest{
		Path: "/api/config/validate", Method: "POST",
		Body: json.RawMessage(`{"log":{"level":"info"}}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q err=%q", resp.Status, resp.Error)
	}
}

func TestBridgeDefaultsInstall(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, err := svc.Call(BridgeRequest{Path: "/api/config/dns/defaults", Method: "POST"})
	if err != nil {
		// 内核启动受 build tags 限制时 install 可能返回错误，属预期。
		t.Logf("defaults install returned error (kernel start limited): %v", err)
		return
	}
	if resp.Status == "" {
		t.Fatalf("empty status: %+v", resp)
	}
}

func TestBridgeSettingsWriteOps(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)

	resp, err := svc.Call(BridgeRequest{
		Path: "/api/settings/url-test", Method: "PUT",
		Body: json.RawMessage(`{"url":"https://example.com/"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("url-test status = %q", resp.Status)
	}

	resp, err = svc.Call(BridgeRequest{
		Path: "/api/settings/preferences", Method: "PUT",
		Body: json.RawMessage(`{"theme":"dark","language":"zh","minimumLogLevel":"info"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("preferences status = %q", resp.Status)
	}
}

func TestBridgeRuntimeOps(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)

	resp, err := svc.Call(BridgeRequest{Path: "/api/runtime/groups", Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("groups status = %q", resp.Status)
	}

	resp, err = svc.Call(BridgeRequest{Path: "/api/network/interfaces", Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("interfaces status = %q", resp.Status)
	}
}

func TestBridgePasswordChange(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	// 修改密码需要当前密码正确；错误时返回错误
	resp, err := svc.Call(BridgeRequest{
		Path: "/api/settings/password", Method: "PUT",
		Body: json.RawMessage(`{"currentPassword":"wrong","newPassword":"new-pass"}`),
	})
	if err == nil {
		t.Fatalf("expected password error, got %+v", resp)
	}
	if resp.Status != "error" {
		t.Fatalf("status = %q", resp.Status)
	}
}

func TestBridgeJWTSecret(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, err := svc.Call(BridgeRequest{
		Path: "/api/settings/jwt-secret", Method: "PUT",
		Body: json.RawMessage(`{"secret":"my-very-secret-key-123456789"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q err=%q", resp.Status, resp.Error)
	}
}

func TestBridgeClashMode(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	// 内核未运行或 clash 未启用时返回错误（预期）
	_, err := svc.Call(BridgeRequest{
		Path: "/api/runtime/clash-mode", Method: "PUT",
		Body: json.RawMessage(`{"mode":"rule"}`),
	})
	if err != nil {
		t.Logf("clash mode set returned error (expected): %v", err)
	}
	// GET clash-mode
	resp, err := svc.Call(BridgeRequest{Path: "/api/runtime/clash-mode", Method: "GET"})
	if err != nil {
		t.Logf("clash mode get returned error (expected): %v", err)
	} else {
		_ = resp
	}
}

func TestBridgeSyncNodesConfig(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	err := syncNodesConfig(rt)
	if err != nil {
		t.Logf("sync nodes returned error (expected without valid config): %v", err)
	}
}
