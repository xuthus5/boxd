package service

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
)

// TestInstallDefaultDNSFirstInstallSkipsEmptyDirectDetour 覆盖首次安装场景：
// 最小配置中的 direct 出站为空，DNS 服务器不应生成指向空 direct 的 detour，
// 否则 sing-box 内核会以 "detour to an empty direct outbound" 拒绝启动导致回滚。
func TestInstallDefaultDNSFirstInstallSkipsEmptyDirectDetour(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if created, err := core.EnsureConfigFile(path); err != nil || !created {
		t.Fatalf("EnsureConfigFile() created=%v err=%v", created, err)
	}

	svc := newTestService(t)
	cfg := newConfig(path, nil, ConfigInstaller{
		DNSInstaller: core.NewDefaultDNSInstaller(),
		ApplyHistory: svc.Deps.ApplyHistory,
	})

	result, err := cfg.InstallDefaultDNS(context.Background())
	if err != nil {
		t.Fatalf("InstallDefaultDNS() error = %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("status = %q", result.Status)
	}

	got, err := cfg.GetConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	parsed, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("config type = %T", got)
	}
	dns, _ := parsed["dns"].(map[string]any)
	if dns == nil {
		t.Fatal("dns section missing")
	}
	rules, _ := dns["rules"].([]any)
	if len(rules) == 0 {
		t.Fatal("dns.rules is empty after install")
	}
	servers, _ := dns["servers"].([]any)
	if len(servers) == 0 {
		t.Fatal("dns.servers is empty after install")
	}
	for _, item := range servers {
		server, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if detour, _ := server["detour"].(string); detour != "" {
			t.Fatalf("unexpected detour %q with empty direct outbound", detour)
		}
	}
}
