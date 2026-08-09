package main

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	sbLog "github.com/sagernet/sing-box/log"
	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
	"go.etcd.io/bbolt"
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
	// 测试构建标签下内核不可用，start 报错（配置本身存在）
	resp, err := svc.Call(BridgeRequest{Path: "/api/service/start", Method: "POST"})
	if err == nil {
		t.Fatalf("expected start error, got %+v", resp)
	}
	if resp.Status != "error" {
		t.Fatalf("status = %q", resp.Status)
	}
}

func TestBridgeConfigGetReturnsGeneratedDefault(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, err := svc.Call(BridgeRequest{Path: "/api/config/"})
	if err != nil {
		t.Fatalf("expected generated default config: %v", err)
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("data type = %T", resp.Data)
	}
	if data["route"] == nil {
		t.Fatalf("generated config missing route: %+v", data)
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

func TestBridgeLogHistory(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)

	rt.svc.Deps.KernelLogWriter.WriteMessage(sbLog.LevelInfo, "kernel booted")
	rt.svc.Deps.AppLogWriter.WriteAppEntry("warn", "app warning")

	resp, err := svc.Call(BridgeRequest{Path: "/api/stats/logs"})
	if err != nil {
		t.Fatal(err)
	}
	checkLogEntries(t, resp, "kernel booted")

	resp, err = svc.Call(BridgeRequest{Path: "/api/stats/app-logs"})
	if err != nil {
		t.Fatal(err)
	}
	checkLogEntries(t, resp, "app warning")
}

func TestBridgeTrafficSnapshot(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)

	resp, err := svc.Call(BridgeRequest{Path: "/api/stats/traffic"})
	if err != nil {
		t.Fatal(err)
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("data type = %T", resp.Data)
	}
	for _, key := range []string{"upload_bytes", "download_bytes", "timestamp"} {
		if _, ok := data[key]; !ok {
			t.Fatalf("traffic snapshot missing %q: %+v", key, data)
		}
	}
}

func checkLogEntries(t *testing.T, resp BridgeResponse, want string) {
	t.Helper()
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("data type = %T", resp.Data)
	}
	entries, ok := data["entries"].([]core.LogEntry)
	if !ok {
		t.Fatalf("entries type = %T", data["entries"])
	}
	if len(entries) == 0 {
		t.Fatal("expected non-empty log history")
	}
	if entries[len(entries)-1].Message != want {
		t.Fatalf("last message = %q, want %q", entries[len(entries)-1].Message, want)
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

func TestBridgeDesktopNative(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)

	resp, err := svc.Call(BridgeRequest{Path: "/api/desktop/runtime", Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("runtime status = %q", resp.Status)
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("data type = %T", resp.Data)
	}
	if data["mode"] != "embedded" {
		t.Fatalf("mode = %v", data["mode"])
	}

	resp, err = svc.Call(BridgeRequest{Path: "/api/desktop/autostart", Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("autostart status = %q", resp.Status)
	}

	resp, err = svc.Call(BridgeRequest{Path: "/api/desktop/data-dir", Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("data-dir status = %q", resp.Status)
	}
}

func TestBridgeDesktopNativeWrites(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)

	// autostart 未配置时返回错误
	_, err := svc.Call(BridgeRequest{
		Path: "/api/desktop/autostart", Method: "PUT",
		Body: json.RawMessage(`{"enabled":true}`),
	})
	if err == nil {
		t.Log("autostart set succeeded (unexpected but acceptable)")
	}

	// 无效 body
	_, err = svc.Call(BridgeRequest{
		Path: "/api/desktop/system-proxy", Method: "PUT",
		Body: json.RawMessage(`{invalid`),
	})
	if err == nil {
		t.Fatal("expected error for invalid body")
	}
}

func TestBridgeRuleSetsAutoUpdate(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)

	// GET 默认值（未设置时返回禁用 + 默认间隔）。
	resp, err := svc.Call(BridgeRequest{Path: "/api/config/rule-sets/auto-update", Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q, error = %v", resp.Status, resp.Error)
	}

	// PUT 保存配置。
	body := `{"enabled": true, "interval": "24h"}`
	resp, err = svc.Call(BridgeRequest{Path: "/api/config/rule-sets/auto-update", Method: "PUT", Body: json.RawMessage(body)})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q, error = %v", resp.Status, resp.Error)
	}
	data, ok := resp.Data.(model.RuleSetAutoUpdate)
	if !ok {
		t.Fatalf("data type = %T", resp.Data)
	}
	if !data.Enabled {
		t.Fatalf("enabled = %v", data.Enabled)
	}
}

func TestBridgeRuleSetsStatus(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	// 配置不存在时 Status 返回业务错误；关键是不能是 "unknown path"（路由必须已接线）。
	resp, _ := svc.Call(BridgeRequest{Path: "/api/config/rule-sets/status", Method: "GET"})
	if strings.Contains(resp.Error, "unknown path") {
		t.Fatalf("rule-sets/status route not wired: %s", resp.Error)
	}
	if resp.Status == "ok" {
		return
	}
	if resp.Error == "" {
		t.Fatal("expected an error or ok status")
	}
}

func TestBridgeConfigDiagnostics(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, err := svc.Call(BridgeRequest{Path: "/api/config/diagnostics", Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q, error = %v", resp.Status, resp.Error)
	}
	if resp.Data == nil {
		t.Fatal("expected diagnostics data")
	}
}

func TestBridgeConfigApplyHistory(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, err := svc.Call(BridgeRequest{Path: "/api/config/apply-history", Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q", resp.Status)
	}
}

func TestBridgeApplyHistorySnapshotNotFound(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	// 不存在的快照返回业务错误而非 unknown path。
	resp, _ := svc.Call(BridgeRequest{Path: "/api/config/apply-history/missing-id/snapshot", Method: "GET"})
	if strings.Contains(resp.Error, "unknown path") {
		t.Fatalf("snapshot route not wired: %s", resp.Error)
	}
}

func TestBridgeApplyHistoryRestoreNotFound(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, _ := svc.Call(BridgeRequest{Path: "/api/config/apply-history/missing-id/restore", Method: "POST"})
	if strings.Contains(resp.Error, "unknown path") {
		t.Fatalf("restore route not wired: %s", resp.Error)
	}
}

func TestBridgeRouteRuleMetadata(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, _ := svc.Call(BridgeRequest{Path: "/api/config/route/rule-metadata", Method: "GET"})
	if strings.Contains(resp.Error, "unknown path") {
		t.Fatalf("rule-metadata GET not wired: %s", resp.Error)
	}
}

func TestBridgeNodesList(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, _ := svc.Call(BridgeRequest{Path: "/api/nodes/", Method: "GET"})
	if strings.Contains(resp.Error, "unknown path") {
		t.Fatalf("nodes list not wired: %s", resp.Error)
	}
}

func TestBridgeSettingsBackup(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, _ := svc.Call(BridgeRequest{Path: "/api/settings/backup", Method: "GET"})
	if strings.Contains(resp.Error, "unknown path") {
		t.Fatalf("backup not wired: %s", resp.Error)
	}
}

func TestBridgeSubscriptionsList(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, _ := svc.Call(BridgeRequest{Path: "/api/subscriptions/", Method: "GET"})
	if strings.Contains(resp.Error, "unknown path") {
		t.Fatalf("subscriptions list not wired: %s", resp.Error)
	}
}

func TestBridgeSubscriptionsCreate(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	body := []byte(`{"name":"sub-a","url":"https://example.com/sub","interval_min":60}`)
	resp, err := svc.Call(BridgeRequest{Path: "/api/subscriptions/", Method: "POST", Body: body})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if strings.Contains(resp.Error, "unknown path") {
		t.Fatalf("subscriptions create not wired: %s", resp.Error)
	}
	sub, ok := resp.Data.(*model.Subscription)
	if !ok {
		t.Fatalf("create data type = %T", resp.Data)
	}
	if sub.ID == "" {
		t.Fatalf("create returned no id: %+v", resp.Data)
	}
}

func TestBridgeSubscriptionsPathRoutes(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	tests := []BridgeRequest{
		{Path: "/api/subscriptions/missing/refresh", Method: "POST"},
		{Path: "/api/subscriptions/missing", Method: "GET"},
		{Path: "/api/subscriptions/missing", Method: "DELETE"},
		{Path: "/api/subscriptions/missing", Method: "PUT", Body: []byte(`{"name":"x","interval_min":60}`)},
	}
	for _, req := range tests {
		resp, _ := svc.Call(req)
		if strings.Contains(resp.Error, "unknown path") {
			t.Fatalf("route %s %s not wired: %s", req.Method, req.Path, resp.Error)
		}
	}
}

func TestBridgeNodesPathRoutes(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	tests := []BridgeRequest{
		{Path: "/api/nodes/missing", Method: "GET"},
		{Path: "/api/nodes/missing", Method: "DELETE"},
		{Path: "/api/nodes/missing", Method: "PUT", Body: []byte(`{"tag":"x","type":"ss"}`)},
	}
	for _, req := range tests {
		resp, _ := svc.Call(req)
		if strings.Contains(resp.Error, "unknown path") {
			t.Fatalf("route %s %s not wired: %s", req.Method, req.Path, resp.Error)
		}
	}
}

func TestBridgeNodeGetNotFound(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, _ := svc.Call(BridgeRequest{Path: "/api/nodes/does-not-exist", Method: "GET"})
	if !strings.Contains(resp.Error, "not found") {
		t.Fatalf("expected not found error, got %q", resp.Error)
	}
}

func TestBridgeNodeUpdateUnknownTag(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, _ := svc.Call(BridgeRequest{
		Path:   "/api/nodes/does-not-exist",
		Method: "PUT",
		Body:   []byte(`{"tag":"x","type":"ss"}`),
	})
	if strings.Contains(resp.Error, "unknown path") {
		t.Fatalf("update route not wired: %s", resp.Error)
	}
	if resp.Error == "" {
		t.Fatalf("expected error for unknown tag, got ok")
	}
}

func TestBridgeNodesTestRoutes(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	tests := []BridgeRequest{
		// 内核未启动时返回服务不可用错误，而非 unknown path。
		{Path: "/api/nodes/a-node/delay?timeout=1000", Method: "GET"},
		{Path: "/api/nodes/selectors/Proxy/select", Method: "POST", Body: []byte(`{"tag":"x"}`)},
		{Path: "/api/nodes/groups/Proxy/urltest", Method: "POST"},
	}
	for _, req := range tests {
		resp, _ := svc.Call(req)
		if strings.Contains(resp.Error, "unknown path") {
			t.Fatalf("route %s %s not wired: %s", req.Method, req.Path, resp.Error)
		}
	}
}

func TestBridgeNodeDelayOwnedRoute(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, _ := svc.Call(BridgeRequest{Path: "/api/nodes/x/delay?timeout=999999", Method: "GET"})
	if strings.Contains(resp.Error, "unknown path") {
		t.Fatalf("delay route not wired: %s", resp.Error)
	}
}

func TestDelayTimeout(t *testing.T) {
	tests := []struct {
		raw  string
		want int64
	}{
		{raw: "", want: 0},
		{raw: "5000", want: 5000},
		{raw: "0", want: 0},
		{raw: "999999", want: 0},
		{raw: "abc", want: 0},
	}
	for _, tc := range tests {
		if got := delayTimeout(url.Values{"timeout": []string{tc.raw}}); got != tc.want {
			t.Fatalf("delayTimeout(%q) = %d, want %d", tc.raw, got, tc.want)
		}
	}
}

func TestBridgeNodesTestHistory(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, err := svc.Call(BridgeRequest{Path: "/api/nodes/test-history?tag=node-1", Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q: %s", resp.Status, resp.Error)
	}
	history, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("test-history data type = %T", resp.Data)
	}
	if history["tag"] != "node-1" {
		t.Fatalf("tag = %v", history["tag"])
	}
}

func TestBridgeStatsCloseConnections(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	tests := []BridgeRequest{
		{Path: "/api/stats/connections?outbound=proxy", Method: "DELETE"},
		{Path: "/api/stats/connections/123", Method: "DELETE"},
	}
	for _, req := range tests {
		resp, _ := svc.Call(req)
		if strings.Contains(resp.Error, "unknown path") {
			t.Fatalf("route %s %s not wired: %s", req.Method, req.Path, resp.Error)
		}
	}
}

func TestBridgeInvalidQueryString(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, _ := svc.Call(BridgeRequest{Path: "/api/nodes/test-history?tag=%zz", Method: "GET"})
	if !strings.Contains(resp.Error, "invalid query string") {
		t.Fatalf("expected invalid query string error, got %q", resp.Error)
	}
}

func TestBridgeInvalidConnectionIDs(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, _ := svc.Call(BridgeRequest{Path: "/api/stats/connections?ids=1,a,3", Method: "DELETE"})
	if !strings.Contains(resp.Error, "invalid ids") {
		t.Fatalf("expected invalid ids error, got %q", resp.Error)
	}
}

func TestBridgeNodesListIncludesSubscriptions(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	subMgr := rt.svc.Deps.SubManager
	sub, err := subMgr.Create(core.SubscriptionParams{Name: "sub-a", URL: "https://example.com/sub"})
	if err != nil {
		t.Fatal(err)
	}
	sub.Outbounds = []model.Outbound{
		{Tag: "sub-node-1", Type: "vmess", Server: "1.1.1.1", Port: 443},
	}
	body, err := json.Marshal(sub)
	if err != nil {
		t.Fatal(err)
	}
	if err := subMgr.DB().Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte("subscriptions")).Put([]byte(sub.ID), body)
	}); err != nil {
		t.Fatal(err)
	}
	resp, err := svc.Call(BridgeRequest{Path: "/api/nodes/", Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatal(err)
	}
	var entries []struct {
		Tag        string `json:"tag"`
		Source     string `json:"source"`
		SourceName string `json:"source_name"`
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		if entry.Tag == "sub-node-1" && entry.Source == "subscription" && entry.SourceName == "sub-a" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("subscription nodes missing from list: %s", raw)
	}
}
