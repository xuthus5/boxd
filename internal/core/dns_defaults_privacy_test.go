package core

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDNSDefaultsKeepRemoteQueriesOnProxy(t *testing.T) {
	result, err := NewDefaultDNSInstaller().Install(policyDefaultsFixture())
	if err != nil {
		t.Fatal(err)
	}
	if result.DNS["final"] != "dns-remote" {
		t.Fatalf("unmatched DNS queries use %v, want dns-remote", result.DNS["final"])
	}
	servers := result.DNS["servers"].([]any)
	if len(servers) != 2 {
		t.Fatalf("DNS servers = %d, want only the two used DoH servers", len(servers))
	}
	remote := servers[1].(map[string]any)
	want := map[string]any{
		"type": "https", "tag": "dns-remote", "server": "8.8.8.8", "detour": "proxy",
		"tls": map[string]any{"server_name": "dns.google"},
	}
	if !reflect.DeepEqual(want, remote) {
		t.Fatalf("remote DNS = %#v, want %#v", remote, want)
	}
}

func TestDNSDefaultsPreferProxyOnOverlappingLists(t *testing.T) {
	result, err := NewDefaultDNSInstaller().Install(policyDefaultsFixture())
	if err != nil {
		t.Fatal(err)
	}
	rules := result.DNS["rules"].([]any)
	proxy := policyRuleIndex(t, rules, routedDNSRule("loyalsoldier-proxy", "dns-remote"))
	direct := policyRuleIndex(t, rules, routedDNSRule("loyalsoldier-direct", "dns-direct"))
	if proxy < 0 || direct < 0 || proxy >= direct {
		t.Fatalf("proxy DNS rule index = %d, direct index = %d; proxy must win overlaps", proxy, direct)
	}
}

func TestDNSDefaultsRejectUnsafeDetours(t *testing.T) {
	cases := []struct {
		name      string
		outbounds []any
	}{
		{name: "missing proxy"},
		{name: "direct tagged proxy", outbounds: []any{map[string]any{"type": "direct", "tag": "proxy"}}},
		{name: "selector direct fallback", outbounds: []any{
			map[string]any{"type": "direct", "tag": "direct"},
			map[string]any{"type": "selector", "tag": "proxy", "outbounds": []string{"direct"}},
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			cfg := map[string]any{"outbounds": test.outbounds}
			if _, err := NewDefaultDNSInstaller().Install(cfg); err == nil {
				t.Fatal("unsafe default DNS silently falls back to direct")
			}
		})
	}
}

func policyDefaultsFixture() map[string]any {
	return map[string]any{
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
			map[string]any{"type": "block", "tag": "block"},
			map[string]any{"type": "selector", "tag": "proxy", "outbounds": []string{"block"}},
		},
		"route": map[string]any{
			"final": "proxy",
			"rule_set": []any{
				policyInlineSet("loyalsoldier-direct", "direct.test", "overlap.test"),
				policyInlineSet("loyalsoldier-proxy", "proxy.test", "overlap.test"),
				policyInlineSet("loyalsoldier-reject", "ads.test", "blocked.test"),
				map[string]any{"type": "inline", "tag": "geoip-cn", "rules": []any{
					map[string]any{"ip_cidr": []string{"203.0.113.0/24"}},
				}},
			},
		},
	}
}

func policyInlineSet(tag string, domains ...string) map[string]any {
	return map[string]any{
		"type": "inline", "tag": tag,
		"rules": []any{map[string]any{"domain": domains}},
	}
}

func policyRuleIndex(t *testing.T, rules []any, want map[string]any) int {
	t.Helper()
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	for index, value := range rules {
		data, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		if string(data) == string(wantJSON) {
			return index
		}
	}
	return -1
}
