package core

import "testing"

func TestDefaultDNSInstallerInstall(t *testing.T) {
	installer := NewDefaultDNSInstaller()
	cfg := map[string]any{
		"outbounds": []any{
			map[string]any{"tag": "proxy", "type": "selector"},
			map[string]any{"tag": "direct", "type": "direct"},
		},
		"route": map[string]any{
			"rule_set": []any{
				map[string]any{"tag": "loyalsoldier-direct"},
				map[string]any{"tag": "loyalsoldier-proxy"},
				map[string]any{"tag": "loyalsoldier-reject"},
			},
		},
	}

	result, err := installer.Install(cfg)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	servers := result.DNS["servers"].([]any)
	if len(servers) != 5 {
		t.Fatalf("servers len = %d", len(servers))
	}
	remote := servers[3].(map[string]any)
	if remote["detour"] != "proxy" {
		t.Fatalf("dns-remote.detour = %#v", remote["detour"])
	}
	rules := result.DNS["rules"].([]any)
	if len(rules) < 4 {
		t.Fatalf("rules len = %d", len(rules))
	}
	if result.DNS["final"] != "dns-remote" {
		t.Fatalf("dns.final = %#v", result.DNS["final"])
	}

	cfg = map[string]any{
		"outbounds": []any{
			map[string]any{"tag": "direct", "type": "direct"},
		},
	}
	result, err = installer.Install(cfg)
	if err != nil {
		t.Fatalf("Install() with direct fallback error = %v", err)
	}
	servers = result.DNS["servers"].([]any)
	remote = servers[3].(map[string]any)
	if remote["detour"] != "direct" {
		t.Fatalf("dns-remote.detour fallback = %#v", remote["detour"])
	}
}
