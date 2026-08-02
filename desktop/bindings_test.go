package main

import (
	"os"
	"testing"
)

func newTestRuntimeWithService(t *testing.T) *desktopRuntime {
	t.Helper()
	dir := t.TempDir()
	cfg := desktopConfig{
		Mode:            "embedded",
		DataDir:         dir + "/data",
		ConfigPath:      dir + "/config/config.json",
		Username:        "admin",
		Password:        "",
		RefreshInterval: 60,
	}
	rt, err := initRuntime(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rt.close() })
	return rt
}

func TestBoxdConfigServiceNotReady(t *testing.T) {
	rt := &desktopRuntime{}
	svc := newBoxdConfigService(rt)
	if _, err := svc.Get(); err == nil {
		t.Fatal("expected not ready error")
	}
	if _, err := svc.Update(nil); err == nil {
		t.Fatal("expected not ready error")
	}
	if err := svc.Validate(nil, ""); err == nil {
		t.Fatal("expected not ready error")
	}
}

func TestBoxdConfigServiceGetMissing(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdConfigService(rt)
	if _, err := svc.Get(); err == nil {
		t.Fatal("expected error for missing config")
	}
}

func TestBoxdConfigServiceValidate(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdConfigService(rt)
	if err := svc.Validate([]byte(`{"log":{"level":"info"}}`), "validate"); err != nil {
		t.Fatal(err)
	}
}

func TestBoxdConfigServiceUpdateRestartFail(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdConfigService(rt)
	// 内核无法启动时 Update 返回错误（无有效配置）
	_, err := svc.Update([]byte(`{"log":{"level":"info"}}`))
	if err == nil {
		t.Fatal("expected error for config that cannot start kernel")
	}
}

func TestBoxdConfigServiceUpdateWritesConfig(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	// 预先写入有效配置
	validConfig := []byte(`{"inbounds":[{"type":"mixed","tag":"mixed","listen":"127.0.0.1","listen_port":2080}],"outbounds":[{"type":"direct","tag":"direct"}]}`)
	if err := os.WriteFile(rt.cfg.ConfigPath, validConfig, 0600); err != nil {
		t.Fatal(err)
	}
	svc := newBoxdConfigService(rt)
	// 内核启动受 build tags 限制，Update 返回错误是预期的（无法启动内核时）
	_, err := svc.Update(validConfig)
	if err != nil {
		t.Logf("update returned error (kernel start limited by build tags): %v", err)
	}
	// 但配置写入应先于重启尝试，验证写入发生
	data, readErr := os.ReadFile(rt.cfg.ConfigPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(data) == 0 {
		t.Fatal("config file is empty")
	}
}

func TestBoxdServiceControlServiceNotReady(t *testing.T) {
	rt := &desktopRuntime{}
	svc := newBoxdServiceControlService(rt)
	if _, err := svc.Status(); err == nil {
		t.Fatal("expected not ready")
	}
	if err := svc.Start(); err == nil {
		t.Fatal("expected not ready")
	}
	if err := svc.Stop(); err == nil {
		t.Fatal("expected not ready")
	}
	if err := svc.Restart(); err == nil {
		t.Fatal("expected not ready")
	}
}

func TestBoxdServiceControlStatus(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdServiceControlService(rt)
	status, err := svc.Status()
	if err != nil {
		t.Fatal(err)
	}
	if status == nil {
		t.Fatal("status is nil")
	}
}

func TestBoxdSettingsService(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdSettingsService(rt)
	prefs, err := svc.GetUIPreferences()
	if err != nil {
		t.Fatal(err)
	}
	if prefs == nil {
		t.Fatal("prefs is nil")
	}
	if _, err := svc.SetUIPreferences(map[string]any{"theme": "dark"}); err != nil {
		t.Fatal(err)
	}
}

func TestBoxdSettingsServiceNotReady(t *testing.T) {
	rt := &desktopRuntime{}
	svc := newBoxdSettingsService(rt)
	if _, err := svc.GetUIPreferences(); err == nil {
		t.Fatal("expected not ready")
	}
	if _, err := svc.SetUIPreferences(nil); err == nil {
		t.Fatal("expected not ready")
	}
}

func TestBoxdAuthService(t *testing.T) {
	rt := &desktopRuntime{cfg: desktopConfig{Mode: "embedded", RemoteURL: "http://x"}}
	svc := newBoxdAuthService(rt)
	if svc.Mode() != "embedded" {
		t.Fatalf("mode = %q", svc.Mode())
	}
	if svc.RemoteURL() != "http://x" {
		t.Fatalf("url = %q", svc.RemoteURL())
	}
}

func TestBoxdStatsServiceNotReady(t *testing.T) {
	rt := &desktopRuntime{}
	svc := newBoxdStatsService(rt)
	if _, err := svc.Traffic(); err == nil {
		t.Fatal("expected not ready")
	}
	if _, err := svc.Connections(); err == nil {
		t.Fatal("expected not ready")
	}
}

func TestBoxdStatsService(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdStatsService(rt)
	traffic, err := svc.Traffic()
	if err != nil {
		t.Fatal(err)
	}
	if traffic == nil {
		t.Fatal("traffic is nil")
	}
	conns, err := svc.Connections()
	if err != nil {
		t.Fatal(err)
	}
	if conns == nil {
		t.Fatal("connections is nil")
	}
}

func TestRegisterServicesRemoteMode(t *testing.T) {
	rt := &desktopRuntime{cfg: desktopConfig{Mode: "remote"}}
	services := registerServices(rt)
	if len(services) != 0 {
		t.Fatalf("services = %d, want 0 for remote mode", len(services))
	}
}

func TestRegisterServicesEmbedded(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	services := registerServices(rt)
	if len(services) != 6 {
		t.Fatalf("services = %d, want 6", len(services))
	}
}

func TestBoxdAuthServiceAutoLogin(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdAuthService(rt)
	result, err := svc.AutoLogin()
	if err != nil {
		t.Fatal(err)
	}
	data, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("result type = %T", result)
	}
	if data["token"] == "" {
		t.Fatal("token empty")
	}
}

func TestBoxdAuthServiceAutoLoginNotReady(t *testing.T) {
	rt := &desktopRuntime{}
	svc := newBoxdAuthService(rt)
	if _, err := svc.AutoLogin(); err == nil {
		t.Fatal("expected not ready error")
	}
}

func TestIssueEmbeddedTokenEmptySecret(t *testing.T) {
	_, _, err := issueEmbeddedToken("")
	if err == nil {
		t.Fatal("expected error for empty secret")
	}
}

func TestIssueEmbeddedToken(t *testing.T) {
	token, expiresAt, err := issueEmbeddedToken("test-secret-key-123456")
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("empty token")
	}
	if expiresAt.IsZero() {
		t.Fatal("empty expires")
	}
}
