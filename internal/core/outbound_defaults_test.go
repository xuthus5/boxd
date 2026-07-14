package core

import "testing"

func TestDefaultOutboundsInstallerInstall(t *testing.T) {
	installer := NewDefaultOutboundsInstaller()
	cfg := map[string]any{
		"outbounds": []any{
			map[string]any{"tag": "node-a", "type": "vless"},
			map[string]any{"tag": "node-b", "type": "trojan"},
			map[string]any{"tag": "ssh-out", "type": "ssh"},
		},
	}

	result, err := installer.Install(cfg)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if result.Final != "proxy" {
		t.Fatalf("Final = %q", result.Final)
	}
	if len(result.Installed) == 0 {
		t.Fatal("Installed should not be empty")
	}

	byTag := make(map[string]map[string]any)
	for _, item := range result.Outbounds {
		ob := item.(map[string]any)
		byTag[ob["tag"].(string)] = ob
	}
	for _, tag := range []string{"direct", "bypass", "block", "proxy", "auto", "whitelist", "blacklist"} {
		if _, ok := byTag[tag]; !ok {
			t.Fatalf("missing default outbound %q", tag)
		}
	}
	if byTag["direct"]["routing_mark"] != 128 {
		t.Fatalf("direct.routing_mark = %#v", byTag["direct"]["routing_mark"])
	}
	if byTag["bypass"]["routing_mark"] != 128 {
		t.Fatalf("bypass.routing_mark = %#v", byTag["bypass"]["routing_mark"])
	}
	if _, ok := byTag["dns-out"]; ok {
		t.Fatalf("dns-out should not be installed in sing-box 1.13")
	}

	proxyMembers := byTag["proxy"]["outbounds"].([]string)
	if len(proxyMembers) != 3 {
		t.Fatalf("proxy members = %#v", proxyMembers)
	}
}

// 验证已存在 proxy selector 组保留其 default 与原始成员，避免破坏用户配置导致内核回滚。
func TestDefaultOutboundsInstallerPreserveExistingProxyDefault(t *testing.T) {
	installer := NewDefaultOutboundsInstaller()
	cfg := map[string]any{
		"outbounds": []any{
			map[string]any{"tag": "my-group", "type": "urltest", "outbounds": []string{"node-a"}},
			map[string]any{"tag": "proxy", "type": "selector", "default": "my-group", "outbounds": []string{"my-group", "node-a"}},
			map[string]any{"tag": "node-a", "type": "vless"},
		},
	}

	result, err := installer.Install(cfg)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	byTag := make(map[string]map[string]any)
	for _, item := range result.Outbounds {
		ob := item.(map[string]any)
		byTag[ob["tag"].(string)] = ob
	}
	proxyOb, ok := byTag["proxy"]
	if !ok {
		t.Fatal("missing proxy outbound")
	}
	if proxyOb["default"] != "my-group" {
		t.Fatalf("proxy.default = %#v, want my-group", proxyOb["default"])
	}
	proxyMembers, _ := proxyOb["outbounds"].([]string)
	if len(proxyMembers) != 2 {
		t.Fatalf("proxy members should preserve original, got %#v", proxyOb["outbounds"])
	}
	for _, tag := range []string{"block", "bypass"} {
		if _, ok := byTag[tag]; !ok {
			t.Fatalf("missing builtin %q", tag)
		}
	}
}
