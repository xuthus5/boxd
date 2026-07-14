package core

import "testing"

func TestDefaultRouteInstallerInstall(t *testing.T) {
	installer := NewDefaultRouteInstaller()
	cfg := map[string]any{
		"outbounds": []any{
			map[string]any{"tag": "direct", "type": "direct"},
			map[string]any{"tag": "bypass", "type": "direct"},
			map[string]any{"tag": "block", "type": "block"},
			map[string]any{"tag": "proxy", "type": "selector"},
		},
		"route": map[string]any{
			"rule_set": []any{
				map[string]any{"tag": "loyalsoldier-direct"},
				map[string]any{"tag": "geoip-cn"},
				map[string]any{"tag": "loyalsoldier-proxy"},
				map[string]any{"tag": "loyalsoldier-reject"},
			},
		},
	}

	result, err := installer.Install(cfg)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	// sniff + hijack-dns + ip_is_private + icmp + quic-reject + ads + cn-domain + cn-ip + proxy = 9
	if len(result.Rules) != 9 {
		t.Fatalf("rules len = %d, want 9", len(result.Rules))
	}
	if len(result.Installed) != 9 {
		t.Fatalf("installed len = %d, want 9", len(result.Installed))
	}
	rules := result.Rules
	if act, _ := rules[0].(map[string]any)["action"].(string); act != "sniff" {
		t.Fatalf("first rule = %#v, want sniff", rules[0])
	}
	if act, _ := rules[1].(map[string]any)["action"].(string); act != "hijack-dns" {
		t.Fatalf("second rule = %#v, want hijack-dns", rules[1])
	}
	if got := ruleSignature(rules[7].(map[string]any)); got != "rule_set:geoip-cn->bypass" {
		t.Fatalf("China IP rule = %q, want geoip-cn bypass", got)
	}
}

func TestDefaultRouteInstallerGeositeFallback(t *testing.T) {
	installer := NewDefaultRouteInstaller()
	// 仅使用 SagerNet 规则集，验证回退逻辑生效。
	cfg := map[string]any{
		"outbounds": []any{
			map[string]any{"tag": "direct", "type": "direct"},
			map[string]any{"tag": "block", "type": "block"},
			map[string]any{"tag": "proxy", "type": "selector"},
		},
		"route": map[string]any{
			"rule_set": []any{
				map[string]any{"tag": "geosite-cn"},
				map[string]any{"tag": "geoip-cn"},
				map[string]any{"tag": "geosite-category-ads-all"},
				map[string]any{"tag": "geosite-google-play"},
			},
		},
	}

	result, err := installer.Install(cfg)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	// sniff + hijack-dns + ip_is_private + icmp + quic-reject + ads + cn-domain + cn-ip + proxy = 9
	if len(result.Rules) != 9 {
		t.Fatalf("rules len = %d, want 9", len(result.Rules))
	}

	found := func(tag string) bool {
		for _, item := range result.Rules {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if sets, ok := m["rule_set"].([]string); ok && len(sets) > 0 && sets[0] == tag {
				return true
			}
			if sets, ok := m["rule_set"].([]any); ok && len(sets) > 0 {
				if s, _ := sets[0].(string); s == tag {
					return true
				}
			}
		}
		return false
	}
	for _, tag := range []string{"geosite-cn", "geoip-cn", "geosite-category-ads-all", "geosite-google-play"} {
		if !found(tag) {
			t.Errorf("missing fallback rule_set %s", tag)
		}
	}
}

func TestDefaultRouteInstallerMinimalOutbounds(t *testing.T) {
	installer := NewDefaultRouteInstaller()
	// 仅有 direct，无 block/proxy，验证规则数量精简。
	cfg := map[string]any{
		"outbounds": []any{
			map[string]any{"tag": "direct", "type": "direct"},
		},
	}

	result, err := installer.Install(cfg)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	// sniff + hijack-dns + ip_is_private + icmp = 4
	if len(result.Rules) != 4 {
		t.Fatalf("rules len = %d, want 4", len(result.Rules))
	}
}

func TestDefaultRouteInstallerDedup(t *testing.T) {
	installer := NewDefaultRouteInstaller()
	cfg := map[string]any{
		"outbounds": []any{
			map[string]any{"tag": "direct", "type": "direct"},
			map[string]any{"tag": "block", "type": "block"},
			map[string]any{"tag": "proxy", "type": "selector"},
		},
		"route": map[string]any{
			"rule_set": []any{
				map[string]any{"tag": "loyalsoldier-direct"},
				map[string]any{"tag": "loyalsoldier-proxy"},
				map[string]any{"tag": "loyalsoldier-reject"},
			},
			// 已存在 sniff 规则，验证去重不重复安装。
			"rules": []any{
				map[string]any{"action": "sniff"},
			},
		},
	}

	result, err := installer.Install(cfg)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if len(result.Installed) != 7 {
		t.Fatalf("installed len = %d, want 7 (sniff dedup)", len(result.Installed))
	}
	if len(result.Rules) != 8 {
		t.Fatalf("rules len = %d, want 8", len(result.Rules))
	}
}

// --- 以下为补充测试 ---

func TestRuleSignatureNetwork(t *testing.T) {
	rule := map[string]any{"network": "udp", "port": 443, "action": "reject"}
	want := "network:udp->443->reject->"
	if got := ruleSignature(rule); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRuleSignatureNetworkList(t *testing.T) {
	rule := map[string]any{"network": []any{"icmp"}, "outbound": "direct"}
	want := "network:icmp->->->direct"
	if got := ruleSignature(rule); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRuleSignatureProtocol(t *testing.T) {
	rule := map[string]any{"protocol": "dns", "outbound": "dns-out", "action": "sniff"}
	got := ruleSignature(rule)
	want := "protocol:dns->dns-out->sniff"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRuleSignatureIPPrivate(t *testing.T) {
	rule := map[string]any{"ip_is_private": true, "outbound": "direct"}
	got := ruleSignature(rule)
	want := "ip_is_private->direct"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRuleSignatureRuleSetStringSlice(t *testing.T) {
	rule := map[string]any{"rule_set": []string{"geosite-cn"}, "outbound": "direct"}
	got := ruleSignature(rule)
	want := "rule_set:geosite-cn->direct"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRuleSignatureRuleSetAnySlice(t *testing.T) {
	rule := map[string]any{"rule_set": []any{"geosite-cn"}, "outbound": "proxy"}
	got := ruleSignature(rule)
	want := "rule_set:geosite-cn->proxy"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRuleSignatureFallback(t *testing.T) {
	rule := map[string]any{"outbound": "direct"}
	got := ruleSignature(rule)
	want := "direct"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStringValue(t *testing.T) {
	if got := stringValue("hello"); got != "hello" {
		t.Errorf("got %q, want 'hello'", got)
	}
	if got := stringValue(123); got != "" {
		t.Errorf("got %q, want ''", got)
	}
	if got := stringValue(nil); got != "" {
		t.Errorf("got %q, want ''", got)
	}
}

func TestPortValue(t *testing.T) {
	if got := portValue("443"); got != "443" {
		t.Errorf("string got %q, want 443", got)
	}
	if got := portValue(443); got != "443" {
		t.Errorf("int got %q, want 443", got)
	}
	if got := portValue(int64(443)); got != "443" {
		t.Errorf("int64 got %q, want 443", got)
	}
	if got := portValue(float64(443)); got != "443" {
		t.Errorf("float64 got %q, want 443", got)
	}
	if got := portValue(nil); got != "" {
		t.Errorf("nil got %q, want ''", got)
	}
	if got := portValue(false); got != "" {
		t.Errorf("bool got %q, want ''", got)
	}
}
