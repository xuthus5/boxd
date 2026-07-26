package core

import (
	"encoding/json"
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

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
	if len(servers) != 4 {
		t.Fatalf("servers len = %d, want 4", len(servers))
	}
	local := servers[0].(map[string]any)
	if local["type"] != "local" || local["tag"] != "dns-local" {
		t.Fatalf("dns-local = %#v", local)
	}
	direct := servers[1].(map[string]any)
	if direct["type"] != "https" || direct["server"] != "223.5.5.5" {
		t.Fatalf("dns-direct = %#v", direct)
	}
	remote := servers[2].(map[string]any)
	if remote["detour"] != "proxy" {
		t.Fatalf("dns-remote.detour = %#v", remote["detour"])
	}
	if remote["type"] != "https" || remote["server"] != "dns.google" || remote["domain_resolver"] != "dns-direct" {
		t.Fatalf("dns-remote = %#v", remote)
	}
	fakeIP := servers[3].(map[string]any)
	if fakeIP["type"] != "fakeip" || fakeIP["inet4_range"] != "198.18.0.0/15" {
		t.Fatalf("dns-fake = %#v", fakeIP)
	}
	if _, exists := result.DNS["fakeip"]; exists {
		t.Fatalf("legacy dns.fakeip should be absent: %#v", result.DNS["fakeip"])
	}
	if result.DNS["strategy"] != "ipv4_only" {
		t.Fatalf("dns.strategy = %#v", result.DNS["strategy"])
	}
	rules := result.DNS["rules"].([]any)
	if len(rules) != 4 {
		t.Fatalf("rules len = %d, want 4", len(rules))
	}
	for _, item := range rules {
		rule := item.(map[string]any)
		if _, exists := rule["outbound"]; exists {
			t.Fatalf("legacy outbound dns rule should be absent: %#v", rule)
		}
	}
	blockRule := rules[1].(map[string]any)
	if blockRule["action"] != "predefined" || blockRule["rcode"] != "NOERROR" {
		t.Fatalf("dns block rule = %#v", blockRule)
	}
	if result.DNS["final"] != "dns-remote" {
		t.Fatalf("dns.final = %#v", result.DNS["final"])
	}
	if result.DefaultDomainResolver != "dns-direct" {
		t.Fatalf("default domain resolver = %#v", result.DefaultDomainResolver)
	}

	route := cfg["route"].(map[string]any)
	route["default_domain_resolver"] = result.DefaultDomainResolver
	generated := cloneAnyMap(cfg)
	generated["dns"] = result.DNS
	body, err := json.Marshal(generated)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	for _, issue := range AnalyzeConfig(body).Issues {
		switch issue.Code {
		case "invalid_singbox_config",
			"legacy_dns_server",
			"legacy_dns_fakeip",
			"outbound_dns_rule_item",
			"missing_domain_resolver",
			"legacy_domain_strategy":
			t.Fatalf("generated defaults diagnostic = %#v", issue)
		}
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
	remote = servers[2].(map[string]any)
	if remote["detour"] != "direct" {
		t.Fatalf("dns-remote.detour fallback = %#v", remote["detour"])
	}
}

func TestDefaultDNSInstallerUsesRouteAndGeositeFallbacks(t *testing.T) {
	installer := NewDefaultDNSInstaller()
	cfg := map[string]any{
		"outbounds": []any{
			map[string]any{"tag": "direct", "type": "direct"},
			map[string]any{"tag": "secure", "type": "selector", "outbounds": []any{"direct"}},
		},
		"route": map[string]any{
			"final": "secure",
			"rule_set": []any{
				map[string]any{"tag": "geosite-cn"},
				map[string]any{"tag": "geosite-google-play"},
				map[string]any{"tag": "geosite-category-ads-all"},
			},
		},
	}

	result, err := installer.Install(cfg)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	servers := result.DNS["servers"].([]any)
	remote := servers[2].(map[string]any)
	if remote["detour"] != "secure" {
		t.Fatalf("dns-remote.detour = %#v, want secure", remote["detour"])
	}

	rules := result.DNS["rules"].([]any)
	if len(rules) != 4 {
		t.Fatalf("rules len = %d, want 4", len(rules))
	}
	assertDNSRule(t, rules[1], "geosite-category-ads-all", "predefined", "")
	assertDNSRule(t, rules[2], "geosite-cn", "route", "dns-direct")
	assertDNSRule(t, rules[3], "geosite-google-play", "route", "dns-remote")
}

type dnsDetourTestCase struct {
	name         string
	outbounds    []any
	directDetour string
	remoteDetour string
}

func TestDefaultDNSInstallerUsesOnlyExistingDetours(t *testing.T) {
	tests := []dnsDetourTestCase{
		{name: "without outbounds", outbounds: []any{}},
		{
			name:         "proxy without direct",
			outbounds:    []any{map[string]any{"type": "direct", "tag": "proxy"}},
			remoteDetour: "proxy",
		},
		{
			name: "direct and proxy",
			outbounds: []any{
				map[string]any{"type": "direct", "tag": "direct"},
				map[string]any{"type": "direct", "tag": "proxy"},
			},
			directDetour: "direct",
			remoteDetour: "proxy",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			verifyDefaultDNSDetours(t, test)
		})
	}
}

func verifyDefaultDNSDetours(t *testing.T, test dnsDetourTestCase) {
	t.Helper()
	cfg := map[string]any{"outbounds": test.outbounds}
	result, err := NewDefaultDNSInstaller().Install(cfg)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	servers := result.DNS["servers"].([]any)
	assertDNSDetour(t, servers[0], test.directDetour)
	assertDNSDetour(t, servers[1], test.directDetour)
	assertDNSDetour(t, servers[2], test.remoteDetour)

	generated := cloneAnyMap(cfg)
	generated["dns"] = result.DNS
	generated["route"] = map[string]any{"default_domain_resolver": result.DefaultDomainResolver}
	body, err := json.Marshal(generated)
	if err != nil {
		t.Fatal(err)
	}
	for _, issue := range AnalyzeConfig(body).Issues {
		if issue.Severity == model.ConfigDiagnosticSeverityError {
			t.Fatalf("generated defaults diagnostic = %#v", issue)
		}
	}
}

func assertDNSDetour(t *testing.T, value any, expected string) {
	t.Helper()
	server := value.(map[string]any)
	actual, exists := server["detour"].(string)
	if expected == "" && exists {
		t.Fatalf("detour = %q, want omitted", actual)
	}
	if expected != "" && actual != expected {
		t.Fatalf("detour = %q, want %q", actual, expected)
	}
}

func assertDNSRule(t *testing.T, value any, ruleSet, action, server string) {
	t.Helper()
	rule := value.(map[string]any)
	ruleSets := rule["rule_set"].([]string)
	if len(ruleSets) != 1 || ruleSets[0] != ruleSet {
		t.Fatalf("rule_set = %#v, want %q", ruleSets, ruleSet)
	}
	actualServer, _ := rule["server"].(string)
	if rule["action"] != action || actualServer != server {
		t.Fatalf("rule = %#v, want action %q and server %q", rule, action, server)
	}
}
